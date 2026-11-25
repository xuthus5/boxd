package httpapi

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/webauth"
)

const (
	sessionActor        = "web-admin"
	sessionSource       = "http"
	loginFailureWindow  = 5 * time.Minute
	loginFailureLimit   = 5
	maxPasswordBodySize = 4096
)

type authStatusResponse struct {
	webauth.Status
	Authenticated bool       `json:"authenticated"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

type passwordRequest struct {
	Password string `json:"password"`
}

type sessionResponse struct {
	Authenticated bool      `json:"authenticated"`
	Token         string    `json:"token"`
	ExpiresAt     time.Time `json:"expires_at"`
	Actor         string    `json:"actor"`
}

type passwordRotationResponse struct {
	Authenticated       bool      `json:"authenticated"`
	Password            string    `json:"password"`
	Token               string    `json:"token"`
	ExpiresAt           time.Time `json:"expires_at"`
	CredentialUpdatedAt time.Time `json:"credential_updated_at"`
	Actor               string    `json:"actor"`
}

type authContextKey struct{}

type authContext struct {
	Token     string
	ViaCookie bool
}

type loginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{failures: make(map[string][]time.Time)}
}

func (l *loginLimiter) blocked(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	attempts := pruneAttempts(l.failures[key], now)
	l.failures[key] = attempts
	return len(attempts) >= loginFailureLimit
}

func (l *loginLimiter) fail(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[key] = append(pruneAttempts(l.failures[key], now), now)
}

func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	delete(l.failures, key)
	l.mu.Unlock()
}

func pruneAttempts(attempts []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-loginFailureWindow)
	result := attempts[:0]
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			result = append(result, attempt)
		}
	}
	return result
}

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/auth/status", s.handleAuthStatus)
	mux.HandleFunc("POST /api/v1/auth/initialize", s.handleInitialize)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/rotate-password", s.handleRotatePassword)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
}

func (s *Server) apiMiddleware(api http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicAPIPath(r) {
			api.ServeHTTP(w, r)
			return
		}
		auth, ok := s.authenticate(r)
		if !ok {
			writeError(w, r, newAPIError(http.StatusUnauthorized, "kernel_unauthorized", "http.authenticate", "authentication failed", "sign in with the local administrator password"))
			return
		}
		if auth.ViaCookie && !safeMethod(r.Method) && !sameOriginRequest(r) {
			writeError(w, r, newAPIError(http.StatusForbidden, "request_forbidden", "http.origin", "cross-origin request rejected", "reload the console from the same address"))
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, auth)
		ctx = control.WithAuditIdentity(ctx, sessionActor, sessionSource)
		api.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicAPIPath(r *http.Request) bool {
	switch r.URL.Path {
	case "/api/v1/health", "/api/v1/auth/status", "/api/v1/auth/initialize", "/api/v1/auth/login":
		return true
	default:
		return false
	}
}

func safeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func sameOriginRequest(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"service":    "boxd-web",
		"started_at": s.startedAt,
		"version":    "v2",
	})
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.auth.Status(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	response := authStatusResponse{Status: status}
	if auth, ok := s.authenticate(r); ok {
		if session, validateErr := s.auth.ValidateSession(r.Context(), auth.Token); validateErr == nil {
			response.Authenticated = true
			response.ExpiresAt = &session.ExpiresAt
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleInitialize(w http.ResponseWriter, r *http.Request) {
	var request passwordRequest
	if !decodePasswordRequest(w, r, &request) {
		return
	}
	key := clientKey(r)
	if s.limiter.blocked(key, time.Now()) {
		writeError(w, r, newAPIError(http.StatusTooManyRequests, "rate_limited", "security.initialize", "too many authentication attempts", "wait a few minutes and try again"))
		return
	}
	ctx := control.WithAuditIdentity(r.Context(), sessionActor, sessionSource)
	session, err := s.auth.Initialize(ctx, request.Password)
	if err != nil {
		s.limiter.fail(key, time.Now())
		s.writeServiceError(w, r, err)
		return
	}
	s.limiter.reset(key)
	s.setSessionCookie(w, r, session)
	s.logs.add("info", "security", "administrator password initialized", nil)
	writeJSON(w, http.StatusCreated, sessionResponse{Authenticated: true, Token: session.Token, ExpiresAt: session.ExpiresAt, Actor: sessionActor})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request passwordRequest
	if !decodePasswordRequest(w, r, &request) {
		return
	}
	key := clientKey(r)
	if s.limiter.blocked(key, time.Now()) {
		writeError(w, r, newAPIError(http.StatusTooManyRequests, "rate_limited", "security.login", "too many authentication attempts", "wait a few minutes and try again"))
		return
	}
	ctx := control.WithAuditIdentity(r.Context(), sessionActor, sessionSource)
	session, err := s.auth.Login(ctx, request.Password)
	if err != nil {
		s.limiter.fail(key, time.Now())
		s.writeServiceError(w, r, err)
		return
	}
	s.limiter.reset(key)
	s.setSessionCookie(w, r, session)
	s.logs.add("info", "security", "administrator signed in", nil)
	writeJSON(w, http.StatusOK, sessionResponse{Authenticated: true, Token: session.Token, ExpiresAt: session.ExpiresAt, Actor: sessionActor})
}

func (s *Server) handleRotatePassword(w http.ResponseWriter, r *http.Request) {
	ctx := control.WithAuditIdentity(r.Context(), sessionActor, sessionSource)
	rotation, err := s.auth.RotatePassword(ctx)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	status, err := s.auth.Status(ctx)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.setSessionCookie(w, r, rotation.Session)
	s.logs.add("info", "security", "administrator password rotated; all prior sessions revoked", nil)
	writeJSON(w, http.StatusOK, passwordRotationResponse{
		Authenticated:       true,
		Password:            rotation.Password,
		Token:               rotation.Session.Token,
		ExpiresAt:           rotation.Session.ExpiresAt,
		CredentialUpdatedAt: status.CredentialUpdatedAt,
		Actor:               sessionActor,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	auth, _ := s.authenticate(r)
	ctx := control.WithAuditIdentity(r.Context(), sessionActor, sessionSource)
	if auth.Token != "" {
		if err := s.auth.Logout(ctx, auth.Token); err != nil {
			s.writeServiceError(w, r, err)
			return
		}
	}
	s.clearSessionCookie(w, r)
	s.logs.add("info", "security", "administrator signed out", nil)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func decodePasswordRequest(w http.ResponseWriter, r *http.Request, request *passwordRequest) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxPasswordBodySize)
	if err := decodeJSON(w, r, request); err != nil {
		writeError(w, r, err.(*apiError))
		return false
	}
	if len(request.Password) == 0 {
		writeError(w, r, newAPIError(http.StatusBadRequest, "invalid_password", "security.password", "password is required", "enter the administrator password"))
		return false
	}
	return true
}

func (s *Server) authenticate(r *http.Request) (authContext, bool) {
	token, viaCookie := s.sessionToken(r)
	if token == "" {
		return authContext{}, false
	}
	if _, err := s.auth.ValidateSession(r.Context(), token); err != nil {
		return authContext{}, false
	}
	return authContext{Token: token, ViaCookie: viaCookie}, true
}

func (s *Server) sessionToken(r *http.Request) (string, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header != "" {
		scheme, token, found := strings.Cut(header, " ")
		if found && strings.EqualFold(scheme, "Bearer") && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token), false
		}
	}
	if cookie, err := r.Cookie(s.cookieName); err == nil {
		return strings.TrimSpace(cookie.Value), true
	}
	return "", false
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, session webauth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
	})
}

func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
