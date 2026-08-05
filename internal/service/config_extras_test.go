package service

import (
	"path/filepath"
	"testing"
)

func newTestConfigWithMetadata(t *testing.T) *Config {
	t.Helper()
	svc := newTestService(t)
	return newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		ApplyHistory:  svc.Deps.ApplyHistory,
		RouteMetadata: svc.Deps.RouteMetadata,
	})
}

func TestConfigDiagnostics(t *testing.T) {
	cfg := newTestConfigWithMetadata(t)
	writeTestConfig(t, cfg.path, map[string]any{
		"inbounds":  []any{},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
	})
	// 仅确保不 panic。
	_ = cfg.Diagnostics()
}

func TestConfigApplyHistoryEmpty(t *testing.T) {
	cfg := newTestConfigWithMetadata(t)
	events, err := cfg.ApplyHistory()
	if err != nil {
		t.Fatal(err)
	}
	if events == nil {
		t.Fatal("expected empty (non-nil) events")
	}
}

func TestConfigSnapshotNotFound(t *testing.T) {
	cfg := newTestConfigWithMetadata(t)
	if _, err := cfg.ConfigSnapshot("nonexistent"); err == nil {
		t.Fatal("expected error for missing snapshot")
	}
}

func TestConfigRestoreSnapshotNotFound(t *testing.T) {
	cfg := newTestConfigWithMetadata(t)
	if _, err := cfg.RestoreConfigSnapshot(t.Context(), "nonexistent"); err == nil {
		t.Fatal("expected error for missing snapshot")
	}
}

func TestConfigRouteRules(t *testing.T) {
	cfg := newTestConfigWithMetadata(t)
	writeTestConfig(t, cfg.path, map[string]any{
		"route": map[string]any{
			"rules": []any{map[string]any{"action": "direct"}},
		},
	})
	rules, err := cfg.RouteRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules = %d", len(rules))
	}
}

func TestConfigGetRouteRuleMetadata(t *testing.T) {
	cfg := newTestConfigWithMetadata(t)
	writeTestConfig(t, cfg.path, map[string]any{
		"route": map[string]any{
			"rules": []any{map[string]any{"action": "direct"}},
		},
	})
	metadata, err := cfg.GetRouteRuleMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if metadata == nil {
		t.Fatal("metadata should not be nil")
	}
}

func TestConfigUpdateRouteRuleMetadata(t *testing.T) {
	cfg := newTestConfigWithMetadata(t)
	writeTestConfig(t, cfg.path, map[string]any{
		"route": map[string]any{
			"rules": []any{
				map[string]any{"action": "direct", "inbound": []string{"mixed-in"}},
			},
		},
	})
	metadata, err := cfg.GetRouteRuleMetadata()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := cfg.UpdateRouteRuleMetadata(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != len(metadata) {
		t.Fatalf("saved %d, want %d", len(saved), len(metadata))
	}
}

func TestConfigDiagnosticsMissingFile(t *testing.T) {
	dir := t.TempDir()
	svc := newTestService(t)
	cfg := newConfig(filepath.Join(dir, "missing.json"), nil, ConfigInstaller{
		ApplyHistory: svc.Deps.ApplyHistory,
	})
	// 缺失文件时应优雅处理（不 panic）。
	_ = cfg.Diagnostics()
}

func TestConfigRouteMetadataNotConfigured(t *testing.T) {
	dir := t.TempDir()
	cfg := newConfig(filepath.Join(dir, "config.json"), nil, ConfigInstaller{})
	if _, err := cfg.GetRouteRuleMetadata(); err == nil {
		t.Fatal("expected error when route metadata is not configured")
	}
}

func TestConfigNilReceiver(t *testing.T) {
	var cfg *Config
	_ = cfg.Diagnostics()
	if _, err := cfg.RouteRules(); err == nil {
		t.Fatal("expected error for nil config")
	}
}
