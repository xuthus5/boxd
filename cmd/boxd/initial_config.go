package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/xuthus5/boxd/internal/config"
	"github.com/xuthus5/boxd/internal/core"
)

func initializeServerSettings(cfg *config.Config, settings *core.SettingsManager) error {
	defaultPassword, err := settings.EnsureAdminCredential(cfg.Username, cfg.Password)
	if err != nil {
		return fmt.Errorf("init administrator credential: %w", err)
	}
	if defaultPassword {
		slog.Warn("default administrator password is active; change it in settings")
	}
	_, generated, err := settings.EnsureJWTSecret()
	if err != nil {
		return fmt.Errorf("init jwt secret: %w", err)
	}
	if generated {
		slog.Info("jwt secret auto-generated and persisted to database")
	}
	return nil
}

func initializeServerConfig(cfg *config.Config, settings *core.SettingsManager) error {
	created, err := core.EnsureDefaultConfig(context.Background(), cfg.ConfigPath, cfg.DataDir)
	if err != nil {
		return fmt.Errorf("ensure default config: %w", err)
	}
	if created {
		slog.Info("default config generated", "path", cfg.ConfigPath)
	}
	_, err = settings.EnsureKernelAutostartDefault(created)
	return err
}
