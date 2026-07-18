package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxui/internal/model"
)

func TestRuleSetAutoUpdateSettings(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "boxui.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	settings := NewSettingsManager(db)

	cfg, err := settings.RuleSetAutoUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || cfg.Interval != defaultRuleSetAutoInterval {
		t.Fatalf("defaults = %#v", cfg)
	}
	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "bad"}); err == nil {
		t.Fatal("expected validation error")
	}
	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "2h"}); err != nil {
		t.Fatal(err)
	}
	cfg, err = settings.RuleSetAutoUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.Interval != "2h" {
		t.Fatalf("saved = %#v", cfg)
	}
}

func TestRuleSetUpdaterLocalAndRemote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case "direct-list.txt":
			_, _ = w.Write([]byte("example.cn\nfull:exact.example.cn\n"))
		case "proxy-list.txt":
			_, _ = w.Write([]byte("proxy.example\n"))
		case "reject-list.txt":
			_, _ = w.Write([]byte("ads.example\n"))
		case "geo.srs":
			if r.Header.Get("If-None-Match") == `"etag-1"` {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("Etag", `"etag-1"`)
			_, _ = w.Write([]byte("srs-binary-content"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	ruleDir := filepath.Join(dir, "rule-sets")
	if err := os.MkdirAll(ruleDir, 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.json")
	cfg := map[string]any{
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "loyalsoldier-direct", "type": "local", "format": "source", "path": filepath.Join(ruleDir, "loyalsoldier-direct.json")},
				map[string]any{"tag": "custom-local", "type": "local", "format": "source", "path": filepath.Join(ruleDir, "custom.json")},
				map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": server.URL + "/geo.srs", "download_detour": "direct"},
				map[string]any{"tag": "inline", "type": "inline", "rules": []any{}},
			},
		},
	}
	body, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}

	installer := &LoyalsoldierRuleSetInstaller{
		ruleSetDir: ruleDir,
		client:     server.Client(),
		sources: []RuleSetSource{
			{Tag: "loyalsoldier-direct", FileName: "loyalsoldier-direct.json", URL: server.URL + "/direct-list.txt"},
			{Tag: "loyalsoldier-proxy", FileName: "loyalsoldier-proxy.json", URL: server.URL + "/proxy-list.txt"},
			{Tag: "loyalsoldier-reject", FileName: "loyalsoldier-reject.json", URL: server.URL + "/reject-list.txt"},
		},
	}
	stopped := 0
	started := 0
	updater := NewRuleSetUpdater(configPath, dir, installer, func() error {
		stopped++
		return nil
	}, func() error {
		started++
		return nil
	})
	updater.client = server.Client()

	status, err := updater.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 4 {
		t.Fatalf("status len = %d", len(status))
	}

	result, err := updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdatedCount < 2 {
		t.Fatalf("updated = %#v", result)
	}
	if stopped != 1 || started != 1 {
		t.Fatalf("stop/start = %d/%d, want 1/1", stopped, started)
	}
	if _, err := os.Stat(filepath.Join(ruleDir, "loyalsoldier-direct.json")); err != nil {
		t.Fatal(err)
	}

	// second remote update should hit not-modified
	result, err = updater.Update(context.Background(), RuleSetUpdateRequest{Tags: []string{"geo"}, Types: []string{"remote"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 || !result.Results[0].OK || !result.Results[0].NotModified {
		t.Fatalf("second remote update = %#v", result.Results)
	}

	// custom local skipped
	result, err = updater.Update(context.Background(), RuleSetUpdateRequest{Tags: []string{"custom-local"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.SkippedCount != 1 {
		t.Fatalf("custom local result = %#v", result)
	}
}

func TestSavedRuleSetBinaryRoundTrip(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	saved := &savedRuleSetBinary{Content: []byte("abc"), LastUpdated: now, LastEtag: "etag"}
	data, err := saved.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var decoded savedRuleSetBinary
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if string(decoded.Content) != "abc" || decoded.LastEtag != "etag" || !decoded.LastUpdated.Equal(now) {
		t.Fatalf("decoded = %#v", decoded)
	}
}
