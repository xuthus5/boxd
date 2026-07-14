package api

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

type fakeRuleSetInstaller struct {
	entries []map[string]any
	err     error
}

func (f *fakeRuleSetInstaller) Install(_ context.Context) ([]map[string]any, error) {
	return f.entries, f.err
}

type fakeRestartable struct {
	errs  []error
	calls int
}

func (f *fakeRestartable) Restart() error {
	if f.calls >= len(f.errs) {
		f.calls++
		return nil
	}
	err := f.errs[f.calls]
	f.calls++
	return err
}

func TestAuthHandlerLoginAndLogout(t *testing.T) {
	handler := NewAuthHandler("admin", "pass", staticSecretProvider("secret-key-123456"))

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "invalid json", body: `{`, want: http.StatusBadRequest},
		{name: "invalid credentials", body: `{"username":"admin","password":"bad"}`, want: http.StatusUnauthorized},
		{name: "valid credentials", body: `{"username":"admin","password":"pass"}`, want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.Login(rr, jsonRequest(http.MethodPost, "/login", tt.body))
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d", rr.Code, tt.want)
			}
			if tt.want == http.StatusOK {
				body := decodeBody[map[string]any](t, rr)
				if body["token"] == "" {
					t.Fatal("token should not be empty")
				}
			}
		})
	}

	rr := httptest.NewRecorder()
	handler.Logout(rr, httptest.NewRequest(http.MethodPost, "/logout", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAuthHandlerLoginRateLimit(t *testing.T) {
	handler := NewAuthHandler("admin", "pass", staticSecretProvider("secret-key-123456"))

	base := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	handler.now = func() time.Time { return base }

	for range maxLoginFailures {
		rr := httptest.NewRecorder()
		req := jsonRequest(http.MethodPost, "/login", `{"username":"admin","password":"bad"}`)
		req.RemoteAddr = "198.51.100.1:1234"
		handler.Login(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 before lockout, got %d", rr.Code)
		}
	}

	rr := httptest.NewRecorder()
	req := jsonRequest(http.MethodPost, "/login", `{"username":"admin","password":"pass"}`)
	req.RemoteAddr = "198.51.100.1:1234"
	handler.Login(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}

	base = base.Add(loginLockDuration + time.Second)
	rr = httptest.NewRecorder()
	req = jsonRequest(http.MethodPost, "/login", `{"username":"admin","password":"pass"}`)
	req.RemoteAddr = "198.51.100.1:1234"
	handler.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 after lock duration, got %d", rr.Code)
	}
}

func TestConfigHandlerJSONAndRaw(t *testing.T) {
	configPath := t.TempDir() + "/config.json"
	writeConfigFile(t, configPath, map[string]any{"log": map[string]any{"level": "info"}})
	handler := NewConfigHandler(configPath, nil, nil, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.GetConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GetConfig status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.UpdateConfig(rr, jsonRequest(http.MethodPut, "/api/config", `{"log":{"level":"debug"}}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("UpdateConfig status = %d", rr.Code)
	}

	restartFail := NewConfigHandler(configPath, &fakeRestartable{errs: []error{errors.New("restart failed"), nil}}, nil, nil, nil, nil)
	rr = httptest.NewRecorder()
	restartFail.UpdateConfig(rr, jsonRequest(http.MethodPut, "/api/config", `{"log":{"level":"warn"}}`))
	envelope := decodeEnvelope(t, rr)
	if rr.Code != http.StatusOK || envelope.Status != model.StatusRolledBack {
		t.Fatalf("restart failure response = %d %s", rr.Code, rr.Body.String())
	}
	cfg := decodeConfigFile(t, configPath)
	logCfg := cfg["log"].(map[string]any)
	if logCfg["level"] != "debug" {
		t.Fatalf("config should rollback to debug, got %#v", logCfg["level"])
	}

	rr = httptest.NewRecorder()
	handler.GetRawConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config/raw", nil))
	if !strings.Contains(rr.Body.String(), `"debug"`) {
		t.Fatalf("raw config = %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.UpdateRawConfig(rr, jsonRequest(http.MethodPut, "/api/config/raw", `{"dns":{}}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("UpdateRawConfig status = %d", rr.Code)
	}

	mode := fileMode(t, configPath)
	if mode != 0600 {
		t.Fatalf("config mode = %o, want 0600", mode)
	}
}

func TestConfigHandlerErrors(t *testing.T) {
	missing := NewConfigHandler(t.TempDir()+"/missing.json", nil, nil, nil, nil, nil)
	rr := httptest.NewRecorder()
	missing.GetConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing config status = %d", rr.Code)
	}

	configPath := t.TempDir() + "/config.json"
	if err := os.WriteFile(configPath, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	handler := NewConfigHandler(configPath, nil, nil, nil, nil, nil)

	rr = httptest.NewRecorder()
	handler.GetConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("invalid config status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.UpdateConfig(rr, jsonRequest(http.MethodPut, "/api/config", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.UpdateConfig(rr, httptest.NewRequest(http.MethodPut, "/api/config", errorBody{}))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("read error status = %d", rr.Code)
	}

	dirHandler := NewConfigHandler(t.TempDir(), nil, nil, nil, nil, nil)
	rr = httptest.NewRecorder()
	dirHandler.UpdateRawConfig(rr, jsonRequest(http.MethodPut, "/api/config/raw", `{}`))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("raw write error status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.UpdateRawConfig(rr, httptest.NewRequest(http.MethodPut, "/api/config/raw", errorBody{}))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("raw read error status = %d", rr.Code)
	}

	restartFail := NewConfigHandler(configPath, &fakeRestartable{errs: []error{errors.New("restart failed"), nil}}, nil, nil, nil, nil)
	rr = httptest.NewRecorder()
	restartFail.UpdateRawConfig(rr, jsonRequest(http.MethodPut, "/api/config/raw", `{}`))
	envelope := decodeEnvelope(t, rr)
	if rr.Code != http.StatusOK || envelope.Status != model.StatusRolledBack {
		t.Fatalf("raw restart failure response = %d %s", rr.Code, rr.Body.String())
	}

	writeConfigFile(t, configPath, map[string]any{"log": map[string]any{"level": "info"}})
	rr = httptest.NewRecorder()
	handler.UpdateRawConfig(rr, jsonRequest(http.MethodPut, "/api/config/raw", `{"outbounds":"bad"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid sing-box config status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	missing.GetRawConfig(rr, httptest.NewRequest(http.MethodGet, "/api/config/raw", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing raw status = %d", rr.Code)
	}
}

func TestConfigHandlerInstallDefaultRuleSets(t *testing.T) {
	configPath := t.TempDir() + "/config.json"
	writeConfigFile(t, configPath, map[string]any{
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "custom", "type": "local", "path": "/tmp/custom.json"},
				map[string]any{"tag": "loyalsoldier-direct", "type": "remote", "url": "https://old.invalid/direct.txt"},
			},
		},
	})

	handler := NewConfigHandler(configPath, nil, &fakeRuleSetInstaller{
		entries: []map[string]any{
			{"tag": "loyalsoldier-direct", "type": "local", "format": "source", "path": "/var/lib/boxui/rule-sets/loyalsoldier-direct.json"},
			{"tag": "loyalsoldier-proxy", "type": "local", "format": "source", "path": "/var/lib/boxui/rule-sets/loyalsoldier-proxy.json"},
			{"tag": "loyalsoldier-reject", "type": "local", "format": "source", "path": "/var/lib/boxui/rule-sets/loyalsoldier-reject.json"},
		},
	}, nil, nil, nil)

	rr := httptest.NewRecorder()
	handler.InstallDefaultRuleSets(rr, httptest.NewRequest(http.MethodPost, "/api/config/rule-sets/defaults", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("InstallDefaultRuleSets status = %d %s", rr.Code, rr.Body.String())
	}

	body := decodeEnvelope(t, rr)
	if body.Status != model.StatusOK {
		t.Fatalf("response status = %#v", body.Status)
	}

	cfg := decodeConfigFile(t, configPath)
	route := cfg["route"].(map[string]any)
	sets := route["rule_set"].([]any)
	if len(sets) != 4 {
		t.Fatalf("rule_set len = %d, want 4", len(sets))
	}
	customFound := false
	directUpdated := false
	for _, item := range sets {
		rs := item.(map[string]any)
		switch rs["tag"] {
		case "custom":
			customFound = true
		case "loyalsoldier-direct":
			if rs["path"] == "/var/lib/boxui/rule-sets/loyalsoldier-direct.json" && rs["type"] == "local" {
				directUpdated = true
			}
		}
	}
	if !customFound || !directUpdated {
		t.Fatalf("merged rule sets invalid: %#v", sets)
	}
}

func TestConfigHandlerInstallDefaultOutbounds(t *testing.T) {
	configPath := t.TempDir() + "/config.json"
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "node-a", "type": "vless", "server": "1.2.3.4", "server_port": 443},
		},
		"route": map[string]any{},
	})

	handler := NewConfigHandler(configPath, nil, nil, core.NewDefaultOutboundsInstaller(), nil, nil)

	rr := httptest.NewRecorder()
	handler.InstallDefaultOutbounds(rr, httptest.NewRequest(http.MethodPost, "/api/config/outbounds/defaults", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("InstallDefaultOutbounds status = %d %s", rr.Code, rr.Body.String())
	}

	cfg := decodeConfigFile(t, configPath)
	outbounds := cfg["outbounds"].([]any)
	route := cfg["route"].(map[string]any)
	if route["final"] != "proxy" {
		t.Fatalf("route.final = %#v", route["final"])
	}

	byTag := make(map[string]map[string]any)
	for _, item := range outbounds {
		ob := item.(map[string]any)
		byTag[ob["tag"].(string)] = ob
	}
	for _, tag := range []string{"direct", "block", "proxy", "auto", "whitelist", "blacklist"} {
		if _, ok := byTag[tag]; !ok {
			t.Fatalf("missing default outbound %q", tag)
		}
	}
	if _, ok := byTag["dns-out"]; ok {
		t.Fatalf("dns-out should not be installed")
	}
}

func TestConfigHandlerInstallDefaultRouteRules(t *testing.T) {
	configPath := t.TempDir() + "/config.json"
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
			map[string]any{"tag": "block", "type": "block"},
			map[string]any{"tag": "proxy", "type": "selector"},
		},
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "loyalsoldier-direct", "type": "local"},
				map[string]any{"tag": "loyalsoldier-proxy", "type": "local"},
				map[string]any{"tag": "loyalsoldier-reject", "type": "local"},
			},
		},
	})

	handler := NewConfigHandler(configPath, nil, nil, nil, core.NewDefaultRouteInstaller(), nil)

	rr := httptest.NewRecorder()
	handler.InstallDefaultRouteRules(rr, httptest.NewRequest(http.MethodPost, "/api/config/route/defaults", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("InstallDefaultRouteRules status = %d %s", rr.Code, rr.Body.String())
	}

	cfg := decodeConfigFile(t, configPath)
	rules := cfg["route"].(map[string]any)["rules"].([]any)
	if len(rules) != 8 {
		t.Fatalf("rules len = %d, want 8", len(rules))
	}
	first := rules[0].(map[string]any)
	if first["action"] != "sniff" {
		t.Fatalf("first route rule = %#v", first)
	}
}

func TestConfigHandlerInstallDefaultDNS(t *testing.T) {
	configPath := t.TempDir() + "/config.json"
	writeConfigFile(t, configPath, map[string]any{
		"outbounds": []any{
			map[string]any{"tag": "direct", "type": "direct"},
			map[string]any{"tag": "proxy", "type": "selector"},
		},
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "loyalsoldier-direct", "type": "local"},
				map[string]any{"tag": "loyalsoldier-proxy", "type": "local"},
				map[string]any{"tag": "loyalsoldier-reject", "type": "local"},
			},
		},
	})

	handler := NewConfigHandler(configPath, nil, nil, nil, nil, core.NewDefaultDNSInstaller())

	rr := httptest.NewRecorder()
	handler.InstallDefaultDNS(rr, httptest.NewRequest(http.MethodPost, "/api/config/dns/defaults", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("InstallDefaultDNS status = %d %s", rr.Code, rr.Body.String())
	}

	cfg := decodeConfigFile(t, configPath)
	dns := cfg["dns"].(map[string]any)
	if dns["final"] != "dns-remote" {
		t.Fatalf("dns.final = %#v", dns["final"])
	}
	servers := dns["servers"].([]any)
	if len(servers) != 5 {
		t.Fatalf("servers len = %d", len(servers))
	}
}

func TestImportAndSettingsHandlers(t *testing.T) {
	nodeMgr, subMgr, settingsMgr, configPath := newAPIManagers(t)
	importHandler := NewImportHandler(nodeMgr, subMgr, configPath)

	rr := httptest.NewRecorder()
	importHandler.ImportLink(rr, jsonRequest(http.MethodPost, "/api/import/link", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid import json status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	importHandler.ImportLink(rr, jsonRequest(http.MethodPost, "/api/import/link", `{"link":""}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty link status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	importHandler.ImportLink(rr, jsonRequest(http.MethodPost, "/api/import/link", `{"link":"unknown://x"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unsupported link status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	importHandler.ImportLink(rr, jsonRequest(http.MethodPost, "/api/import/link", `{"link":"trojan://pass@example.com:443#trojan-test"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("import link status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	importHandler.SaveNode(rr, jsonRequest(http.MethodPost, "/api/import/save", `{"tag":"n1","type":"vless","server":"1.2.3.4","port":443,"config":{"uuid":"u"}}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("save node status = %d", rr.Code)
	}
	if nodeMgr.Get("n1") == nil {
		t.Fatal("saved node not found")
	}

	rr = httptest.NewRecorder()
	importHandler.SaveNode(rr, jsonRequest(http.MethodPost, "/api/import/save", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("save invalid json status = %d", rr.Code)
	}

	settingsHandler := NewSettingsHandler(settingsMgr)
	rr = httptest.NewRecorder()
	settingsHandler.GetTestURL(rr, httptest.NewRequest(http.MethodGet, "/api/settings/url-test", nil))
	body := decodeBody[map[string]string](t, rr)
	if body["url"] != defaultTestURL {
		t.Fatalf("default url = %q", body["url"])
	}

	rr = httptest.NewRecorder()
	settingsHandler.SetTestURL(rr, jsonRequest(http.MethodPut, "/api/settings/url-test", `{"url":"http://example.test/"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("set url status = %d", rr.Code)
	}
	if got := settingsMgr.Get("url_test"); got != "http://example.test/" {
		t.Fatalf("stored url = %q", got)
	}

	rr = httptest.NewRecorder()
	settingsHandler.SetTestURL(rr, jsonRequest(http.MethodPut, "/api/settings/url-test", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid settings status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	settingsHandler.SetTestURL(rr, jsonRequest(http.MethodPut, "/api/settings/url-test", `{"url":""}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("empty settings status = %d", rr.Code)
	}
	if got := settingsMgr.Get("url_test"); got != defaultTestURL {
		t.Fatalf("empty URL should store default, got %q", got)
	}
}

func TestMountStaticServesAssetsAndFallback(t *testing.T) {
	router := chi.NewRouter()
	mountStatic(router, fstest.MapFS{
		"ui/dist/index.html":       {Data: []byte("index")},
		"ui/dist/assets/app.js":    {Data: []byte("app")},
		"ui/dist/assets/app.css":   {Data: []byte("css")},
		"ui/dist/assets/empty.txt": {Data: []byte{}},
	})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rr.Body.String() != "app" {
		t.Fatalf("asset body = %q", rr.Body.String())
	}
	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset Cache-Control = %q", got)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing asset status = %d, want 404", rr.Code)
	}
	if rr.Body.String() == "index" {
		t.Fatal("missing asset must not fall back to index")
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if rr.Body.String() != "index" {
		t.Fatalf("fallback body = %q", rr.Body.String())
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("fallback Cache-Control = %q", got)
	}
}

func TestMountStaticSkipsMissingIndex(t *testing.T) {
	router := chi.NewRouter()
	mountStatic(router, fstest.MapFS{})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}

	router = chi.NewRouter()
	mountStatic(router, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("nil static status = %d, want 404", rr.Code)
	}
}

func TestNewRouterHealthWorksWithoutStaticFiles(t *testing.T) {
	nodeMgr, subMgr, settingsMgr, configPath := newAPIManagers(t)
	logWriter := core.NewLogWriter(10)
	instance := core.NewSBInstance(configPath, logWriter)
	router := NewRouter(
		fstest.MapFS{},
		NewAuthHandler("admin", "pass", staticSecretProvider("secret-key-123456")),
		NewConfigHandler(configPath, nil, nil, nil, nil, nil),
		NewServiceHandler(instance),
		NewStatsHandler(logWriter, logWriter, instance),
		NewImportHandler(nodeMgr, subMgr, configPath),
		NewSubscriptionHandler(subMgr, nodeMgr, configPath),
		NewNodesHandler(nodeMgr, subMgr, configPath),
		NewTestHandler(func() string { return defaultTestURL }, nodeMgr, nil),
		NewSettingsHandler(settingsMgr),
		NewNetworkHandler(),
		NewKernelHandler("test-version"),
		NewRuntimeHandler(instance),
		settingsMgr,
		nil,
		instance,
	)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health status = %d", rr.Code)
	}
}

func fileMode(t *testing.T, path string) fs.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

type errorBody struct{}

func (errorBody) Read(p []byte) (int, error) {
	return 0, errors.New("read failed")
}

func (errorBody) Close() error {
	return nil
}
