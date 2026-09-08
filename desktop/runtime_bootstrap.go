package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/service"
)

type desktopInitialState struct {
	db              *bbolt.DB
	settings        *core.SettingsManager
	autostartKernel bool
}

const desktopLogCapacity = 200

func initializeDesktopState(cfg desktopConfig) (desktopInitialState, error) {
	created, err := prepareDesktopConfig(cfg)
	if err != nil {
		return desktopInitialState{}, err
	}
	db, err := bbolt.Open(filepath.Join(cfg.DataDir, "boxd.db"), 0600, nil)
	if err != nil {
		return desktopInitialState{}, fmt.Errorf("open database: %w", err)
	}
	state := desktopInitialState{db: db, settings: core.NewSettingsManager(db)}
	if err := initializeDesktopCredentials(state.settings, cfg); err != nil {
		_ = db.Close()
		return desktopInitialState{}, err
	}
	state.autostartKernel, err = state.settings.EnsureKernelAutostartDefault(created)
	if err != nil {
		_ = db.Close()
		return desktopInitialState{}, err
	}
	return state, nil
}

func prepareDesktopConfig(cfg desktopConfig) (bool, error) {
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return false, fmt.Errorf("create data dir: %w", err)
	}
	// 缓存使用数据目录，避免写入只读安装目录。
	if err := os.Chdir(cfg.DataDir); err != nil {
		slog.Warn("chdir to data dir failed", "err", err)
	}
	created, err := core.EnsureDefaultConfig(context.Background(), cfg.ConfigPath, cfg.DataDir)
	if err != nil {
		return false, fmt.Errorf("ensure config file: %w", err)
	}
	if created {
		slog.Info("generated default config", "path", cfg.ConfigPath)
	}
	return created, nil
}

func initializeDesktopCredentials(settings *core.SettingsManager, cfg desktopConfig) error {
	if _, err := settings.EnsureAdminCredential(cfg.Username, cfg.Password); err != nil {
		return fmt.Errorf("init credential: %w", err)
	}
	if _, _, err := settings.EnsureJWTSecret(); err != nil {
		return fmt.Errorf("init jwt secret: %w", err)
	}
	return nil
}

func newDesktopDependencies(cfg desktopConfig, state desktopInitialState) service.Deps {
	kernelLogWriter := core.NewLogWriter(desktopLogCapacity)
	instance := core.NewSBInstance(cfg.ConfigPath, kernelLogWriter)
	return service.Deps{
		DB: state.db, ConfigPath: cfg.ConfigPath, DataDir: cfg.DataDir,
		Username: cfg.Username, Version: core.Version, Settings: state.settings, Instance: instance,
		NodeManager: core.NewNodeManager(state.db), SubManager: core.NewSubscriptionManager(state.db, cfg.DataDir),
		RuleSetInstaller: core.NewLoyalsoldierRuleSetInstaller(cfg.DataDir),
		RuleSetUpdater:   core.NewRuleSetUpdater(cfg.ConfigPath, cfg.DataDir, nil, instance.Stop, instance.Start),
		KernelLogWriter:  kernelLogWriter, AppLogWriter: core.NewLogWriter(desktopLogCapacity),
		ApplyHistory: core.NewConfigApplyHistoryManager(state.db), RouteMetadata: core.NewRouteRuleMetadataManager(state.db),
	}
}

func configureDesktopBackground(rt *desktopRuntime, start func() error) {
	deps := rt.svc.Deps
	rules := core.NewRuleSetAutoUpdater(deps.Settings, deps.RuleSetUpdater)
	subscriptions := core.NewSubscriptionAutoRefresher(deps.SubManager, nil, rt.cfg.RefreshInterval)
	rt.backgroundStop = func() {
		subscriptions.Stop()
		rules.Stop()
	}
	rt.startFn = func() error {
		if err := start(); err != nil {
			return err
		}
		rules.Start()
		subscriptions.Start()
		return nil
	}
}
