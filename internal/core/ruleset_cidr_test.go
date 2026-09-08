package core

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRuleSetDefaultSourcesRetainLegacyUpdates(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	want := []string{"loyalsoldier-direct", "loyalsoldier-proxy", "loyalsoldier-reject", "geoip-cn"}
	got := make([]string, 0, len(installer.sources))
	for _, src := range installer.sources {
		got = append(got, src.Tag)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("default tags = %v, want %v", got, want)
	}
	for _, tag := range []string{"geosite-cn", "geosite-google-play", "geosite-category-ads-all"} {
		src, ok := installer.SourceByTag(tag)
		if !ok || src.Format != "binary" || !installer.IsBuiltinLocal(tag) {
			t.Fatalf("legacy source %s no longer supports updates: %+v", tag, src)
		}
	}
	src, ok := installer.SourceByTag("geoip-cn")
	wantURL := "https://raw.githubusercontent.com/gaoyifan/china-operator-ip/ip-lists/china.txt"
	if !ok || src.Format != "cidr" || src.FileName != "geoip-cn.srs" || src.URL != wantURL {
		t.Fatalf("CN source = %+v, found = %v", src, ok)
	}
}

func TestRuleSetParseCIDRs(t *testing.T) {
	t.Parallel()
	content := []byte("# comment\n\n223.5.5.1/24\n223.5.5.0/24\n2400:3200::1/32\n")
	got, err := parseRuleSetCIDRs("geoip-cn", content)
	want := []string{"223.5.5.0/24", "2400:3200::/32"}
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatalf("prefixes = %v, error = %v, want %v", got, err, want)
	}
	data, err := convertCIDRRuleSet("geoip-cn", content)
	if err != nil {
		t.Fatal(err)
	}
	assertBundledCNAddresses(t, data)
}

func TestRuleSetParseCIDRsErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{name: "empty", body: "# no ranges\n"},
		{name: "invalid prefix", body: "not-a-prefix\n"},
		{name: "missing ipv6", body: "223.5.5.0/24\n"},
		{name: "missing ipv4", body: "2400:3200::/32\n"},
		{name: "oversized line", body: strings.Repeat("x", 1<<20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := convertCIDRRuleSet("geoip-cn", []byte(tt.body))
			if err == nil || data != nil {
				t.Fatalf("data = %v, error = %v, want conversion failure", data, err)
			}
		})
	}
}

func TestRuleSetOnlineInstallDefaultSources(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	var mu sync.Mutex
	calls := make(map[string]bool)
	bodies := map[string]string{
		"direct-list.txt": "baidu.com\n", "proxy-list.txt": "google.com\n",
		"reject-list.txt": "doubleclick.net\n", "china.txt": "223.5.5.0/24\n", "china6.txt": "2400:3200::/32\n",
	}
	installer.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		name := filepath.Base(req.URL.Path)
		mu.Lock()
		calls[name] = true
		mu.Unlock()
		body, ok := bodies[name]
		if !ok {
			t.Errorf("unexpected default download: %s", req.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
	entries, err := installer.Install(t.Context())
	if err != nil || len(entries) != 4 {
		t.Fatalf("entries = %v, error = %v, want four rule sets", entries, err)
	}
	if len(calls) != len(bodies) {
		t.Fatalf("downloaded sources = %v, want all five source files", calls)
	}
	data, err := os.ReadFile(filepath.Join(installer.RuleSetDir(), "geoip-cn.srs"))
	if err != nil {
		t.Fatal(err)
	}
	assertBundledCNAddresses(t, data)
}

func TestRuleSetCIDRUpdater(t *testing.T) {
	t.Parallel()
	installer := newCIDRTestInstaller(t, "2400:3200::/32\n", http.StatusOK)
	updater, path := newCIDRTestUpdater(t, installer)
	response, err := updater.Update(t.Context(), RuleSetUpdateRequest{})
	if err != nil || response.UpdatedCount != 1 || response.FailedCount != 0 {
		t.Fatalf("update response = %+v, error = %v", response, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertBundledCNAddresses(t, data)
}

func TestRuleSetCIDRUpdaterKeepsCacheOnFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ipv6   string
		status int
	}{
		{name: "download failure", status: http.StatusBadGateway},
		{name: "invalid prefix", ipv6: "invalid\n", status: http.StatusOK},
		{name: "empty family", ipv6: "# no ipv6\n", status: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			installer := newCIDRTestInstaller(t, tt.ipv6, tt.status)
			updater, path := newCIDRTestUpdater(t, installer)
			before := []byte("previous-cache")
			if err := atomicWriteFile0600(path, before); err != nil {
				t.Fatal(err)
			}
			response, err := updater.Update(t.Context(), RuleSetUpdateRequest{})
			if err != nil || response.UpdatedCount != 0 || response.FailedCount != 1 {
				t.Fatalf("update response = %+v, error = %v, want one failure", response, err)
			}
			data, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, data) {
				t.Fatalf("failed update replaced cache: %q, error = %v", data, err)
			}
		})
	}
}

func newCIDRTestInstaller(t *testing.T, ipv6 string, status int) *LoyalsoldierRuleSetInstaller {
	t.Helper()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	installer.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, code := "223.5.5.0/24\n", http.StatusOK
		if strings.HasSuffix(req.URL.Path, "china6.txt") {
			body, code = ipv6, status
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
	return installer
}

func newCIDRTestUpdater(t *testing.T, installer *LoyalsoldierRuleSetInstaller) (*RuleSetUpdater, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "existing-cn.srs")
	configPath := filepath.Join(dir, "config.json")
	config := map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"type": "local", "tag": "geoip-cn", "format": "binary", "path": path},
	}}}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	return NewRuleSetUpdater(configPath, dir, installer, nil, nil), path
}
