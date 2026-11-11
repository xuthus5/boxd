package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestValidateConfigSectionAndEmptyShape(t *testing.T) {
	for _, section := range []string{sectionInbounds, sectionRoute, sectionNetworkNamespaces, sectionCertificateProvider} {
		if err := validateConfigSection(section); err != nil {
			t.Errorf("validate %s: %v", section, err)
		}
	}
	if err := validateConfigSection("unknown"); err == nil {
		t.Fatal("unknown section was accepted")
	}
	if got := emptyConfigSection(sectionInbounds); got != "[]\n" {
		t.Fatalf("array empty section = %q", got)
	}
	if got := emptyConfigSection(sectionRoute); got != "{}\n" {
		t.Fatalf("object empty section = %q", got)
	}
}

func TestDecodeJSONValueRejectsTrailingData(t *testing.T) {
	value, err := decodeJSONValue(`{"level":"info"}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := value.(map[string]any); !ok {
		t.Fatalf("decoded value = %T", value)
	}
	if _, err := decodeJSONValue("{"); err == nil {
		t.Fatal("invalid JSON was accepted")
	}
	if _, err := decodeJSONValue(`{} trailing`); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}

func TestRuntimeBackendConfigSectionsIntegration(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testRuntimeConfig(dir)
	rt, backend := openRuntimeBackend(t, cfg)
	t.Cleanup(func() { closeRuntime(t, rt) })
	ctx := context.Background()

	logJSON, err := backend.LoadConfigSection(ctx, sectionLog)
	if err != nil {
		t.Fatalf("load log section: %v", err)
	}
	if !strings.Contains(logJSON, `"level"`) {
		t.Fatalf("log section = %q", logJSON)
	}
	nextLog := `{"level":"debug","timestamp":true}`
	if err := backend.ValidateConfigSection(ctx, sectionLog, nextLog); err != nil {
		t.Fatalf("validate log section: %v", err)
	}
	if err := backend.ApplyConfigSection(ctx, sectionLog, nextLog); err != nil {
		t.Fatalf("apply log section: %v", err)
	}
	logJSON, err = backend.LoadConfigSection(ctx, sectionLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logJSON, `"debug"`) {
		t.Fatalf("applied log section = %q", logJSON)
	}
	if _, err := backend.LoadConfigSection(ctx, "unknown"); err == nil {
		t.Fatal("unknown section was loaded")
	}
	if err := backend.ApplyConfigSection(ctx, sectionLog, `{"level":`); err == nil {
		t.Fatal("invalid section JSON was applied")
	}
	if err := backend.InstallDefaultConfigSection(ctx, sectionEndpoints); err == nil {
		t.Fatal("unsupported default installer was accepted")
	}

	diagnostics, err := backend.ConfigDiagnostics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics.Status == "" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
	events, err := backend.ConfigApplyHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("apply history is empty")
	}
	restoreID := findSnapshotID(t, backend, events, `"debug"`)
	if _, err := backend.ConfigSnapshot(ctx, restoreID); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if err := backend.RestoreConfigSnapshot(ctx, restoreID); err != nil {
		t.Fatalf("restore snapshot: %v", err)
	}
	logJSON, err = backend.LoadConfigSection(ctx, sectionLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logJSON, `"debug"`) {
		t.Fatalf("restored log section = %q", logJSON)
	}
	if _, err := backend.ConfigSnapshot(ctx, ""); err == nil {
		t.Fatal("empty snapshot id was accepted")
	}

	items, err := backend.RuleSetStatus(ctx)
	if err != nil {
		t.Fatalf("rule-set status: %v", err)
	}
	if items == nil {
		t.Fatal("rule-set status returned nil")
	}
	auto, err := backend.RuleSetAutoUpdate(ctx)
	if err != nil {
		t.Fatalf("rule-set auto update: %v", err)
	}
	if auto.Interval == "" {
		t.Fatalf("auto update = %+v", auto)
	}

	backupPath := filepath.Join(dir, "backup.tar.gz")
	if err := backend.CreateBackup(ctx, backupPath); err != nil {
		t.Fatalf("create backup: %v", err)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("backup is empty")
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup mode = %o, want 600", info.Mode().Perm())
	}
}

func findSnapshotID(t *testing.T, backend *runtimeBackend, events []model.ConfigApplyEvent, needle string) string {
	t.Helper()
	for _, event := range events {
		body, err := backend.ConfigSnapshot(context.Background(), event.ID)
		if err != nil {
			continue
		}
		if strings.Contains(body, needle) {
			return event.ID
		}
	}
	t.Fatalf("no snapshot contains %q", needle)
	return ""
}

func TestRuntimeBackendAdvancedUnavailable(t *testing.T) {
	var b *runtimeBackend
	ctx := context.Background()
	if _, err := b.LoadConfigSection(ctx, sectionLog); err == nil {
		t.Fatal("LoadConfigSection worked without runtime")
	}
	if err := b.ValidateConfigSection(ctx, sectionLog, "{}"); err == nil {
		t.Fatal("ValidateConfigSection worked without runtime")
	}
	if _, err := b.ConfigDiagnostics(ctx); err == nil {
		t.Fatal("ConfigDiagnostics worked without runtime")
	}
	if _, err := b.ConfigApplyHistory(ctx); err == nil {
		t.Fatal("ConfigApplyHistory worked without runtime")
	}
	if _, err := b.RuleSetStatus(ctx); err == nil {
		t.Fatal("RuleSetStatus worked without runtime")
	}
	if err := b.ChangePassword(ctx, "old", "new"); err == nil {
		t.Fatal("ChangePassword worked without runtime")
	}
	if err := b.CreateBackup(ctx, filepath.Join(t.TempDir(), "backup.tar.gz")); err == nil {
		t.Fatal("CreateBackup worked without runtime")
	}
}
