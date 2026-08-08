package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/model"
)

func TestRuleSetAutoUpdateSettings(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "boxd.db"), 0600, nil)
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
				map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": rulesetTestURL(server, "/geo.srs"), "download_detour": "direct"},
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
		client:     rulesetTestClient(server),
		sources: []RuleSetSource{
			{Tag: "loyalsoldier-direct", FileName: "loyalsoldier-direct.json", URL: rulesetTestURL(server, "/direct-list.txt")},
			{Tag: "loyalsoldier-proxy", FileName: "loyalsoldier-proxy.json", URL: rulesetTestURL(server, "/proxy-list.txt")},
			{Tag: "loyalsoldier-reject", FileName: "loyalsoldier-reject.json", URL: rulesetTestURL(server, "/reject-list.txt")},
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
	updater.client = rulesetTestClient(server)

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

func TestBuiltinLocalRuleSetTags(t *testing.T) {
	tags := BuiltinLocalRuleSetTags()
	if len(tags) != 7 {
		t.Fatalf("tags = %#v", tags)
	}
}

func TestRuleSetAutoUpdaterLifecycle(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "boxd.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	settings := NewSettingsManager(db)

	// disabled updater with nil RuleSetUpdater
	auto := NewRuleSetAutoUpdater(settings, nil)
	if cfg, err := auto.Config(); err != nil || cfg.Enabled {
		t.Fatalf("config = %#v err=%v", cfg, err)
	}
	auto.Start()
	auto.Start() // idempotent
	if d := auto.nextDelay(); d != time.Hour {
		t.Fatalf("disabled delay = %v", d)
	}
	auto.tick(context.Background()) // no-op when disabled / nil updater
	auto.Stop()
	auto.Stop() // idempotent

	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "2h"}); err != nil {
		t.Fatal(err)
	}
	if d := auto.nextDelay(); d != 2*time.Hour {
		t.Fatalf("enabled delay = %v", d)
	}
	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "30s"}); err != nil {
		t.Fatal(err)
	}
	if d := auto.nextDelay(); d != time.Minute {
		t.Fatalf("min delay = %v", d)
	}
	// invalid interval in store simulation: write raw bad value path via enabled empty interval fallback
	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "1h"}); err != nil {
		t.Fatal(err)
	}

	// With updater: tick should call Update for builtin local tags.
	dir := t.TempDir()
	ruleDir := filepath.Join(dir, "rule-sets")
	if err := os.MkdirAll(ruleDir, 0700); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("example.cn\n"))
	}))
	t.Cleanup(server.Close)
	configPath := filepath.Join(dir, "config.json")
	cfg := map[string]any{
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "loyalsoldier-direct", "type": "local", "format": "source", "path": filepath.Join(ruleDir, "loyalsoldier-direct.json")},
				map[string]any{"tag": "loyalsoldier-proxy", "type": "local", "format": "source", "path": filepath.Join(ruleDir, "loyalsoldier-proxy.json")},
				map[string]any{"tag": "loyalsoldier-reject", "type": "local", "format": "source", "path": filepath.Join(ruleDir, "loyalsoldier-reject.json")},
			},
		},
	}
	body, _ := json.Marshal(cfg)
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	installer := &LoyalsoldierRuleSetInstaller{
		ruleSetDir: ruleDir,
		client:     rulesetTestClient(server),
		sources: []RuleSetSource{
			{Tag: "loyalsoldier-direct", FileName: "loyalsoldier-direct.json", URL: rulesetTestURL(server, "/direct-list.txt")},
			{Tag: "loyalsoldier-proxy", FileName: "loyalsoldier-proxy.json", URL: rulesetTestURL(server, "/proxy-list.txt")},
			{Tag: "loyalsoldier-reject", FileName: "loyalsoldier-reject.json", URL: rulesetTestURL(server, "/reject-list.txt")},
		},
	}
	updater := NewRuleSetUpdater(configPath, dir, installer, nil, nil)
	updater.client = rulesetTestClient(server)
	auto = NewRuleSetAutoUpdater(settings, updater)
	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "1h"}); err != nil {
		t.Fatal(err)
	}
	auto.tick(context.Background())
	// files may be created by local update path
	auto.Start()
	// allow loop to arm timer; then stop quickly
	time.Sleep(20 * time.Millisecond)
	auto.Stop()
}

func TestRuleSetUpdaterLoadConfigErrors(t *testing.T) {
	updater := NewRuleSetUpdater(filepath.Join(t.TempDir(), "missing.json"), t.TempDir(), NewLoyalsoldierRuleSetInstaller(t.TempDir()), nil, nil)
	if _, err := updater.Status(context.Background()); err == nil {
		t.Fatal("expected missing config error")
	}
	if _, err := updater.Update(context.Background(), RuleSetUpdateRequest{}); err == nil {
		t.Fatal("expected missing config error on update")
	}
}

func TestRuleSetUpdaterStopStartErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Etag", `"e1"`)
		_, _ = w.Write([]byte("bin"))
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	cfg := map[string]any{
		"route": map[string]any{
			"rule_set": []any{
				map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": rulesetTestURL(server, "/geo.srs")},
			},
		},
	}
	body, _ := json.Marshal(cfg)
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), func() error {
		return errTest("stop failed")
	}, func() error { return nil })
	updater.client = rulesetTestClient(server)
	if _, err := updater.Update(context.Background(), RuleSetUpdateRequest{}); err == nil {
		t.Fatal("expected stop error")
	}

	updater = NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), func() error { return nil }, func() error {
		return errTest("start failed")
	})
	updater.client = rulesetTestClient(server)
	if _, err := updater.Update(context.Background(), RuleSetUpdateRequest{}); err == nil {
		t.Fatal("expected start error")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestRuleSetUpdaterRemoteEdgeCases(t *testing.T) {
	dir := t.TempDir()
	// empty url
	configPath := filepath.Join(dir, "empty-url.json")
	body, _ := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": ""},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	result, err := updater.Update(context.Background(), RuleSetUpdateRequest{Tags: []string{"geo"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.FailedCount != 1 {
		t.Fatalf("empty url result = %#v", result)
	}
	if result.Results[0].ErrorCode != RuleSetErrorInvalidURL {
		t.Fatalf("empty url code = %#v", result.Results[0])
	}

	// unexpected status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(server.Close)
	configPath = filepath.Join(dir, "bad-status.json")
	body, _ = json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": rulesetTestURL(server, "/x.srs")},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater = NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	updater.client = rulesetTestClient(server)
	result, err = updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FailedCount != 1 {
		t.Fatalf("bad status result = %#v", result)
	}
	if result.Results[0].ErrorCode != RuleSetErrorHTTP {
		t.Fatalf("bad status code = %#v", result.Results[0])
	}

	// empty body
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	configPath = filepath.Join(dir, "empty-body.json")
	body, _ = json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": rulesetTestURL(server, "/x.srs")},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater = NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	updater.client = rulesetTestClient(server)
	result, err = updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FailedCount != 1 {
		t.Fatalf("empty body result = %#v", result)
	}
	if result.Results[0].ErrorCode != RuleSetErrorEmpty {
		t.Fatalf("empty body code = %#v", result.Results[0])
	}
}

func TestRuleSetUpdaterRemoteSecurityLimits(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cfg.json")
	writeRuleSetConfig := func(t *testing.T, rawURL string) {
		t.Helper()
		body, err := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
			map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": rawURL},
		}}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, body, 0600); err != nil {
			t.Fatal(err)
		}
	}

	requestCount := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    req,
		}, nil
	})}
	writeRuleSetConfig(t, "http://127.0.0.1:8080/rules.srs")
	stopCount := 0
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), func() error {
		stopCount++
		return nil
	}, nil)
	updater.client = client
	result, err := updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil || result.FailedCount != 1 || result.Results[0].ErrorCode != RuleSetErrorBlockedURL {
		t.Fatalf("blocked result = %#v err=%v", result, err)
	}
	if requestCount != 0 {
		t.Fatalf("blocked request count = %d, want 0", requestCount)
	}
	if stopCount != 0 {
		t.Fatalf("blocked rule-set stop count = %d, want 0", stopCount)
	}

	writeRuleSetConfig(t, "https://ruleset.test/large.srs")
	client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			ContentLength: maxRuleSetBodyBytes + 1,
			Body:          io.NopCloser(strings.NewReader("unused")),
			Request:       req,
		}, nil
	})}
	updater.client = client
	result, err = updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil || result.FailedCount != 1 || result.Results[0].ErrorCode != RuleSetErrorTooLarge {
		t.Fatalf("large result = %#v err=%v", result, err)
	}

	client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", maxRuleSetBodyBytes+1))),
			Request:    req,
		}, nil
	})}
	updater.client = client
	result, err = updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil || result.FailedCount != 1 || result.Results[0].ErrorCode != RuleSetErrorTooLarge {
		t.Fatalf("streaming large result = %#v err=%v", result, err)
	}
}

func TestRuleSetUpdaterHandlesNilHTTPResponses(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cfg.json")
	body, err := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "geo", "type": "remote", "url": "https://ruleset.test/rules.srs"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	updater.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	})}
	result, err := updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil || result.FailedCount != 1 || result.Results[0].Error == "" {
		t.Fatalf("nil response result = %#v err=%v", result, err)
	}
}

func TestRuleSetUpdaterInvalidConfigJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(configPath, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	if _, err := updater.Status(context.Background()); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestRuleSetUpdaterStatusBuiltinRemote(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cfg.json")
	body, _ := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "geosite-cn", "type": "remote", "format": "binary", "url": "https://example.com/a.srs"},
		map[string]any{"tag": "custom", "type": "local", "path": filepath.Join(dir, "missing.json")},
		map[string]any{"tag": "no-type"},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	// create local file for custom to hit modtime branch for non-builtin
	if err := os.WriteFile(filepath.Join(dir, "missing.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	// fix path in config to existing
	body, _ = json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "geosite-cn", "type": "remote", "format": "binary", "url": "https://example.com/a.srs"},
		map[string]any{"tag": "custom", "type": "local", "path": filepath.Join(dir, "missing.json")},
		map[string]any{"tag": "no-type"},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	status, err := updater.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 3 {
		t.Fatalf("status = %#v", status)
	}
}

func TestAtomicWriteFile0600AndRuleSetDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "file.json")
	if err := atomicWriteFile0600(path, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != `{"a":1}` {
		t.Fatalf("data = %q err=%v", data, err)
	}
	installer := NewLoyalsoldierRuleSetInstaller(dir)
	if installer.RuleSetDir() == "" {
		t.Fatal("empty rule set dir")
	}
	if !installer.IsBuiltinLocal("loyalsoldier-direct") || installer.IsBuiltinLocal("nope") {
		t.Fatal("builtin detection failed")
	}
	if _, ok := installer.SourceByTag("loyalsoldier-proxy"); !ok {
		t.Fatal("source missing")
	}
}

func TestNewRuleSetUpdaterNilInstaller(t *testing.T) {
	updater := NewRuleSetUpdater(filepath.Join(t.TempDir(), "c.json"), t.TempDir(), nil, nil, nil)
	if updater.installer == nil || updater.client == nil {
		t.Fatalf("updater = %#v", updater)
	}
}

func TestSavedRuleSetBinaryErrorsAndCacheHelpers(t *testing.T) {
	var saved savedRuleSetBinary
	if err := saved.UnmarshalBinary(nil); err == nil {
		t.Fatal("expected short binary error")
	}
	if err := saved.UnmarshalBinary([]byte{1}); err == nil {
		t.Fatal("expected truncated content length error")
	}

	// valid roundtrip then corrupt tail
	ok := &savedRuleSetBinary{Content: []byte("x"), LastUpdated: time.Unix(10, 0), LastEtag: "e"}
	data, err := ok.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := saved.UnmarshalBinary(data[:len(data)-1]); err == nil {
		t.Fatal("expected truncated etag error")
	}
	var oversized bytes.Buffer
	if err := oversized.WriteByte(1); err != nil {
		t.Fatal(err)
	}
	if err := writeUvarint(&oversized, maxRuleSetBodyBytes+1); err != nil {
		t.Fatal(err)
	}
	if err := saved.UnmarshalBinary(oversized.Bytes()); !errors.Is(err, ErrRuleSetContentTooLarge) {
		t.Fatalf("oversized cache error = %v", err)
	}
	if err := saved.UnmarshalBinary([]byte{2}); err == nil {
		t.Fatal("expected unsupported cache version error")
	}
	if _, err := (&savedRuleSetBinary{Content: make([]byte, maxRuleSetBodyBytes+1)}).MarshalBinary(); !errors.Is(err, ErrRuleSetContentTooLarge) {
		t.Fatalf("oversized marshal error = %v", err)
	}

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.db")
	updater := NewRuleSetUpdater(filepath.Join(dir, "c.json"), dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	// open read-only missing
	if _, err := updater.openCacheReadOnly(); err == nil {
		t.Fatal("expected missing cache error")
	}
	// write cache entry then load
	if err := updater.saveRemoteCache("geo", []byte("bin"), "etag", time.Unix(20, 0)); err != nil {
		t.Fatal(err)
	}
	db, err := bbolt.Open(cachePath, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := loadRuleSetCache(db, "missing"); got != nil {
		_ = db.Close()
		t.Fatalf("missing tag = %#v", got)
	}
	if got := loadRuleSetCache(db, "geo"); got == nil || string(got.Content) != "bin" || got.LastEtag != "etag" {
		_ = db.Close()
		t.Fatalf("loaded = %#v", got)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := updater.touchRemoteCache("geo", time.Unix(30, 0)); err != nil {
		t.Fatal(err)
	}
	if err := updater.touchRemoteCache("missing", time.Unix(30, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestAtomicWriteFile0600Errors(t *testing.T) {
	dir := t.TempDir()
	// parent path is a file, mkdir should fail
	parent := filepath.Join(dir, "file")
	if err := os.WriteFile(parent, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile0600(filepath.Join(parent, "child.json"), []byte("{}")); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestValidateRuleSetAutoUpdateEmptyInterval(t *testing.T) {
	if err := ValidateRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: ""}); err == nil {
		t.Fatal("expected empty interval error")
	}
	if err := ValidateRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "24h"}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCacheReadWriteMkdirError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(filepath.Join(dir, "c.json"), dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	updater.cachePath = filepath.Join(blocker, "cache.db")
	if _, err := updater.openCacheReadWrite(); err == nil {
		t.Fatal("expected mkdir error")
	}
	if err := updater.saveRemoteCache("t", []byte("x"), "e", time.Unix(1, 0)); err == nil {
		t.Fatal("expected save cache error")
	}
}

func TestRuleSetEntriesAndSelectFilters(t *testing.T) {
	entries := ruleSetEntries(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "a", "type": "local"},
		map[string]any{"tag": "b", "type": "remote"},
		"ignore",
	}}})
	if len(entries) != 2 {
		t.Fatalf("entries = %#v", entries)
	}
	selected := selectRuleSets(entries, RuleSetUpdateRequest{Tags: []string{"b"}, Types: []string{"remote"}})
	if len(selected) != 1 || stringValue(selected[0]["tag"]) != "b" {
		t.Fatalf("selected = %#v", selected)
	}
	selected = selectRuleSets(entries, RuleSetUpdateRequest{Types: []string{"local"}})
	if len(selected) != 1 {
		t.Fatalf("local selected = %#v", selected)
	}
	// null cfg route
	if got := ruleSetEntries(nil); len(got) != 0 {
		t.Fatalf("nil cfg = %#v", got)
	}
	if got := ruleSetEntries(map[string]any{"route": "bad"}); len(got) != 0 {
		t.Fatalf("bad route = %#v", got)
	}
}

func TestUpdateLocalNonBuiltinAndFetchError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cfg.json")
	body, _ := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "custom-local", "type": "local", "path": filepath.Join(dir, "c.json")},
		map[string]any{"tag": "loyalsoldier-direct", "type": "local", "path": filepath.Join(dir, "d.json")},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	// installer with unreachable source URL for builtin
	installer := &LoyalsoldierRuleSetInstaller{
		ruleSetDir: dir,
		client:     &http.Client{Timeout: time.Millisecond},
		sources: []RuleSetSource{
			{Tag: "loyalsoldier-direct", FileName: "loyalsoldier-direct.json", URL: "http://127.0.0.1:1/direct-list.txt"},
		},
	}
	updater := NewRuleSetUpdater(configPath, dir, installer, nil, nil)
	result, err := updater.Update(context.Background(), RuleSetUpdateRequest{Tags: []string{"custom-local", "loyalsoldier-direct"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.SkippedCount < 1 || result.FailedCount < 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestUpdateRemoteCacheDisabled(t *testing.T) {
	dir := t.TempDir()
	// make cache path a regular file so openCacheReadWrite fails for mkdir? set cachePath to under file
	blocker := filepath.Join(dir, "block")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload-bytes"))
	}))
	t.Cleanup(server.Close)
	configPath := filepath.Join(dir, "cfg.json")
	body, _ := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "geo", "type": "remote", "format": "binary", "url": rulesetTestURL(server, "/a.srs")},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	updater.client = rulesetTestClient(server)
	updater.cachePath = filepath.Join(blocker, "cache.db")
	result, err := updater.Update(context.Background(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FailedCount != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestLoadConfigNullObject(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "null.json")
	if err := os.WriteFile(configPath, []byte("null"), 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	cfg, err := updater.loadConfig()
	if err != nil || cfg == nil {
		t.Fatalf("cfg=%#v err=%v", cfg, err)
	}
}

func TestRuleSetAutoUpdateCorruptJSON(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "boxd.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	settings := NewSettingsManager(db)
	if err := settings.Set(ruleSetAutoUpdateKey, "{"); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.RuleSetAutoUpdate(); err == nil {
		t.Fatal("expected decode error")
	}
	// empty interval normalization on read
	if err := settings.Set(ruleSetAutoUpdateKey, `{"enabled":true,"interval":""}`); err != nil {
		t.Fatal(err)
	}
	cfg, err := settings.RuleSetAutoUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != defaultRuleSetAutoInterval {
		t.Fatalf("interval = %q", cfg.Interval)
	}
}

func TestRuleSetAutoUpdaterTickError(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "boxd.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	settings := NewSettingsManager(db)
	if err := settings.SetRuleSetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "1h"}); err != nil {
		t.Fatal(err)
	}
	// updater with missing config => Update error => warn branch
	updater := NewRuleSetUpdater(filepath.Join(t.TempDir(), "missing.json"), t.TempDir(), NewLoyalsoldierRuleSetInstaller(t.TempDir()), nil, nil)
	auto := NewRuleSetAutoUpdater(settings, updater)
	auto.tick(context.Background())
}

func TestStatusRemoteWithoutIntervalUsesDefault(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cfg.json")
	// remote without update_interval, with cache content
	body, _ := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "custom-remote", "type": "remote", "format": "binary", "url": "https://example.com/x.srs"},
	}}})
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	if err := updater.saveRemoteCache("custom-remote", []byte("abc"), "e", time.Unix(40, 0)); err != nil {
		t.Fatal(err)
	}
	status, err := updater.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 || status[0].UpdateInterval != DefaultRemoteRuleSetInterval() || status[0].Builtin {
		t.Fatalf("status = %#v", status)
	}
	if status[0].LastEtag != "e" || status[0].FileSize != 3 {
		t.Fatalf("cache fields = %#v", status[0])
	}
}

func TestRuleSetStatusMarksBlockedRemoteURL(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cfg.json")
	body, err := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"tag": "private", "type": "remote", "url": "http://127.0.0.1/rules.srs"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	updater := NewRuleSetUpdater(configPath, dir, NewLoyalsoldierRuleSetInstaller(dir), nil, nil)
	status, err := updater.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 || status[0].Updatable || status[0].Note == "" {
		t.Fatalf("status = %#v", status)
	}
}
