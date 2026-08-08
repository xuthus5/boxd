package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestApplyInstalledConfigWithRollback(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := newConfig(configPath, &fakeRestart{errs: []error{errors.New("boom"), nil}}, ConfigInstaller{
		OutboundInstaller: core.NewDefaultOutboundsInstaller(),
	})
	result, err := cfg.InstallDefaultOutbounds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "rolled_back" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestMergeRuleSetsReplaceAndAppend(t *testing.T) {
	existing := []any{
		map[string]any{"type": "remote", "tag": "geo-a", "url": "old"},
	}
	installed := []map[string]any{
		{"type": "remote", "tag": "geo-a", "url": "new"},
		{"type": "remote", "tag": "geo-b", "url": "b"},
	}
	merged := mergeRuleSets(existing, installed)
	if len(merged) != 2 {
		t.Fatalf("merged = %v", merged)
	}
	first, _ := merged[0].(map[string]any)
	if first["url"] != "new" {
		t.Fatalf("first url = %v", first["url"])
	}
}

func TestMergeRuleSetsNoTags(t *testing.T) {
	merged := mergeRuleSets(nil, []map[string]any{{"type": "remote", "url": "x"}})
	if len(merged) != 1 {
		t.Fatalf("merged = %v", merged)
	}
}

func TestShouldReplaceExistingOutboundBranches(t *testing.T) {
	managed := map[string]bool{"g1": true}
	tests := []struct {
		name  string
		entry map[string]any
		want  bool
	}{
		{name: "nil", entry: nil, want: true},
		{name: "managed node type", entry: map[string]any{"type": "vless", "tag": "n"}, want: true},
		{name: "dns", entry: map[string]any{"type": "dns", "tag": "dns"}, want: true},
		{name: "direct preserved", entry: map[string]any{"type": "direct", "tag": "direct"}, want: false},
		{name: "direct other", entry: map[string]any{"type": "direct", "tag": "bypass"}, want: false},
		{name: "managed group urltest", entry: map[string]any{"type": "urltest", "tag": "g1"}, want: true},
		{name: "managed group selector", entry: map[string]any{"type": "selector", "tag": "g1"}, want: true},
		{name: "unmanaged selector", entry: map[string]any{"type": "selector", "tag": "other"}, want: false},
		{name: "proxy selector", entry: map[string]any{"type": "selector", "tag": "proxy"}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldReplaceExistingOutbound(test.entry, managed); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestRollbackConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rollbackConfigFile(path, []byte(`{"a":1}`), true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"a":1}` {
		t.Fatalf("data = %s", data)
	}
	if err := rollbackConfigFile(path, nil, false); err != nil {
		t.Fatal(err)
	}
	if err := rollbackConfigFile(filepath.Join(dir, "missing.json"), nil, false); err != nil {
		t.Fatal(err)
	}
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")
	if err := atomicWriteFile(path, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"a":1}` {
		t.Fatalf("data = %s", data)
	}
}

func TestRestartAdapterWithInstance(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "missing.json")
	instance := core.NewSBInstance(configPath, core.NewLogWriter(5))
	adapter := restartAdapter{instance: instance}
	if err := adapter.Restart(); err == nil {
		t.Fatal("expected restart error for missing config")
	}
	adapter2 := restartAdapter{}
	if err := adapter2.Restart(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncOutboundsAndRestartNilInstance(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	if err := nodeMgr.Add(model.Outbound{Tag: "desired", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if err := SyncOutboundsAndRestart(nodeMgr, subMgr, configPath, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSyncOutboundsAndRestartRollbackRestartFailures(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	previous := []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if err := os.WriteFile(configPath, previous, 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	if err := nodeMgr.Add(model.Outbound{Tag: "desired", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	instance := &fakeRestart{errs: []error{
		errors.New("new restart failed"),
		errors.New("restored restart failed"),
	}}
	err := SyncOutboundsAndRestart(nodeMgr, subMgr, configPath, instance)
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsSubstring(err.Error(), "restored configuration restart failed") &&
		!containsSubstring(err.Error(), "restart failed after restoring previous configuration") {
		t.Fatalf("err = %v", err)
	}
}

func TestRuntimeDiagnosticDetail(t *testing.T) {
	tests := []struct {
		code  string
		value string
		want  string
	}{
		{code: "duplicate_tag", value: "same", want: `duplicate tag "same"`},
		{code: "missing_tag", want: "tag is required"},
		{code: "unknown_code", value: "v", want: "unknown code"},
		{code: "empty_group", value: "g", want: `"g" outbound group must contain at least one member`},
	}
	for _, test := range tests {
		got := runtimeDiagnosticDetail(model.ConfigDiagnostic{Code: test.code, Value: test.value})
		if got != test.want {
			t.Fatalf("%s: got %q, want %q", test.code, got, test.want)
		}
	}
}

func TestValidateRuntimeConfigUnmarshalError(t *testing.T) {
	err := ValidateRuntimeConfig(context.Background(), []byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error")
	}
	var invalid *ErrInvalidRuntime
	if !errors.As(err, &invalid) {
		t.Fatalf("type = %T", err)
	}
}
