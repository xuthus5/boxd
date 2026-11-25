package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/gofrs/flock"
	desktop "github.com/xuthus5/boxd/internal/adapters/inbound/mygo"
	"github.com/xuthus5/boxd/internal/bootstrap"
	"github.com/xuthus5/boxd/internal/driver/singbox"
	"github.com/xuthus5/boxd/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	showVersion := flag.Bool("version", false, "print build version and exit")
	dataDir := flag.String("data-dir", "", "desktop application data directory")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return nil
	}

	resolvedDir, err := resolveDataDir(*dataDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(resolvedDir, 0o700); err != nil {
		return fmt.Errorf("create desktop data directory: %w", err)
	}

	mygo.App.SetName("BOXD")
	mygo.App.SetVersion(version.Version)
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}

	lock := flock.New(filepath.Join(resolvedDir, "boxd-desktop.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("lock desktop data directory: %w", err)
	}
	if !locked {
		return fmt.Errorf("另一个 BOXD 桌面实例正在使用数据目录 %s", resolvedDir)
	}
	defer func() {
		_ = lock.Unlock()
		_ = lock.Close()
	}()

	logFile, logger, err := openLogger(resolvedDir)
	if err != nil {
		return err
	}
	defer logFile.Close()

	logger.Info("desktop build resolved",
		slog.String("version", version.Version),
		slog.String("commit", version.Commit),
		slog.String("built_at", version.Date),
	)

	preferences, err := loadDesktopPreferences(resolvedDir)
	if err != nil {
		logger.Warn("desktop preferences could not be read; using defaults", slog.Any("error", err))
	}

	loader := desktop.Loader(func(ctx context.Context) (*desktop.Runtime, error) {
		app, err := bootstrap.New(ctx, bootstrap.Options{
			DataDir: resolvedDir,
			Factory: singbox.NewFactory(),
			Logger:  logger,
		})
		if err != nil {
			return nil, err
		}
		return &desktop.Runtime{
			Control:      app.Control,
			Close:        app.Close,
			DataDir:      resolvedDir,
			DatabasePath: app.DatabasePath(),
			Version:      version.Version,
			Commit:       version.Commit,
			BuiltAt:      version.Date,
		}, nil
	})

	host := desktop.NewHost(loader, desktop.HostOptions{
		Version: version.Version,
		Commit:  version.Commit,
		BuiltAt: version.Date,
		Dark:    preferences.isDark(),
		OnThemeChange: func(dark bool) error {
			return saveDesktopPreferences(resolvedDir, desktopPreferences{Theme: preferenceTheme(dark)})
		},
	})
	mygo.App.OnSecondInstance(func([]string, string) { host.Show() })
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { host.Close() })
	mygo.App.OnQuit(host.Close)
	mygo.App.WhenReady(host.Open)

	if err := mygo.App.Run(); err != nil {
		return fmt.Errorf("run BOXD desktop: %w", err)
	}
	host.Close()
	logger.Info("desktop shutdown complete")
	return nil
}

func resolveDataDir(configured string) (string, error) {
	if configured != "" {
		return filepath.Abs(configured)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(base, "boxd-desktop"), nil
}

func openLogger(dataDir string) (*os.File, *slog.Logger, error) {
	path := filepath.Join(dataDir, "desktop.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open desktop log: %w", err)
	}
	writer := io.MultiWriter(os.Stderr, file)
	return file, slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo})), nil
}
