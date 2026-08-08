package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestUniqueStringsAndFetchErrors(t *testing.T) {
	got := uniqueStrings([]string{"a", "a", "", "b", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got = %#v", got)
	}
	installer := &LoyalsoldierRuleSetInstaller{client: http.DefaultClient, ruleSetDir: t.TempDir()}
	if _, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "x", URL: "://bad"}); err == nil {
		t.Fatal("expected bad url error")
	}
}

func TestRuleSetInstallerRejectsOversizedSource(t *testing.T) {
	installer := &LoyalsoldierRuleSetInstaller{
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: maxRuleSetBodyBytes + 1,
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader("unused")),
				Request:       req,
			}, nil
		})},
		ruleSetDir: t.TempDir(),
	}
	_, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "large", URL: "https://ruleset.test/source.txt"})
	if !errors.Is(err, ErrRuleSetContentTooLarge) {
		t.Fatalf("error = %v, want oversized error", err)
	}
}

func TestJsdelivrMirrorURLs(t *testing.T) {
	rawURL := "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"
	got := jsdelivrMirrorURLs(rawURL)
	want := []string{
		"https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/direct-list.txt",
		"https://fastly.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/direct-list.txt",
	}
	if len(got) != len(want) {
		t.Fatalf("mirrors = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mirror[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if jsdelivrMirrorURLs("https://example.com/a.txt") != nil {
		t.Fatal("non raw.githubusercontent.com URL must have no mirrors")
	}
	if jsdelivrMirrorURLs("https://raw.githubusercontent.com/owner/repo/branch") != nil {
		t.Fatal("URL without file path must have no mirrors")
	}
}

func TestRuleSetSourceURLsOrder(t *testing.T) {
	primary := "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/proxy-list.txt"
	urls := ruleSetSourceURLs(primary)
	wantLen := 3
	if len(urls) != wantLen {
		t.Fatalf("urls = %#v, want 3 entries", urls)
	}
	if urls[0] != primary {
		t.Fatalf("primary = %q, want %q", urls[0], primary)
	}
	if got, want := urls[1], "https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/proxy-list.txt"; got != want {
		t.Fatalf("mirror[1] = %q, want %q", got, want)
	}
	if noMirror := ruleSetSourceURLs("https://example.com/feed"); len(noMirror) != 1 {
		t.Fatalf("non-raw URL should keep single source, got %#v", noMirror)
	}
}

func TestFetchAndConvertUsesPrimaryWhenHealthy(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "raw.githubusercontent.com" {
			return nil, fmt.Errorf("unexpected host %s", req.URL.Host)
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("# c\nexample.cn\n")),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	rule, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "t", URL: "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"})
	if err != nil {
		t.Fatalf("fetchAndConvert() error = %v", err)
	}
	if len(rule.Rules) != 1 || len(rule.Rules[0].DomainSuffix) != 1 || rule.Rules[0].DomainSuffix[0] != "example.cn" {
		t.Fatalf("rule = %#v", rule.Rules)
	}
}

func TestFetchAndConvertFallsBackToMirror(t *testing.T) {
	var calls []string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.URL.String())
		if req.URL.Host != "cdn.jsdelivr.net" {
			return &http.Response{
				StatusCode:    http.StatusServiceUnavailable,
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: -1,
				Request:       req,
			}, nil
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("full:exact.example\n")),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	rule, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "t", URL: "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"})
	if err != nil {
		t.Fatalf("fetchAndConvert() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2 (primary then mirror), urls = %#v", len(calls), calls)
	}
	if !strings.HasPrefix(calls[1], "https://cdn.jsdelivr.net/") {
		t.Fatalf("second call = %q, want jsdelivr mirror", calls[1])
	}
	if len(rule.Rules[0].Domain) != 1 || rule.Rules[0].Domain[0] != "exact.example" {
		t.Fatalf("rule = %#v", rule.Rules)
	}
}

func TestFetchAndConvertAllSourcesFail(t *testing.T) {
	var calls []string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.URL.String())
		return &http.Response{
			StatusCode:    http.StatusServiceUnavailable,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("")),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	_, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "t", URL: "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/reject-list.txt"})
	if err == nil {
		t.Fatal("expected error when all sources fail")
	}
	if len(calls) != 3 {
		t.Fatalf("calls = %d, want 3 (primary+cdn+fastly), got %#v", len(calls), calls)
	}
	for _, u := range calls {
		switch {
		case strings.HasPrefix(u, "https://raw.githubusercontent.com/"),
			strings.HasPrefix(u, "https://cdn.jsdelivr.net/"),
			strings.HasPrefix(u, "https://fastly.jsdelivr.net/"):
		default:
			t.Fatalf("unexpected call %q", u)
		}
	}
}
