package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoyalsoldierRuleSetInstaller(t *testing.T) {
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	if installer.client == nil || len(installer.sources) != 3 {
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
		client:     server.Client(),
		sources: []RuleSetSource{
			{Tag: "loyalsoldier-direct", FileName: "loyalsoldier-direct.json", URL: server.URL + "/direct-list.txt"},
			{Tag: "loyalsoldier-proxy", FileName: "loyalsoldier-proxy.json", URL: server.URL + "/proxy-list.txt"},
			{Tag: "loyalsoldier-reject", FileName: "loyalsoldier-reject.json", URL: server.URL + "/reject-list.txt"},
		},
	}

	entries, err := installer.Install(context.Background())
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	// 3 个本地规则集 + 4 个远程规则集
	if len(entries) != 7 {
		t.Fatalf("entries len = %d, want 7", len(entries))
	}

	data, err := os.ReadFile(filepath.Join(dir, "rule-sets", "loyalsoldier-direct.json"))
	if err != nil {
		t.Fatal(err)
	}
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

func TestRemoteRuleSetDefaultsContent(t *testing.T) {
	// 验证远程规则集条目结构完整，可被路由规则引用。
	wantTags := map[string]bool{
		"geosite-cn":               true,
		"geoip-cn":                 true,
		"geosite-google-play":      true,
		"geosite-category-ads-all": true,
	}
	if len(remoteRuleSetDefaults) != len(wantTags) {
		t.Fatalf("remoteRuleSetDefaults len = %d, want %d", len(remoteRuleSetDefaults), len(wantTags))
	}
	for _, entry := range remoteRuleSetDefaults {
		tag, _ := entry["tag"].(string)
		if !wantTags[tag] {
			t.Errorf("unexpected remote rule-set tag %q", tag)
		}
		if entry["type"] != "remote" {
			t.Errorf("%s type = %v, want remote", tag, entry["type"])
		}
		if entry["format"] != "binary" {
			t.Errorf("%s format = %v, want binary", tag, entry["format"])
		}
		if url, _ := entry["url"].(string); url == "" {
			t.Errorf("%s missing url", tag)
		}
		if entry["download_detour"] != "direct" {
			t.Errorf("%s download_detour = %v, want direct", tag, entry["download_detour"])
		}
		if entry["update_interval"] != "1d" {
			t.Errorf("%s update_interval = %v, want 1d", tag, entry["update_interval"])
		}
	}
}
