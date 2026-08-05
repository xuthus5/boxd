package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func newTestRuleSetService(t *testing.T) (*RuleSetService, string) {
	t.Helper()
	dir := t.TempDir()
	db := newTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	settings := core.NewSettingsManager(db)
	configPath := filepath.Join(dir, "config.json")
	updater := core.NewRuleSetUpdater(configPath, dir, nil, nil, nil)
	return NewRuleSetService(updater, settings), configPath
}

func TestRuleSetServiceAutoUpdateDefaults(t *testing.T) {
	svc, _ := newTestRuleSetService(t)
	cfg, err := svc.AutoUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Fatal("expected auto update disabled by default")
	}
	if cfg.Interval == "" {
		t.Fatal("expected default interval")
	}
}

func TestRuleSetServiceSetAutoUpdate(t *testing.T) {
	svc, _ := newTestRuleSetService(t)
	cfg := model.RuleSetAutoUpdate{Enabled: true, Interval: "24h"}
	if err := svc.SetAutoUpdate(cfg); err != nil {
		t.Fatal(err)
	}
	saved, err := svc.AutoUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Enabled || saved.Interval != "24h" {
		t.Fatalf("saved = %#v", saved)
	}
}

func TestRuleSetServiceSetAutoUpdateInvalid(t *testing.T) {
	svc, _ := newTestRuleSetService(t)
	cfg := model.RuleSetAutoUpdate{Enabled: true, Interval: ""}
	if err := svc.SetAutoUpdate(cfg); err == nil {
		t.Fatal("expected error for empty interval")
	}
}

func TestRuleSetServiceStatusMissingConfig(t *testing.T) {
	svc, _ := newTestRuleSetService(t)
	// 配置不存在时，Status 应返回错误而非 panic。
	if _, err := svc.Status(context.Background()); err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestRuleSetServiceNilDeps(t *testing.T) {
	svc := NewRuleSetService(nil, nil)
	if _, err := svc.Status(context.Background()); err == nil {
		t.Fatal("expected error when updater is nil")
	}
	if _, err := svc.Update(context.Background(), core.RuleSetUpdateRequest{}); err == nil {
		t.Fatal("expected error when updater is nil")
	}
	if _, err := svc.AutoUpdate(); err == nil {
		t.Fatal("expected error when settings is nil")
	}
	if err := svc.SetAutoUpdate(model.RuleSetAutoUpdate{Enabled: true, Interval: "24h"}); err == nil {
		t.Fatal("expected error when settings is nil")
	}
}
