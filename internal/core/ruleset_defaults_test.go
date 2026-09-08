package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewLoyalsoldierRuleSetInstaller(t *testing.T) {
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	if installer.client == nil || len(installer.sources) != 4 {
		t.Fatalf("installer = %#v", installer)
	}
}

func TestLoyalsoldierRuleSetInstallerInstall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case "direct-list.txt":
			_, _ = w.Write([]byte("# comment\nexample.cn\nfull:exact.example.cn\n"))
		case "proxy-list.txt":
			_, _ = w.Write([]byte("proxy.example\nkeyword:google\n"))
		case "reject-list.txt":
			_, _ = w.Write([]byte("ads.example\nregexp:^ad\\.\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	installer := &LoyalsoldierRuleSetInstaller{
		ruleSetDir: filepath.Join(dir, "rule-sets"),
		client:     rulesetTestClient(server),
		sources: []RuleSetSource{
			{Tag: "loyalsoldier-direct", FileName: "loyalsoldier-direct.json", URL: rulesetTestURL(server, "/direct-list.txt")},
			{Tag: "loyalsoldier-proxy", FileName: "loyalsoldier-proxy.json", URL: rulesetTestURL(server, "/proxy-list.txt")},
			{Tag: "loyalsoldier-reject", FileName: "loyalsoldier-reject.json", URL: rulesetTestURL(server, "/reject-list.txt")},
		},
	}

	entries, err := installer.Install(context.Background())
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	// 手工 installer 只含 3 个文本源，无内置二进制源
	if len(entries) != 3 {
		t.Fatalf("entries len = %d, want 3", len(entries))
	}

	data, err := os.ReadFile(filepath.Join(dir, "rule-sets", "loyalsoldier-direct.json"))
	if err != nil {
		t.Fatal(err)
	}
	assertInstalledSourceRuleSet(t, data)
}

func assertInstalledSourceRuleSet(t *testing.T, data []byte) {
	t.Helper()
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["version"] == nil {
		t.Fatalf("missing version in output: %s", string(data))
	}
	rules, ok := parsed["rules"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("rules = %#v", parsed["rules"])
	}
	firstRule := rules[0].(map[string]any)
	if firstRule["domain_suffix"] == nil || firstRule["domain"] == nil {
		t.Fatalf("direct rule missing expected fields: %#v", firstRule)
	}
}

func TestBuiltinBinaryRuleSetSources(t *testing.T) {
	sources := builtinRuleSetSources()
	if len(sources) != 7 {
		t.Fatalf("builtin sources = %d, want 7", len(sources))
	}
	binaryTags := map[string]string{
		"geosite-cn":               "geosite-cn.srs",
		"geosite-google-play":      "geosite-google-play.srs",
		"geosite-category-ads-all": "geosite-category-ads-all.srs",
	}
	found := 0
	for _, src := range sources {
		if src.Format != "binary" {
			continue
		}
		found++
		if wantFile := binaryTags[src.Tag]; wantFile == "" || src.FileName != wantFile {
			t.Errorf("%s file = %q, want %q", src.Tag, src.FileName, wantFile)
		}
		if !strings.HasPrefix(src.URL, "https://raw.githubusercontent.com/SagerNet/") {
			t.Errorf("%s URL = %q, want SagerNet raw URL", src.Tag, src.URL)
		}
	}
	if found != len(binaryTags) {
		t.Fatalf("binary sources = %d, want %d", found, len(binaryTags))
	}
	convertedTags := map[string]bool{
		"loyalsoldier-direct": true, "loyalsoldier-proxy": true, "loyalsoldier-reject": true, "geoip-cn": true,
	}
	for _, tag := range BuiltinLocalRuleSetTags() {
		if !convertedTags[tag] && binaryTags[tag] == "" {
			t.Errorf("unexpected builtin tag %q", tag)
		}
	}
	if len(BuiltinLocalRuleSetTags()) != 7 {
		t.Fatalf("BuiltinLocalRuleSetTags = %d, want 7", len(BuiltinLocalRuleSetTags()))
	}
}

func TestLoyalsoldierRuleSetInstallerInstallsBinarySources(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls[r.URL.Path]++
		mu.Unlock()
		switch filepath.Base(r.URL.Path) {
		case "direct-list.txt":
			_, _ = w.Write([]byte("# comment\nexample.cn\n"))
		case "geo.srs":
			_, _ = w.Write([]byte("srs-bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	installer := &LoyalsoldierRuleSetInstaller{
		ruleSetDir: filepath.Join(dir, "rule-sets"),
		client:     rulesetTestClient(server),
		sources: []RuleSetSource{
			{Tag: "loyalsoldier-direct", FileName: "loyalsoldier-direct.json", URL: rulesetTestURL(server, "/direct-list.txt")},
			{Tag: "geo", FileName: "geo.srs", URL: rulesetTestURL(server, "/geo.srs"), Format: "binary"},
		},
	}
	entries, err := installer.Install(context.Background())
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0]["format"] != "source" || entries[1]["format"] != "binary" || entries[1]["type"] != "local" {
		t.Fatalf("entries = %#v", entries)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "rule-sets", "geo.srs"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "srs-bytes" {
		t.Fatalf("srs content = %q", string(raw))
	}
	if calls["/geo.srs"] != 1 {
		t.Fatalf("expected single binary download, calls = %#v", calls)
	}
}
