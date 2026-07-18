package api

import (
	"io/fs"
	"net/http"
	"strings"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxui/internal/core"
)

func NewRouter(
	staticFS fs.FS,
	authHandler *AuthHandler,
	configHandler *ConfigHandler,
	serviceHandler *ServiceHandler,
	statsHandler *StatsHandler,
	importHandler *ImportHandler,
	subscriptionHandler *SubscriptionHandler,
	nodesHandler *NodesHandler,
	testHandler *TestHandler,
	settingsHandler *SettingsHandler,
	networkHandler *NetworkHandler,
	kernelHandler *KernelHandler,
	runtimeHandler *RuntimeHandler,
	ruleSetHandler *RuleSetHandler,
	secrets core.SecretProvider,
	corsAllowedOrigins []string,
	instance *core.SBInstance,
	readiness ...func() error,
) http.Handler {
	r := chi.NewRouter()

	r.Use(LoggerMiddleware)
	r.Use(SecurityHeadersMiddleware)
	r.Use(BodyLimitMiddleware)
	r.Use(CORSMiddleware(corsAllowedOrigins))
	r.Use(RecoveryMiddleware)

	var readinessCheck func() error
	if len(readiness) > 0 {
		readinessCheck = readiness[0]
	}
	healthHandler := NewHealthHandler(readinessCheck)
	r.Get("/health", healthHandler.Liveness)
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.With(AuthMiddleware(secrets)).Post("/logout", authHandler.Logout)
	})

	r.Route("/api/config", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Get("/", configHandler.GetConfig)
		r.Put("/", configHandler.UpdateConfig)
		r.Get("/raw", configHandler.GetRawConfig)
		r.Put("/raw", configHandler.UpdateRawConfig)
		r.Post("/dns/defaults", configHandler.InstallDefaultDNS)
		r.Post("/rule-sets/defaults", configHandler.InstallDefaultRuleSets)
		r.Get("/rule-sets/status", ruleSetHandler.Status)
		r.Post("/rule-sets/update", ruleSetHandler.Update)
		r.Get("/rule-sets/auto-update", ruleSetHandler.GetAutoUpdate)
		r.Put("/rule-sets/auto-update", ruleSetHandler.SetAutoUpdate)
		r.Post("/outbounds/defaults", configHandler.InstallDefaultOutbounds)
		r.Post("/route/defaults", configHandler.InstallDefaultRouteRules)
		r.Get("/route/rule-metadata", configHandler.GetRouteRuleMetadata)
		r.Put("/route/rule-metadata", configHandler.UpdateRouteRuleMetadata)
	})

	r.Route("/api/service", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Post("/start", serviceHandler.Start)
		r.Post("/stop", serviceHandler.Stop)
		r.Post("/restart", serviceHandler.Restart)
		r.Get("/status", serviceHandler.Status)
	})

	r.Route("/api/stats", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Get("/traffic/history", statsHandler.TrafficHistory)
		r.Get("/traffic", statsHandler.TrafficSSE)
		r.Get("/logs", statsHandler.LogsSSE)
		r.Get("/app-logs", statsHandler.AppLogsSSE)
		r.Get("/connections", statsHandler.ConnectionsSSE)
		r.Delete("/connections", statsHandler.CloseAllConnections)
		r.Delete("/connections/{id}", statsHandler.CloseConnection)
	})

	r.Route("/api/import", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Post("/link", importHandler.ImportLink)
		r.Post("/save", importHandler.SaveNode)
	})

	r.Route("/api/nodes", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Get("/", nodesHandler.List)
		r.Get("/{tag}", nodesHandler.Get)
		r.Put("/{tag}", nodesHandler.Update)
		r.Get("/groups", runtimeHandler.OutboundGroups)
		r.Get("/{tag}/delay", runtimeHandler.OutboundDelay)
		r.Post("/test", testHandler.Run)
		r.Post("/test-batch", testHandler.RunBatch)
		r.Get("/test-results", testHandler.ListResults)
		r.Post("/selectors/{group}/select", runtimeHandler.SelectOutbound)
		r.Post("/groups/{group}/urltest", runtimeHandler.URLTestDelays)
		r.Post("/sync-config", nodesHandler.SyncToConfig)
		r.Delete("/{tag}", nodesHandler.Delete)
	})

	r.Route("/api/subscriptions", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Get("/", subscriptionHandler.List)
		r.Post("/", subscriptionHandler.Create)
		r.Get("/{id}", subscriptionHandler.Get)
		r.Put("/{id}", subscriptionHandler.Update)
		r.Delete("/{id}", subscriptionHandler.Delete)
		r.Post("/{id}/refresh", subscriptionHandler.Refresh)
		r.Post("/refresh-all", subscriptionHandler.RefreshAll)
	})

	r.Route("/api/settings", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Get("/url-test", settingsHandler.GetTestURL)
		r.Put("/url-test", settingsHandler.SetTestURL)
		r.Get("/urltest-defaults", settingsHandler.GetURLTestDefaults)
		r.Put("/urltest-defaults", settingsHandler.SetURLTestDefaults)
		r.Get("/kernel-autostart", settingsHandler.GetKernelAutostart)
		r.Put("/kernel-autostart", settingsHandler.SetKernelAutostart)
		r.Get("/jwt-secret", settingsHandler.GetJWTSecret)
		r.Put("/jwt-secret", settingsHandler.SetJWTSecret)
		r.Get("/password", settingsHandler.GetPasswordStatus)
		r.Put("/password", settingsHandler.ChangePassword)
	})

	r.Route("/api/network", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Get("/interfaces", networkHandler.GetInterfaces)
	})

	r.Route("/api/runtime", func(r chi.Router) {
		r.Use(AuthMiddleware(secrets))
		r.Get("/version", kernelHandler.Version)
		r.Get("/memory", kernelHandler.Memory)
		r.Post("/gc", kernelHandler.GC)
		r.Post("/dns/flush", runtimeHandler.FlushDNS)
		r.Post("/fakeip/flush", runtimeHandler.FlushFakeIP)
	})

	mountStatic(r, staticFS)

	return r
}

func mountStatic(r chi.Router, staticFS fs.FS) {
	if staticFS == nil {
		return
	}

	staticSubFS, err := fs.Sub(staticFS, "ui/dist")
	if err != nil {
		return
	}
	if _, err := fs.Stat(staticSubFS, "index.html"); err != nil {
		return
	}

	fileServer := http.FileServer(http.FS(staticSubFS))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		rawPath := r.URL.Path
		cleanPath := strings.TrimPrefix(rawPath, "/")
		if _, err := fs.Stat(staticSubFS, cleanPath); err == nil {
			if strings.HasPrefix(cleanPath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(cleanPath, "assets/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
