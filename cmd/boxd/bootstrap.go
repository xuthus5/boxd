package main

import (
	"log/slog"
	"net/http"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/api"
	"github.com/xuthus5/boxd/internal/config"
	"github.com/xuthus5/boxd/internal/core"
)

type backgroundService interface {
	Start()
	Stop()
}

type handlerRuntime struct {
	handler         http.Handler
	kernelLogWriter *core.LogWriter
	appLogWriter    *core.LogWriter
	services        []backgroundService
	stopStats       func()
	stopKernel      func() error
}

type handlerBuildState struct {
	cfg                 *config.Config
	db                  *bbolt.DB
	settingsManager     *core.SettingsManager
	kernelLogWriter     *core.LogWriter
	appLogWriter        *core.LogWriter
	instance            *core.SBInstance
	subscriptionManager *core.SubscriptionManager
	nodeManager         *core.NodeManager
	ruleSetInstaller    *core.LoyalsoldierRuleSetInstaller
	ruleSetUpdater      *core.RuleSetUpdater
}

type apiHandlerSet struct {
	authHandler         *api.AuthHandler
	configHandler       *api.ConfigHandler
	serviceHandler      *api.ServiceHandler
	statsHandler        *api.StatsHandler
	importHandler       *api.ImportHandler
	subscriptionHandler *api.SubscriptionHandler
	nodesHandler        *api.NodesHandler
	testHandler         *api.TestHandler
	settingsHandler     *api.SettingsHandler
	backupHandler       *api.BackupHandler
	networkHandler      *api.NetworkHandler
	kernelHandler       *api.KernelHandler
	runtimeHandler      *api.RuntimeHandler
	ruleSetHandler      *api.RuleSetHandler
}

func (r *handlerRuntime) Start() {
	for _, service := range r.services {
		service.Start()
	}
}

func (r *handlerRuntime) Stop() {
	for index := len(r.services) - 1; index >= 0; index-- {
		r.services[index].Stop()
	}
	if r.stopStats != nil {
		r.stopStats()
	}
	if r.stopKernel != nil {
		if err := r.stopKernel(); err != nil {
			slog.Error("failed to stop sing-box", "err", err)
		}
	}
}

func newHandler(cfg *config.Config, db *bbolt.DB, settingsManager *core.SettingsManager) *handlerRuntime {
	state := newHandlerBuildState(cfg, db, settingsManager)
	handlers := newAPIHandlerSet(state)
	ruleSetAutoUpdater := core.NewRuleSetAutoUpdater(settingsManager, state.ruleSetUpdater)
	subscriptionAutoRefresher := core.NewSubscriptionAutoRefresher(
		state.subscriptionManager,
		handlers.subscriptionHandler.SyncConfig,
		cfg.RefreshInterval,
	)
	router := newHandlerRouter(state, handlers)

	if settingsManager.Get("kernel_autostart") == "true" {
		if err := state.instance.Start(); err != nil {
			slog.Error("kernel autostart failed", "err", err)
		} else {
			slog.Info("kernel autostarted")
		}
	}

	return &handlerRuntime{
		handler:         router,
		kernelLogWriter: state.kernelLogWriter,
		appLogWriter:    state.appLogWriter,
		services:        []backgroundService{ruleSetAutoUpdater, subscriptionAutoRefresher},
		stopStats:       handlers.statsHandler.Stop,
		stopKernel:      state.instance.Stop,
	}
}

func newHandlerBuildState(
	cfg *config.Config,
	db *bbolt.DB,
	settingsManager *core.SettingsManager,
) handlerBuildState {
	kernelLogWriter := core.NewLogWriter(200)
	appLogWriter := core.NewLogWriter(200)
	instance := core.NewSBInstance(cfg.ConfigPath, kernelLogWriter)
	subscriptionManager := core.NewSubscriptionManager(db, cfg.DataDir)
	nodeManager := core.NewNodeManager(db)
	ruleSetInstaller := core.NewLoyalsoldierRuleSetInstaller(cfg.DataDir)
	ruleSetUpdater := core.NewRuleSetUpdater(cfg.ConfigPath, cfg.DataDir, ruleSetInstaller, instance.Stop, instance.Start)
	return handlerBuildState{
		cfg:                 cfg,
		db:                  db,
		settingsManager:     settingsManager,
		kernelLogWriter:     kernelLogWriter,
		appLogWriter:        appLogWriter,
		instance:            instance,
		subscriptionManager: subscriptionManager,
		nodeManager:         nodeManager,
		ruleSetInstaller:    ruleSetInstaller,
		ruleSetUpdater:      ruleSetUpdater,
	}
}

func newAPIHandlerSet(state handlerBuildState) apiHandlerSet {
	cfg := state.cfg
	settingsManager := state.settingsManager
	configHandler := api.NewConfigHandlerWithHistory(
		cfg.ConfigPath,
		state.instance,
		state.ruleSetInstaller,
		core.NewDefaultOutboundsInstaller(),
		core.NewDefaultRouteInstaller(),
		core.NewDefaultDNSInstaller(),
		core.NewConfigApplyHistoryManager(state.db),
		core.NewRouteRuleMetadataManager(state.db),
	)
	subscriptionHandler := api.NewSubscriptionHandler(state.subscriptionManager, state.nodeManager, cfg.ConfigPath, state.instance)
	return apiHandlerSet{
		authHandler:         api.NewAuthHandler(cfg.Username, cfg.Password, settingsManager),
		configHandler:       configHandler,
		serviceHandler:      api.NewServiceHandler(state.instance),
		statsHandler:        api.NewStatsHandler(state.kernelLogWriter, state.appLogWriter, state.instance),
		importHandler:       api.NewImportHandler(state.nodeManager, state.subscriptionManager, cfg.ConfigPath, state.instance),
		subscriptionHandler: subscriptionHandler,
		nodesHandler:        api.NewNodesHandler(state.nodeManager, state.subscriptionManager, cfg.ConfigPath, state.instance),
		testHandler: api.NewTestHandler(func() string {
			u := settingsManager.Get("url_test")
			if u == "" {
				u = "https://cp.cloudflare.com/"
			}
			return u
		}, state.nodeManager, state.instance),
		settingsHandler: api.NewSettingsHandler(settingsManager, cfg.Username),
		backupHandler:   api.NewBackupHandler(state.db, cfg.ConfigPath, core.Version),
		networkHandler:  api.NewNetworkHandler(),
		kernelHandler:   api.NewKernelHandler(core.Version),
		runtimeHandler:  api.NewRuntimeHandler(state.instance),
		ruleSetHandler:  api.NewRuleSetHandler(state.ruleSetUpdater, settingsManager),
	}
}

func newHandlerRouter(state handlerBuildState, handlers apiHandlerSet) http.Handler {
	cfg := state.cfg
	return api.NewRouter(
		staticFS,
		handlers.authHandler,
		handlers.configHandler,
		handlers.serviceHandler,
		handlers.statsHandler,
		handlers.importHandler,
		handlers.subscriptionHandler,
		handlers.nodesHandler,
		handlers.testHandler,
		handlers.settingsHandler,
		handlers.backupHandler,
		handlers.networkHandler,
		handlers.kernelHandler,
		handlers.runtimeHandler,
		handlers.ruleSetHandler,
		state.settingsManager,
		cfg.CORSAllowedOrigins,
		state.instance,
		func() error { return checkReadiness(state.db, cfg.ConfigPath) },
	)
}
