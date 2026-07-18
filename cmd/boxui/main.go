package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/api"
	"github.com/xuthus5/boxui/internal/config"
	"github.com/xuthus5/boxui/internal/core"
)

func main() {
	cfg := config.Parse()
	if err := execute(cfg, os.Stdout); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func execute(cfg *config.Config, stdout io.Writer) error {
	if cfg.ShowVersion {
		return writeVersion(stdout)
	}
	return run(cfg)
}

func writeVersion(writer io.Writer) error {
	_, err := fmt.Fprintln(writer, core.Version)
	return err
}

func run(cfg *config.Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}
	if cfg.RestorePath != "" {
		if err := core.RestoreBackup(cfg.RestorePath, cfg.DataDir, cfg.ConfigPath); err != nil {
			return fmt.Errorf("restore backup: %w", err)
		}
		slog.Info("backup restored", "path", cfg.RestorePath)
		return nil
	}

	db, err := openDatabase(cfg.DataDir)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database", "err", err)
		}
	}()

	settingsManager := core.NewSettingsManager(db)
	defaultPassword, err := settingsManager.EnsureAdminCredential(cfg.Username, cfg.Password)
	if err != nil {
		return fmt.Errorf("init administrator credential: %w", err)
	}
	if defaultPassword {
		slog.Warn("default administrator password is active; change it in settings")
	}
	secret, generated, err := settingsManager.EnsureJWTSecret()
	if err != nil {
		return fmt.Errorf("init jwt secret: %w", err)
	}
	if generated {
		slog.Info("jwt secret auto-generated and persisted to database")
	}
	_ = secret
	if cfg.BackupPath != "" {
		if err := core.CreateBackup(db, cfg.ConfigPath, cfg.BackupPath, core.Version); err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
		slog.Info("backup created", "path", cfg.BackupPath)
		return nil
	}

	handler, _, appLogWriter := newHandler(cfg, db, settingsManager)

	logLevel := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(core.NewAppLogHandler(os.Stderr, appLogWriter, logLevel)))

	server := makeHTTPServer(cfg.Listen, handler)
	return serveUntilSignal(server, cfg, makeSignalChannel())
}

func validateConfig(cfg *config.Config) error {
	if (cfg.TLSCert == "") != (cfg.TLSKey == "") {
		return fmt.Errorf("tls certificate and key must be configured together")
	}
	if cfg.TLSCert != "" {
		if err := validateRegularFile(cfg.TLSCert, "tls certificate"); err != nil {
			return err
		}
		if err := validateRegularFile(cfg.TLSKey, "tls key"); err != nil {
			return err
		}
	}
	// JWT 密钥不再来自环境变量：启动时从数据库加载，若无则随机生成并持久化。
	return nil
}

func validateRegularFile(path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s is not accessible: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file", name)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s is not readable: %w", name, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s after validation: %w", name, err)
	}
	return nil
}

func openDatabase(dataDir string) (*bbolt.DB, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "boxui.db")
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

func newHandler(cfg *config.Config, db *bbolt.DB, settingsManager *core.SettingsManager) (http.Handler, *core.LogWriter, *core.LogWriter) {
	kernelLogWriter := core.NewLogWriter(200)
	appLogWriter := core.NewLogWriter(200)
	instance := core.NewSBInstance(cfg.ConfigPath, kernelLogWriter)
	subscriptionManager := core.NewSubscriptionManager(db, cfg.DataDir)
	nodeManager := core.NewNodeManager(db)
	routeMetadataManager := core.NewRouteRuleMetadataManager(db)

	authHandler := api.NewAuthHandler(cfg.Username, cfg.Password, settingsManager)
	ruleSetInstaller := core.NewLoyalsoldierRuleSetInstaller(cfg.DataDir)
	ruleSetUpdater := core.NewRuleSetUpdater(cfg.ConfigPath, cfg.DataDir, ruleSetInstaller, instance.Stop, instance.Start)
	ruleSetAutoUpdater := core.NewRuleSetAutoUpdater(settingsManager, ruleSetUpdater)
	ruleSetAutoUpdater.Start()
	configHandler := api.NewConfigHandler(
		cfg.ConfigPath,
		instance,
		ruleSetInstaller,
		core.NewDefaultOutboundsInstaller(),
		core.NewDefaultRouteInstaller(),
		core.NewDefaultDNSInstaller(),
		routeMetadataManager,
	)
	ruleSetHandler := api.NewRuleSetHandler(ruleSetUpdater, settingsManager)
	serviceHandler := api.NewServiceHandler(instance)
	statsHandler := api.NewStatsHandler(kernelLogWriter, appLogWriter, instance)
	importHandler := api.NewImportHandler(nodeManager, subscriptionManager, cfg.ConfigPath)
	subscriptionHandler := api.NewSubscriptionHandler(subscriptionManager, nodeManager, cfg.ConfigPath)
	nodesHandler := api.NewNodesHandler(nodeManager, subscriptionManager, cfg.ConfigPath)
	settingsHandler := api.NewSettingsHandler(settingsManager, cfg.Username)
	testHandler := api.NewTestHandler(func() string {
		u := settingsManager.Get("url_test")
		if u == "" {
			u = "https://cp.cloudflare.com/"
		}
		return u
	}, nodeManager, instance)
	networkHandler := api.NewNetworkHandler()
	runtimeHandler := api.NewRuntimeHandler(instance)
	kernelHandler := api.NewKernelHandler(core.Version)

	router := api.NewRouter(
		staticFS,
		authHandler,
		configHandler,
		serviceHandler,
		statsHandler,
		importHandler,
		subscriptionHandler,
		nodesHandler,
		testHandler,
		settingsHandler,
		networkHandler,
		kernelHandler,
		runtimeHandler,
		ruleSetHandler,
		settingsManager,
		cfg.CORSAllowedOrigins,
		instance,
		func() error { return checkReadiness(db, cfg.ConfigPath) },
	)

	if settingsManager.Get("kernel_autostart") == "true" {
		if err := instance.Start(); err != nil {
			slog.Error("kernel autostart failed", "err", err)
		} else {
			slog.Info("kernel autostarted")
		}
	}

	return router, kernelLogWriter, appLogWriter
}

func checkReadiness(db *bbolt.DB, configPath string) error {
	if err := db.View(func(tx *bbolt.Tx) error {
		if tx.Bucket([]byte("settings")) == nil {
			return fmt.Errorf("settings bucket is unavailable")
		}
		return nil
	}); err != nil {
		return fmt.Errorf("database is unavailable: %w", err)
	}
	parent := filepath.Dir(configPath)
	info, err := os.Stat(parent)
	if err != nil {
		return fmt.Errorf("config directory is unavailable: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("config parent is not a directory")
	}
	return nil
}

type server interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	Shutdown(ctx context.Context) error
}

func newHTTPServer(addr string, handler http.Handler) server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

var makeHTTPServer = newHTTPServer

var makeSignalChannel = signalChannel

func signalChannel() chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}

func serveUntilSignal(server server, cfg *config.Config, quit <-chan os.Signal) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("boxui listening", "addr", cfg.Listen, "tls", cfg.TLSCert != "")
		var err error
		if cfg.TLSCert != "" && cfg.TLSKey != "" {
			err = server.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
	}

	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}
