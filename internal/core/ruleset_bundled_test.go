package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/sagernet/sing-box/common/srs"
)

func TestRuleSetInstallBundledOffline(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	installer.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("bundled initialization must not access the network")
		return nil, errors.New("offline")
	})}
	entries, err := installer.InstallBundled(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	wantTags := []string{"loyalsoldier-direct", "loyalsoldier-proxy", "loyalsoldier-reject", "geoip-cn"}
	if len(entries) != len(wantTags) {
		t.Fatalf("entries = %v, want exactly %v", entries, wantTags)
	}
	for index, tag := range wantTags {
		entry := entries[index]
		if entry["tag"] != tag || entry["type"] != "local" || len(entry) != 4 {
			t.Fatalf("entry = %#v, want minimal local %s entry", entry, tag)
		}
		path := entry["path"].(string)
		assertBundledRuleSetFile(t, path, tag)
		if tag == "geoip-cn" && entry["format"] != "binary" {
			t.Fatalf("geoip entry format = %v, want binary", entry["format"])
		}
	}
	info, err := os.Stat(installer.RuleSetDir())
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("directory info = %v, error = %v, want mode 0700", info, err)
	}
}

func assertBundledRuleSetFile(t *testing.T, path, tag string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("file info = %v, error = %v, want mode 0600", info, err)
	}
	wantTime := time.Date(2026, time.September, 7, 23, 54, 59, 0, time.UTC)
	if tag == "geoip-cn" {
		wantTime = time.Date(2026, time.September, 7, 7, 38, 18, 0, time.UTC)
	}
	if !info.ModTime().Equal(wantTime) {
		t.Fatalf("snapshot mtime = %v, want upstream release %v", info.ModTime(), wantTime)
	}
	if tag == "geoip-cn" {
		assertBundledCNAddresses(t, data)
		return
	}
	var parsed sourceRuleSetFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	minimum := map[string]int{"loyalsoldier-direct": 100000, "loyalsoldier-proxy": 25000, "loyalsoldier-reject": 180000}
	example := map[string]string{"loyalsoldier-direct": "baidu.com", "loyalsoldier-proxy": "google.com", "loyalsoldier-reject": "doubleclick.net"}
	if len(parsed.Rules) != 1 || len(parsed.Rules[0].DomainSuffix) < minimum[tag] {
		t.Fatalf("%s snapshot is incomplete", tag)
	}
	if !slices.Contains(parsed.Rules[0].DomainSuffix, example[tag]) {
		t.Fatalf("%s snapshot does not match %s", tag, example[tag])
	}
}

func assertBundledCNAddresses(t *testing.T, data []byte) {
	t.Helper()
	parsed, err := srs.Read(bytes.NewReader(data), false)
	if err != nil || len(parsed.Options.Rules) != 1 {
		t.Fatalf("CN snapshot decode error = %v, rules = %d", err, len(parsed.Options.Rules))
	}
	ipSet := parsed.Options.Rules[0].DefaultOptions.IPSet
	if ipSet == nil {
		t.Fatal("CN snapshot has no IP set")
	}
	for _, address := range []string{"223.5.5.5", "2400:3200::1"} {
		if !ipSet.Contains(netip.MustParseAddr(address)) {
			t.Errorf("CN snapshot must contain %s", address)
		}
	}
	if ipSet.Contains(netip.MustParseAddr("1.1.1.1")) || ipSet.Contains(netip.MustParseAddr("2606:4700:4700::1111")) {
		t.Fatal("CN snapshot incorrectly includes Cloudflare overseas ranges")
	}
}

func TestRuleSetInstallBundledPreservesValidCache(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	path := filepath.Join(installer.RuleSetDir(), "loyalsoldier-direct.json")
	content := []byte(`{"version":4,"rules":[{"domain_suffix":["updated.example"]}]}`)
	if err := atomicWriteFile0600(path, content); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.InstallBundled(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(content, data) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("valid updated cache changed: content=%q mtime=%v error=%v", data, after.ModTime(), err)
	}
}

func TestRuleSetInstallBundledRepairsCorruptCache(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	for _, src := range installer.sources {
		if err := atomicWriteFile0600(filepath.Join(installer.RuleSetDir(), src.FileName), []byte("corrupt")); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := installer.InstallBundled(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		assertBundledRuleSetFile(t, entry["path"].(string), entry["tag"].(string))
	}
}

func TestRuleSetInstallBundledCanceled(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "untouched")
	installer := NewLoyalsoldierRuleSetInstaller(dir)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	entries, err := installer.InstallBundled(ctx)
	if !errors.Is(err, context.Canceled) || entries != nil {
		t.Fatalf("entries = %v, error = %v, want nil and context.Canceled", entries, err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled installation created directory: %v", err)
	}
}
