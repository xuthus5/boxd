package main

import (
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
	startFn         func() error
	autostartKernel bool
}

// initRuntime 初始化内嵌模式所需的 DB 与核心依赖。
func initRuntime(cfg desktopConfig) (*desktopRuntime, error) {
	if cfg.Mode == "remote" {
		return &desktopRuntime{cfg: cfg}, nil
	}
	state, err := initializeDesktopState(cfg)
	if err != nil {
		return nil, err
	}
	deps := newDesktopDependencies(cfg, state)
	rt := &desktopRuntime{
		cfg:             cfg,
		db:              state.db,
		svc:             service.New(deps),
		instance:        deps.Instance,
		autostartKernel: state.autostartKernel,
	}
	configureDesktopBackground(rt, rt.instance.Start)
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
