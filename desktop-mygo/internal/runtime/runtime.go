package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

const (
	logCapacity    = 200
	databaseName   = "boxd.db"
	unixDataMode   = 0o700
	unixFileMode   = 0o600
	remoteNotReady = "remote desktop mode is not implemented in the MyGo prototype"
)

type Runtime struct {
	cfg            Config
	db             *bbolt.DB
	svc            *service.ServiceSet
	instance       *core.SBInstance
	settings       *core.SettingsManager
	kernelLog      *core.LogWriter
	appLog         *core.LogWriter
	ruleSetUpdater *core.RuleSetAutoUpdater
	subscriptions  *core.SubscriptionAutoRefresher
	autostart      bool

	closeOnce sync.Once
	closeErr  error
}

func Open(cfg Config) (*Runtime, error) {
	if cfg.Mode == modeRemote {
		return nil, errors.New(remoteNotReady)
	}
	if cfg.Mode == "" {
		cfg.Mode = modeEmbedded
	}
	if err := os.MkdirAll(cfg.DataDir, unixDataMode); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.Chmod(cfg.DataDir, unixDataMode); err != nil {
		return nil, fmt.Errorf("secure data dir: %w", err)
	}
	if err := os.Chdir(cfg.DataDir); err != nil {
		slog.Warn("chdir to data dir failed", "err", err)
	}
	created, err := core.EnsureDefaultConfig(context.Background(), cfg.ConfigPath, cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("ensure config file: %w", err)
	}
	db, err := bbolt.Open(filepath.Join(cfg.DataDir, databaseName), unixFileMode, nil)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	rt := &Runtime{cfg: cfg, db: db}
	if err := rt.initialize(created); err != nil {
		_ = db.Close()
		return nil, err
	}
	return rt, nil
}

func (r *Runtime) initialize(created bool) error {
	r.settings = core.NewSettingsManager(r.db)
	if _, err := r.settings.EnsureAdminCredential(r.cfg.Username, r.cfg.Password); err != nil {
		return fmt.Errorf("init credential: %w", err)
	}
	if _, _, err := r.settings.EnsureJWTSecret(); err != nil {
		return fmt.Errorf("init jwt secret: %w", err)
	}
	var err error
	r.autostart, err = r.settings.EnsureKernelAutostartDefault(created)
	if err != nil {
		return fmt.Errorf("init kernel autostart: %w", err)
	}
	r.kernelLog = core.NewLogWriter(logCapacity)
	r.appLog = core.NewLogWriter(logCapacity)
	r.instance = core.NewSBInstance(r.cfg.ConfigPath, r.kernelLog)
	subscriptionManager := core.NewSubscriptionManager(r.db, r.cfg.DataDir)
	r.instance.SetSelectorApplyProvider(func() (map[string]string, error) {
		return core.ManagedSubscriptionSelectorTargets(subscriptionManager)
	})
	deps := service.Deps{
		DB: r.db, ConfigPath: r.cfg.ConfigPath, DataDir: r.cfg.DataDir,
		Username: r.cfg.Username, Version: core.Version, Settings: r.settings,
		Instance: r.instance, NodeManager: core.NewNodeManager(r.db), SubManager: subscriptionManager,
		RuleSetInstaller: core.NewLoyalsoldierRuleSetInstaller(r.cfg.DataDir),
		RuleSetUpdater:   core.NewRuleSetUpdater(r.cfg.ConfigPath, r.cfg.DataDir, nil, r.instance.Stop, r.instance.Start),
		KernelLogWriter:  r.kernelLog, AppLogWriter: r.appLog,
		ApplyHistory: core.NewConfigApplyHistoryManager(r.db), RouteMetadata: core.NewRouteRuleMetadataManager(r.db),
	}
	r.svc = service.New(deps)
	r.ruleSetUpdater = core.NewRuleSetAutoUpdater(r.settings, deps.RuleSetUpdater)
	r.subscriptions = core.NewSubscriptionAutoRefresher(subscriptionManager, nil, r.cfg.RefreshInterval)
	slog.SetDefault(slog.New(core.NewAppLogHandler(os.Stderr, r.appLog, slog.LevelInfo)))
	return nil
}

func (r *Runtime) Service() *service.ServiceSet { return r.svc }

func (r *Runtime) KernelAutostart() bool { return r.autostart }

func (r *Runtime) StartKernel() error {
	if r.instance == nil {
		return errors.New("runtime is not initialized")
	}
	if err := r.instance.Start(); err != nil {
		return err
	}
	if r.ruleSetUpdater != nil {
		r.ruleSetUpdater.Start()
	}
	if r.subscriptions != nil {
		r.subscriptions.Start()
	}
	return nil
}

func (r *Runtime) Logs() []core.LogEntry {
	if r.kernelLog == nil {
		return nil
	}
	kernel := r.kernelLog.Recent()
	app := r.appLog.Recent()
	logs := make([]core.LogEntry, 0, len(kernel)+len(app))
	logs = append(logs, kernel...)
	logs = append(logs, app...)
	return logs
}

func (r *Runtime) NodeTestResults() map[string]map[string]model.TestResult {
	if r.svc == nil || r.svc.Deps.NodeManager == nil {
		return nil
	}
	return r.svc.Deps.NodeManager.GetAllTestResults()
}

func (r *Runtime) Close() error {
	r.closeOnce.Do(func() {
		if r.ruleSetUpdater != nil {
			r.ruleSetUpdater.Stop()
		}
		if r.subscriptions != nil {
			r.subscriptions.Stop()
		}
		if r.instance != nil {
			if err := r.instance.Stop(); err != nil {
				r.closeErr = err
			}
		}
		if r.db != nil {
			if err := r.db.Close(); err != nil && r.closeErr == nil {
				r.closeErr = err
			}
		}
	})
	return r.closeErr
}
