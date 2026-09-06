package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/service"
)

// desktopConfig 桌面应用运行配置。
type desktopConfig struct {
	Mode            string // embedded | remote
	RemoteURL       string
	DataDir         string
	ConfigPath      string
	Username        string
	Password        string
	RefreshInterval int
}

// defaultDesktopConfig 内嵌模式默认使用统一数据目录 ~/.boxd/。
func defaultDesktopConfig() desktopConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	boxdDir := filepath.Join(home, ".boxd")
	return desktopConfig{
		Mode:            "embedded",
		RemoteURL:       "http://127.0.0.1:9091",
		DataDir:         boxdDir,
		ConfigPath:      filepath.Join(boxdDir, "config.json"),
		Username:        "admin",
		Password:        "",
		RefreshInterval: 60,
	}
}

// desktopRuntime 聚合桌面端共享的运行时依赖。
type desktopRuntime struct {
	cfg             desktopConfig
	db              *bbolt.DB
	svc             *service.ServiceSet
	instance        *core.SBInstance
	autostart       *application.AutostartManager
	backgroundStop  func() // 停止后台服务
}

// initRuntime 初始化内嵌模式所需的 DB 与核心依赖。
func initRuntime(cfg desktopConfig) (*desktopRuntime, error) {
	if cfg.Mode == "remote" {
		return &desktopRuntime{cfg: cfg}, nil
	}

	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	// 将工作目录切换到数据目录，避免 sing-box cache.db 写入安装目录（Program Files 无写权限）。
	if err := os.Chdir(cfg.DataDir); err != nil {
		log.Printf("warning: chdir to data dir failed: %v", err)
	}
	// 配置文件缺失时自动生成最小可用配置，保证内核可启动。
	if created, err := core.EnsureConfigFile(cfg.ConfigPath); err != nil {
		return nil, fmt.Errorf("ensure config file: %w", err)
	} else if created {
		log.Printf("generated default config at %s", cfg.ConfigPath)
	}

	dbPath := filepath.Join(cfg.DataDir, "boxd.db")
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	settings := core.NewSettingsManager(db)
	if _, err := settings.EnsureAdminCredential(cfg.Username, cfg.Password); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init credential: %w", err)
	}
	if _, _, err := settings.EnsureJWTSecret(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init jwt secret: %w", err)
	}

	kernelLogWriter := core.NewLogWriter(200)
	appLogWriter := core.NewLogWriter(200)
	instance := core.NewSBInstance(cfg.ConfigPath, kernelLogWriter)

	deps := service.Deps{
		DB:               db,
		ConfigPath:       cfg.ConfigPath,
		DataDir:          cfg.DataDir,
		Username:         cfg.Username,
		Version:          core.Version,
		Settings:         settings,
		Instance:         instance,
		NodeManager:      core.NewNodeManager(db),
		SubManager:       core.NewSubscriptionManager(db, cfg.DataDir),
		RuleSetInstaller: core.NewLoyalsoldierRuleSetInstaller(cfg.DataDir),
		RuleSetUpdater:   core.NewRuleSetUpdater(cfg.ConfigPath, cfg.DataDir, nil, instance.Stop, instance.Start),
		KernelLogWriter:  kernelLogWriter,
		AppLogWriter:     appLogWriter,
		ApplyHistory:     core.NewConfigApplyHistoryManager(db),
		RouteMetadata:    core.NewRouteRuleMetadataManager(db),
	}
	rt := &desktopRuntime{
		cfg:      cfg,
		db:       db,
		svc:      service.New(deps),
		instance: instance,
	}
	// 启动后台服务（规则集自动更新 + 订阅自动刷新），与 cmd/boxd 服务端行为一致。
	ruleSetAutoUpdater := core.NewRuleSetAutoUpdater(settings, deps.RuleSetUpdater)
	subscriptionAutoRefresher := core.NewSubscriptionAutoRefresher(deps.SubManager, nil, cfg.RefreshInterval)
	rt.backgroundStop = func() {
		subscriptionAutoRefresher.Stop()
		ruleSetAutoUpdater.Stop()
	}
	if settings.Get("kernel_autostart") == "true" {
		if err := instance.Start(); err != nil {
			log.Printf("kernel autostart failed: %v", err)
		} else {
			log.Printf("kernel autostarted")
			ruleSetAutoUpdater.Start()
			subscriptionAutoRefresher.Start()
		}
	}
	return rt, nil
}

// close 关闭桌面运行时依赖。
func (r *desktopRuntime) close() error {
	if r.backgroundStop != nil {
		r.backgroundStop()
	}
	var err error
	if r.instance != nil {
		err = r.instance.Stop()
	}
	if r.db != nil {
		if closeErr := r.db.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// parseDesktopConfig 解析命令行/环境配置。
func parseDesktopConfig() desktopConfig {
	cfg := defaultDesktopConfig()
	if env := os.Getenv("BOXD_DESKTOP_MODE"); env == "remote" {
		cfg.Mode = "remote"
	}
	if env := os.Getenv("BOXD_REMOTE_URL"); env != "" {
		cfg.RemoteURL = env
	}
	if env := os.Getenv("BOXD_DATA_DIR"); env != "" {
		cfg.DataDir = env
	}
	if env := os.Getenv("BOXD_CONFIG"); env != "" {
		cfg.ConfigPath = env
	}
	if env := os.Getenv("BOXD_USERNAME"); env != "" {
		cfg.Username = env
	}
	if env := os.Getenv("BOXD_PASSWORD"); env != "" {
		cfg.Password = env
	}
	return cfg
}
