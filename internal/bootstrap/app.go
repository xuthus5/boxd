// Package bootstrap is the only composition root for production hosts.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/dnsprobe"
	"github.com/xuthus5/boxd/internal/adapters/outbound/httpdownload"
	"github.com/xuthus5/boxd/internal/adapters/outbound/networkprobe"
	"github.com/xuthus5/boxd/internal/adapters/outbound/nodelink"
	"github.com/xuthus5/boxd/internal/adapters/outbound/singboxconfig"
	"github.com/xuthus5/boxd/internal/adapters/outbound/sqlite"
	"github.com/xuthus5/boxd/internal/adapters/outbound/subscriptionjson"
	"github.com/xuthus5/boxd/internal/controlplane"
	"github.com/xuthus5/boxd/internal/kernel/manager"
	"github.com/xuthus5/boxd/internal/webauth"
	"github.com/xuthus5/boxd/kernel"
)

const databaseName = "boxd.sqlite"

// Options configures a host-neutral application instance.
type Options struct {
	DataDir            string
	Factory            kernel.Factory
	Compiler           control.ConfigCompiler
	Downloader         control.Downloader
	SubscriptionParser control.SubscriptionParser
	NodeCodec          control.NodeCodec
	NodeProber         control.NodeProber
	DNSProber          control.DNSProber
	Logger             *slog.Logger
}

// App owns all long-lived infrastructure in shutdown order.
type App struct {
	Control control.Application
	Kernel  kernel.Manager

	manager   *manager.Manager
	store     *sqlite.DB
	scheduler control.SchedulerAPI
	logger    *slog.Logger
}

// New opens storage, applies migrations, and wires the control plane.
func New(ctx context.Context, opts Options) (*App, error) {
	if opts.DataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	if opts.Factory == nil {
		return nil, fmt.Errorf("engine factory is required")
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if err := os.MkdirAll(opts.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	databasePath := filepath.Join(opts.DataDir, databaseName)
	store, err := sqlite.Open(ctx, databasePath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := store.Migrate(ctx); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	kernelManager, err := manager.New(ctx, manager.Options{
		Factory: opts.Factory,
		Store:   store.Configs(),
		Open: kernel.OpenOptions{
			Name:    "boxd",
			WorkDir: opts.DataDir,
		},
	})
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("construct kernel manager: %w", err)
	}

	compiler := opts.Compiler
	if compiler == nil {
		if opts.Factory.Kind() != kernel.EngineSingBox {
			_ = kernelManager.Close()
			_ = store.Close()
			return nil, fmt.Errorf("a config compiler is required for engine %s", opts.Factory.Kind())
		}
		compiler = singboxconfig.New()
	}
	profileService := controlplane.NewProfileService(kernelManager, store.Profiles(), compiler)
	downloader := opts.Downloader
	if downloader == nil {
		downloader = httpdownload.New()
	}
	nodeCodec := opts.NodeCodec
	if nodeCodec == nil {
		nodeCodec = nodelink.New()
	}
	subscriptionParser := opts.SubscriptionParser
	if subscriptionParser == nil {
		subscriptionParser = subscriptionjson.New(nodeCodec)
	}
	nodeProber := opts.NodeProber
	if nodeProber == nil {
		nodeProber = networkprobe.New()
	}
	dnsProber := opts.DNSProber
	if dnsProber == nil {
		dnsProber = dnsprobe.New()
	}
	application := controlplane.NewWithDependencies(
		kernelManager,
		store.Configs(),
		store.Settings(),
		profileService,
		controlplane.Dependencies{
			Downloader:           downloader,
			SubscriptionParser:   subscriptionParser,
			NodeCodec:            nodeCodec,
			NodeProber:           nodeProber,
			DNSProber:            dnsProber,
			RuleSetRepository:    store.RuleSets(),
			TestResultRepository: store.TestResults(),
			AuditRepository:      store.Audit(),
		},
	)
	if _, err := application.Setup().EnsureDefaults(ctx); err != nil {
		_ = kernelManager.Close()
		_ = store.Close()
		return nil, fmt.Errorf("initialize default profile: %w", err)
	}
	scheduler := application.Scheduler()
	if err := scheduler.Start(ctx); err != nil {
		_ = kernelManager.Close()
		_ = store.Close()
		return nil, fmt.Errorf("start subscription scheduler: %w", err)
	}
	if err := application.Observability().StartSampling(ctx); err != nil {
		_ = scheduler.Stop(context.Background())
		_ = kernelManager.Close()
		_ = store.Close()
		return nil, fmt.Errorf("start observation sampler: %w", err)
	}
	logger.Info("control plane initialized",
		slog.String("database", databasePath),
		slog.String("engine", string(opts.Factory.Kind())),
	)
	return &App{
		Control:   application,
		Kernel:    kernelManager,
		manager:   kernelManager,
		store:     store,
		scheduler: scheduler,
		logger:    logger,
	}, nil
}

// WebAuthRepository returns the browser credential store. Only the Web ingress
// should consume it; the control-plane facade has no authentication surface.
func (a *App) WebAuthRepository() webauth.Repository {
	if a == nil || a.store == nil {
		return nil
	}
	return a.store.WebAuth()
}

// DatabasePath returns the active SQLite file.
func (a *App) DatabasePath() string {
	if a == nil || a.store == nil {
		return ""
	}
	return a.store.Path()
}

// Close follows adapter, kernel, then repository shutdown order.
func (a *App) Close() error {
	if a == nil {
		return nil
	}
	ctx := context.Background()
	var closeErrors []error
	if a.Control != nil {
		a.Control.Observability().StopSampling()
	}
	if a.scheduler != nil {
		if err := a.scheduler.Stop(ctx); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if a.manager != nil {
		if err := a.manager.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if a.store != nil {
		if err := a.store.Checkpoint(ctx); err != nil {
			closeErrors = append(closeErrors, err)
		}
		if err := a.store.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}
