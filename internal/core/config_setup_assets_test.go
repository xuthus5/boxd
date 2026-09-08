package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupAssetsInstallOnlyReferencedDefaults(t *testing.T) {
	env := setupTestEnvironment(t)
	plan, err := BuildSetupPlan([]byte(`{}`), SetupRequest{Modules: []string{"rule_sets"}}, env)
	if err != nil {
		t.Fatal(err)
	}
	if err := InstallSetupAssets(t.Context(), plan, env.DataDir); err != nil {
		t.Fatal(err)
	}
	route := plan.Config["route"].(map[string]any)
	for _, value := range route["rule_set"].([]any) {
		entry := value.(map[string]any)
		path := entry["path"].(string)
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 || info.Mode().Perm() != 0600 {
			t.Fatalf("rule asset not installed privately: %s, %v", path, err)
		}
	}
}

func TestSetupAssetsPreserveCustomSources(t *testing.T) {
	env := setupTestEnvironment(t)
	entries := []any{}
	for _, source := range defaultRuleSetSources() {
		entries = append(entries, map[string]any{"tag": source.Tag, "type": "local", "path": "/custom/" + source.FileName})
	}
	plan := &SetupPlan{Modules: []string{"rule_sets"}, Config: map[string]any{"route": map[string]any{"rule_set": entries}}}
	if err := InstallSetupAssets(t.Context(), plan, env.DataDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(env.DataDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("custom references must not create unused bundled files")
	}
}

func TestSetupAssetsCancellationAndNoSelection(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, modules := range [][]string{nil, {"rule_sets"}} {
		plan := &SetupPlan{Modules: modules, Config: map[string]any{}}
		if err := InstallSetupAssets(ctx, plan, t.TempDir()); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	}
	if err := InstallSetupAssets(t.Context(), &SetupPlan{}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestSetupAssetsReportWriteFailure(t *testing.T) {
	env := setupTestEnvironment(t)
	if err := os.WriteFile(env.DataDir, nil, 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildSetupPlan(nil, SetupRequest{Modules: []string{"rule_sets"}}, env)
	if err != nil {
		t.Fatal(err)
	}
	if err := InstallSetupAssets(t.Context(), plan, env.DataDir); err == nil {
		t.Fatal("expected rule asset write failure")
	}
}

func TestSetupSourceBackupIsVerifiableAndIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	body := []byte("broken original\n")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	hash := ConfigContentHash(body)
	for range 2 {
		if err := BackupSetupSource(path, dir, hash); err != nil {
			t.Fatal(err)
		}
	}
	backup := filepath.Join(dir, "config-backups", "before-setup-"+hash+".json")
	saved, err := os.ReadFile(backup)
	if err != nil || ConfigContentHash(saved) != hash {
		t.Fatalf("backup does not match original: %v", err)
	}
	if err := os.WriteFile(backup, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := BackupSetupSource(path, dir, hash); err == nil {
		t.Fatal("a corrupt existing backup must not be silently accepted")
	}
}

func TestSetupSourceBackupRejectsStaleOrUnwritableSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := BackupSetupSource(path, dir, "stale"); !errors.Is(err, ErrSetupStale) {
		t.Fatalf("stale source was not rejected: %v", err)
	}
	if err := BackupSetupSource(path, path, ConfigContentHash([]byte("{}"))); err == nil {
		t.Fatal("backup directory errors must prevent applying")
	}
	if err := BackupSetupSource(dir, dir, ""); err == nil {
		t.Fatal("source read failure was ignored")
	}
	if err := BackupSetupSource(filepath.Join(dir, "missing"), dir, ConfigContentHash(nil)); err != nil {
		t.Fatalf("missing original does not need a backup: %v", err)
	}
}
