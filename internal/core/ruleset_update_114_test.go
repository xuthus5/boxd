package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

func TestConfiguredRuleSetTagExpansion(t *testing.T) {
	entry := map[string]any{"type": "remote", "tag": []any{"a", "b", "a", ""}, "url": "https://example.org/{tag}.srs", "initial_path": "/tmp/{tag}.srs"}
	expanded := expandConfiguredRuleSetTags(entry)
	if len(expanded) != 2 || expanded[1]["tag"] != "b" || expanded[1]["url"] != "https://example.org/b.srs" || expanded[0]["initial_path"] != "/tmp/a.srs" {
		t.Fatalf("expanded entries = %#v", expanded)
	}
	if !reflect.DeepEqual([]any{"a", "b", "a", ""}, entry["tag"]) || entry["url"] != "https://example.org/{tag}.srs" {
		t.Fatalf("configuration mutated: %#v", entry)
	}
	entries := ruleSetEntries(map[string]any{"route": map[string]any{"rule_set": []any{entry, map[string]any{"tag": "b", "type": "local", "path": "ignored"}}}})
	if len(entries) != 2 {
		t.Fatalf("duplicate tag should not update twice: %#v", entries)
	}
	selected := selectRuleSets(entries, RuleSetUpdateRequest{Tags: []string{"b", "b"}})
	if len(selected) != 1 || selected[0]["tag"] != "b" {
		t.Fatalf("selected entries = %#v", selected)
	}
}

func TestRuleSetHTTPClientPrecedenceAndDefaultOutbound(t *testing.T) {
	cases := []struct {
		body, source, tag string
		managed           bool
	}{
		{`{"entry":{"http_client":"explicit"},"route":{"default_http_client":"default"}}`, "rule_set", "explicit", true},
		{`{"entry":{"http_client":{}},"route":{"default_http_client":"default"}}`, "inline", "", true},
		{`{"entry":{"http_client":{"headers":{"Authorization":"secret"}}}}`, "inline", "", true},
		{`{"entry":{"download_detour":"proxy"}}`, "legacy_detour", "proxy", true},
		{`{"route":{"default_http_client":"default"},"http_clients":[{"tag":"first"}]}`, "route_default", "default", true},
		{`{"http_clients":[{"tag":"first"},{"tag":"second"}]}`, "global_default", "first", true},
		{`{"outbounds":[{"type":"socks","tag":"proxy"}],"route":{"final":"proxy"}}`, "default_outbound", "proxy", true},
		{`{"outbounds":[{"type":"direct","tag":"direct","bind_interface":"eth0"}],"route":{"final":"direct"}}`, "default_outbound", "direct", true},
		{`{"route":{"final":"missing"}}`, "default_outbound", "missing", true},
		{`{"outbounds":[{"type":"direct","tag":"direct"}]}`, "", "", false},
		{`{"endpoints":[{"type":"wireguard","tag":"vpn"}]}`, "", "", false},
		{`{}`, "", "", false},
	}
	for _, test := range cases {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(test.body), &cfg); err != nil {
			t.Fatal(err)
		}
		entry := objectValue(cfg["entry"])
		if entry == nil {
			entry = make(map[string]any)
		}
		entry["type"] = "remote"
		policy := configuredRuleSetHTTPPolicy(cfg, entry)
		if policy.source != test.source || policy.tag != test.tag || policy.managed != test.managed {
			t.Fatalf("policy = %+v, want source=%s tag=%s managed=%v", policy, test.source, test.tag, test.managed)
		}
	}
}

func TestKernelManagedRuleSetsNeverDownloadOrRestart(t *testing.T) {
	for _, client := range []any{"shared", map[string]any{}, map[string]any{"headers": map[string]any{"Authorization": "private-client-credential"}}} {
		cfg := map[string]any{"route": map[string]any{"rule_set": []any{map[string]any{"type": "remote", "tag": "remote", "url": "https://example.org/remote.srs", "http_client": client}}}}
		updater := newRuleSet114Updater(t, cfg)
		updater.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("kernel client must not be bypassed")
			return nil, errors.New("unexpected request")
		})}
		updater.stop = func() error { t.Fatal("managed update must not stop kernel"); return nil }
		updater.start = func() error { t.Fatal("managed update must not start kernel"); return nil }
		status, err := updater.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(status) != 1 || status[0].Updatable || !status[0].KernelManaged || status[0].NoteCode != RuleSetErrorKernelManaged {
			t.Fatalf("managed status = %#v", status)
		}
		serialized, err := json.Marshal(status)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(serialized), "private-client-credential") {
			t.Fatal("client credential leaked into status")
		}
		result, err := updater.Update(t.Context(), RuleSetUpdateRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if result.UpdatedCount != 0 || result.SkippedCount != 1 || result.FailedCount != 0 || result.Restarted || result.Results[0].ErrorCode != RuleSetErrorKernelManaged {
			t.Fatalf("managed result = %#v", result)
		}
	}
}

func TestMultiTagRuleSetUpdatesEachExpandedResourceOnce(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("rules for " + r.URL.Path))
	}))
	t.Cleanup(server.Close)
	cfg := map[string]any{"route": map[string]any{"rule_set": []any{map[string]any{
		"type": "remote", "tag": []any{"a", "b", "a"}, "url": rulesetTestURL(server, "/{tag}.srs"),
	}}}}
	updater := newRuleSet114Updater(t, cfg)
	updater.client = rulesetTestClient(server)
	status, err := updater.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 2 || status[0].Tag != "a" || status[1].URL != rulesetTestURL(server, "/b.srs") {
		t.Fatalf("status = %#v", status)
	}
	result, err := updater.Update(t.Context(), RuleSetUpdateRequest{Tags: []string{"a", "a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdatedCount != 2 || requests.Load() != 2 {
		t.Fatalf("result=%#v requests=%d", result, requests.Load())
	}
}

func TestDuplicateRemoteResourcesShareDownloadAndPopulateEachCacheTag(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("shared data"))
	}))
	t.Cleanup(server.Close)
	url := rulesetTestURL(server, "/shared.srs")
	cfg := map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"type": "remote", "tag": "a", "url": url}, map[string]any{"type": "remote", "tag": "b", "url": url},
	}}}
	updater := newRuleSet114Updater(t, cfg)
	updater.client = rulesetTestClient(server)
	result, err := updater.Update(t.Context(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdatedCount != 1 || requests.Load() != 1 || len(result.Results) != 2 || !result.Results[1].NotModified {
		t.Fatalf("result=%#v requests=%d", result, requests.Load())
	}
	cache, err := updater.openCacheReadOnly()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cache.Close() })
	for _, tag := range []string{"a", "b"} {
		if saved := loadRuleSetCacheForURL(cache, tag, url); saved == nil || string(saved.Content) != "shared data" {
			t.Fatalf("cache tag %s = %#v", tag, saved)
		}
	}
}

func TestEmptyInlineHTTPClientIsExplicitInSingBox114(t *testing.T) {
	var client option.HTTPClientOptions
	if err := json.Unmarshal([]byte(`{}`), &client); err != nil {
		t.Fatal(err)
	}
	if client.IsEmpty() || client.Version != 2 {
		t.Fatalf("empty JSON client must select native defaults: %+v", client)
	}
}

func newRuleSet114Updater(t *testing.T, cfg map[string]any) *RuleSetUpdater {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	return NewRuleSetUpdater(path, dir, nil, nil, nil)
}

func TestRuleSetBatchFailuresAndLocalReuse(t *testing.T) {
	updater := newRuleSet114Updater(t, map[string]any{})
	batch := ruleSetUpdateBatch{updater: updater, results: make(map[string]model.RuleSetUpdateResult)}
	entry := map[string]any{"type": "local", "tag": "custom", "path": "/tmp/shared.json"}
	first := batch.update(context.Background(), entry)
	second := batch.update(context.Background(), entry)
	if first.ErrorCode != second.ErrorCode || second.OK {
		t.Fatalf("reused failure = %#v", second)
	}
}
