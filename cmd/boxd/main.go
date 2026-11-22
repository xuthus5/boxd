package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/xuthus5/boxd/internal/bootstrap"
	"github.com/xuthus5/boxd/internal/driver/singbox"
	"github.com/xuthus5/boxd/kernel"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dataDir := flag.String("data-dir", "", "application data directory")
	startKernel := flag.Bool("start", false, "start the kernel after initialization")
	flag.Parse()

	resolvedDir, err := resolveDataDir(*dataDir)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	engineVersion := singbox.CurrentVersion()
	logger.Info("engine dependency resolved",
		slog.String("engine", string(kernel.EngineSingBox)),
		slog.String("version", engineVersion.EffectiveVersion),
		slog.String("supported_series", engineVersion.Series),
	)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.New(ctx, bootstrap.Options{
		DataDir: resolvedDir,
		Factory: singbox.NewFactory(),
		Logger:  logger,
	})
	if err != nil {
		return fmt.Errorf("initialize boxd: %w", err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			logger.Error("shutdown failed", slog.Any("error", closeErr))
		}
	}()

	if *startKernel {
		if err := app.Control.Kernel().Start(ctx); err != nil {
			return fmt.Errorf("start kernel: %w", err)
		}
	}
	status, err := app.Control.Kernel().Status(ctx)
	if err != nil {
		return fmt.Errorf("read kernel status: %w", err)
	}
	logger.Info("boxd ready",
		slog.String("data_dir", resolvedDir),
		slog.String("database", app.DatabasePath()),
		slog.String("kernel_state", string(status.State)),
	)

	<-ctx.Done()
	logger.Info("shutdown requested")
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
	return filepath.Join(base, "boxd"), nil
}
