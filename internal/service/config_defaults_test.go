package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

func TestInstallDefaultRuleSets(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
	})
	installer := newFakeRuleSetInstaller()
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		RuleSetInstaller: installer,
		ApplyHistory:     svc.Deps.ApplyHistory,
	})
	result, err := cfg.InstallDefaultRuleSets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
	if result.InstalledCount != 1 {
		t.Fatalf("installed = %d", result.InstalledCount)
	}
	data, err := os.ReadFile(svc.Deps.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(string(data), "rule-set-tag") {
		t.Fatalf("rule_set missing: %s", data)
	}
}

func TestInstallDefaultRuleSetsNotConfigured(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{})
	_, err := cfg.InstallDefaultRuleSets(context.Background())
	if err == nil {
		t.Fatal("expected error for nil installer")
	}
}

func TestInstallDefaultRuleSetsInstallerError(t *testing.T) {
	svc := newTestService(t)
	installer := &fakeRuleSetInstaller{err: errors.New("download failed")}
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{RuleSetInstaller: installer})
	_, err := cfg.InstallDefaultRuleSets(context.Background())
	if err == nil {
		t.Fatal("expected installer error")
	}
}

func TestInstallDefaultOutbounds(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		OutboundInstaller: core.NewDefaultOutboundsInstaller(),
		ApplyHistory:      svc.Deps.ApplyHistory,
	})
	result, err := cfg.InstallDefaultOutbounds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestInstallDefaultOutboundsNotConfigured(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{})
	_, err := cfg.InstallDefaultOutbounds(context.Background())
	if err == nil {
		t.Fatal("expected error for nil installer")
	}
}

func TestInstallDefaultRouteRules(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		RouteInstaller: core.NewDefaultRouteInstaller(),
		ApplyHistory:   svc.Deps.ApplyHistory,
	})
	result, err := cfg.InstallDefaultRouteRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestInstallDefaultRouteRulesNotConfigured(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{})
	_, err := cfg.InstallDefaultRouteRules(context.Background())
	if err == nil {
		t.Fatal("expected error for nil installer")
	}
}

func TestInstallDefaultDNS(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		DNSInstaller: core.NewDefaultDNSInstaller(),
		ApplyHistory: svc.Deps.ApplyHistory,
	})
	result, err := cfg.InstallDefaultDNS(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestInstallDefaultDNSNotConfigured(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{})
	_, err := cfg.InstallDefaultDNS(context.Background())
	if err == nil {
		t.Fatal("expected error for nil installer")
	}
}

func TestInstallDefaultInbounds(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		InboundInstaller: core.NewDefaultInboundsInstaller(),
		ApplyHistory:     svc.Deps.ApplyHistory,
	})
	result, err := cfg.InstallDefaultInbounds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestInstallDefaultInboundsNotConfigured(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{})
	_, err := cfg.InstallDefaultInbounds(context.Background())
	if err == nil {
		t.Fatal("expected error for nil installer")
	}
}

func TestInstallDefaultExperimental(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		ExperimentalInstaller: core.NewDefaultExperimentalInstaller(),
		ApplyHistory:          svc.Deps.ApplyHistory,
	})
	result, err := cfg.InstallDefaultExperimental(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestInstallDefaultExperimentalNotConfigured(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{})
	_, err := cfg.InstallDefaultExperimental(context.Background())
	if err == nil {
		t.Fatal("expected error for nil installer")
	}
}

func TestConfigDefaultsReadMissingConfig(t *testing.T) {
	cfg := newConfig(filepath.Join(t.TempDir(), "missing.json"), nil, ConfigInstaller{
		OutboundInstaller: core.NewDefaultOutboundsInstaller(),
	})
	result, err := cfg.InstallDefaultOutbounds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestConfigDefaultsInvalidJSON(t *testing.T) {
	svc := newTestService(t)
	if err := os.WriteFile(svc.Deps.ConfigPath, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		OutboundInstaller: core.NewDefaultOutboundsInstaller(),
	})
	_, err := cfg.InstallDefaultOutbounds(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestConfigApplyConfigWithInstance(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{"outbounds": []any{}})
	cfg := newConfig(svc.Deps.ConfigPath, &fakeRestart{}, ConfigInstaller{
		ApplyHistory: svc.Deps.ApplyHistory,
	})
	result, err := cfg.ApplyConfig(context.Background(), []byte(`{"log":{"level":"debug"}}`), "update")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestConfigValidateConfigRuntimeError(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{
		ApplyHistory: svc.Deps.ApplyHistory,
	})
	err := cfg.ValidateConfig(context.Background(), []byte(outboundDependencyCycleBody), "validate")
	if err == nil {
		t.Fatal("expected runtime error")
	}
	var invalid *ErrInvalidRuntime
	if !errors.As(err, &invalid) {
		t.Fatalf("expected ErrInvalidRuntime, got %T", err)
	}
}

func TestConfigValidateConfigUnknownError(t *testing.T) {
	svc := newTestService(t)
	cfg := newConfig(svc.Deps.ConfigPath, nil, ConfigInstaller{})
	err := cfg.ValidateConfig(context.Background(), []byte(`{"log":{"level":"info"}}`), "validate")
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfigRecordApplyNil(t *testing.T) {
	var cfg *Config
	cfg.recordApply("source", "ok", []byte("{}"), nil)
}

func TestInstallResultCount(t *testing.T) {
	if got := installedCount([]map[string]any{{}}); got != 1 {
		t.Fatalf("installed = %d", got)
	}
	if got := installedCount("string"); got != 0 {
		t.Fatalf("installed = %d", got)
	}
	if got := installedCount(nil); got != 0 {
		t.Fatalf("installed = %d", got)
	}
}

type fakeRuleSetInstaller struct {
	entries []map[string]any
	err     error
}

func newFakeRuleSetInstaller() *fakeRuleSetInstaller {
	return &fakeRuleSetInstaller{entries: []map[string]any{
		{"type": "remote", "tag": "rule-set-tag", "url": "https://example.com/rules.json"},
	}}
}

func (f *fakeRuleSetInstaller) Install(_ context.Context) ([]map[string]any, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

const outboundDependencyCycleBody = `{
  "outbounds":[
    {"type":"selector","tag":"group-a","outbounds":["group-b"]},
    {"type":"urltest","tag":"group-b","outbounds":["group-a"]}
  ]
}`

var _ core.RuleSetDefaultsInstaller = (*fakeRuleSetInstaller)(nil)
