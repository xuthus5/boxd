// Package httpapi exposes the control-plane application over a same-origin HTTP
// API and serves the embedded browser console. Control-plane operations delegate
// to control.Application; browser authentication is a Web-ingress concern backed
// by internal/webauth and is never exposed through the control facade.
package httpapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/webauth"
	"github.com/xuthus5/boxd/kernel"
)

const (
	DefaultAddress   = "[::]:7662"
	sessionCookieKey = "boxd_session"
)

// Options configures the HTTP inbound adapter.
type Options struct {
	Address        string
	Logger         *slog.Logger
	AuthRepository webauth.Repository
	TLSCertFile    string
	TLSKeyFile     string
}

// Server owns the HTTP listener and all browser-facing adapters.
type Server struct {
	app         control.Application
	auth        *webauth.Service
	logger      *slog.Logger
	address     string
	cookieName  string
	limiter     *loginLimiter
	tlsCertFile string
	tlsKeyFile  string
	tlsConfig   *tls.Config
	startedAt   time.Time
	logs        *logHub
	observation *liveObservation
	handler     http.Handler
}

// New constructs the HTTP adapter. The returned server is not listening until
// Serve is called.
func New(app control.Application, options Options) (*Server, error) {
	if app == nil {
		return nil, errors.New("control application is required")
	}
	if options.AuthRepository == nil {
		return nil, errors.New("web authentication repository is required")
	}
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	address := strings.TrimSpace(options.Address)
	if address == "" {
		address = DefaultAddress
	}
	certFile := strings.TrimSpace(options.TLSCertFile)
	keyFile := strings.TrimSpace(options.TLSKeyFile)
	if (certFile == "") != (keyFile == "") {
		return nil, errors.New("both TLS certificate and key files are required")
	}
	var tlsConfig *tls.Config
	if certFile != "" {
		if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
			return nil, fmt.Errorf("load TLS certificate: %w", err)
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	server := &Server{
		app:         app,
		auth:        webauth.New(options.AuthRepository),
		logger:      logger,
		address:     address,
		cookieName:  sessionCookieKey,
		limiter:     newLoginLimiter(),
		tlsCertFile: certFile,
		tlsKeyFile:  keyFile,
		tlsConfig:   tlsConfig,
		startedAt:   time.Now().UTC(),
		logs:        newLogHub(500),
		observation: newLiveObservation(),
	}
	server.handler = server.routes()
	return server, nil
}

// Handler returns the root HTTP handler. It is exposed for integration tests
// and for embedding the UI in another local server.
func (s *Server) Handler() http.Handler { return s.handler }

// Address returns the configured listen address.
func (s *Server) Address() string { return s.address }

// Serve listens until the context is canceled or the server fails. A canceled
// context performs a bounded graceful shutdown.
func (s *Server) Serve(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.address, err)
	}
	return s.ServeListener(ctx, listener)
}

// ServeListener is the testable form of Serve.
func (s *Server) ServeListener(ctx context.Context, listener net.Listener) error {
	samplerCtx, stopSampler := context.WithCancel(ctx)
	defer stopSampler()
	go s.runObservationSampler(samplerCtx)
	httpServer := &http.Server{
		Handler:           s.handler,
		TLSConfig:         s.tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
			s.logger.Warn("http server shutdown incomplete", slog.Any("error", shutdownErr))
		}
	}()
	scheme := "http"
	if s.tlsConfig != nil {
		scheme = "https"
	}
	s.logger.Info("web console listening", slog.String("address", listener.Addr().String()), slog.String("scheme", scheme))
	var err error
	if s.tlsConfig != nil {
		err = httpServer.ServeTLS(listener, s.tlsCertFile, s.tlsKeyFile)
	} else {
		err = httpServer.Serve(listener)
	}
	if errors.Is(err, http.ErrServerClosed) {
		<-shutdownDone
		return nil
	}
	return err
}

func (s *Server) runObservationSampler(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.sampleObservation(ctx, now.UTC())
		}
	}
}

func (s *Server) sampleObservation(parent context.Context, now time.Time) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	status, err := s.app.Kernel().Status(ctx)
	if err != nil {
		return
	}
	if status.State != kernel.StateRunning {
		s.observation.Update(status, nil, nil, control.RouteProfile{}, nil, nil, nil, now)
		return
	}
	connections, err := s.app.Observability().Connections(ctx)
	if err != nil && !isNotRunning(err) {
		return
	}
	closedConnections, err := s.app.Observability().ClosedConnections(ctx)
	if err != nil && !isNotRunning(err) {
		return
	}
	inbounds, err := s.app.Inbounds().List(ctx)
	if err != nil {
		return
	}
	catalog, err := s.app.Egress().Catalog(ctx)
	if err != nil {
		return
	}
	routes, err := s.app.Routes().Get(ctx)
	if err != nil {
		return
	}
	s.observation.Update(status, connections, closedConnections, routes, inbounds, catalog.Nodes, catalog.Chains, now)
}

func (s *Server) routes() http.Handler {
	api := http.NewServeMux()
	s.registerAuthRoutes(api)
	s.registerSnapshotRoutes(api)
	s.registerKernelRoutes(api)
	s.registerInboundRoutes(api)
	s.registerEgressRoutes(api)
	s.registerRouteRoutes(api)
	s.registerSubscriptionRoutes(api)
	s.registerRuleSetRoutes(api)
	s.registerDNSRoutes(api)
	s.registerSettingsRoutes(api)
	s.registerObservabilityRoutes(api)
	s.registerEventRoutes(api)
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, newAPIError(http.StatusNotFound, "not_found", "http.router", "API endpoint was not found", ""))
	})

	root := http.NewServeMux()
	root.Handle("/api/", s.apiMiddleware(api))
	root.Handle("/", s.staticHandler())
	return s.recoverMiddleware(s.securityHeaders(s.requestLogMiddleware(root)))
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += int64(n)
	return n, err
}

func (w *responseRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *responseRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (s *Server) requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		level := "info"
		if status >= 500 {
			level = "error"
		} else if status >= 400 {
			level = "warn"
		}
		source := "http"
		if strings.HasPrefix(r.URL.Path, "/api/") {
			source = "api"
		}
		s.logs.add(level, source, fmt.Sprintf("%s %s %d %s", r.Method, r.URL.Path, status, time.Since(startedAt).Round(time.Millisecond)), nil)
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logs.add("error", "http", "request handler panicked", map[string]any{"path": r.URL.Path})
				s.logger.Error("http handler panic", slog.Any("panic", recovered), slog.String("path", r.URL.Path))
				writeError(w, r, newAPIError(http.StatusInternalServerError, "internal", "http.recover", "internal server error", "retry the request"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
