package core

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuleSetInstallBundledFailureRetry(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	blocked := filepath.Join(installer.RuleSetDir(), "loyalsoldier-proxy.json")
	if err := os.MkdirAll(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	entries, err := installer.InstallBundled(t.Context())
	if err == nil || entries != nil {
		t.Fatalf("partial install: entries = %v, error = %v, want nil and failure", entries, err)
	}
	first := filepath.Join(installer.RuleSetDir(), "loyalsoldier-direct.json")
	before, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(installer.RuleSetDir(), "geoip-cn.srs")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed install continued writing: %v", err)
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	entries, err = installer.InstallBundled(t.Context())
	if err != nil || len(entries) != 4 {
		t.Fatalf("retry entries = %v, error = %v", entries, err)
	}
	after, err := os.Stat(first)
	if err != nil || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("retry replaced the successful first snapshot: %v", err)
	}
	assertNoRuleSetTemporaryFiles(t, installer.RuleSetDir())
}

func TestRuleSetInstallBundledFilesystemErrors(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	installer := NewLoyalsoldierRuleSetInstaller(file)
	if entries, err := installer.InstallBundled(t.Context()); err == nil || entries != nil {
		t.Fatalf("entries = %v, error = %v, want directory failure", entries, err)
	}
	installer = NewLoyalsoldierRuleSetInstaller(t.TempDir())
	installer.sources = []RuleSetSource{{Tag: "not-bundled", FileName: "missing.json"}}
	if entries, err := installer.InstallBundled(t.Context()); err == nil || entries != nil {
		t.Fatalf("entries = %v, error = %v, want missing snapshot failure", entries, err)
	}
}

func TestRuleSetInstallBundledCancellationAfterFirstFile(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	first := filepath.Join(installer.RuleSetDir(), "loyalsoldier-direct.json")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cancelContext := cancelAfterRuleSetFileContext{Context: ctx, cancel: cancel, path: first}
	entries, err := installer.InstallBundled(cancelContext)
	if !errors.Is(err, context.Canceled) || entries != nil {
		t.Fatalf("entries = %v, error = %v, want nil and canceled", entries, err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("first file was not installed before cancellation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installer.RuleSetDir(), "loyalsoldier-proxy.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("installation continued after cancellation: %v", err)
	}
	assertNoRuleSetTemporaryFiles(t, installer.RuleSetDir())
}

type cancelAfterRuleSetFileContext struct {
	context.Context
	cancel context.CancelFunc
	path   string
}

func (c cancelAfterRuleSetFileContext) Err() error {
	if _, err := os.Stat(c.path); err == nil {
		c.cancel()
	}
	return c.Context.Err()
}

func TestRuleSetWriteBundledSnapshotErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	snapshot := bundledRuleSetSnapshot{Content: []byte("data"), UpdatedAt: time.Unix(1, 0)}
	if err := writeBundledRuleSetSnapshot(t.Context(), dir, snapshot); err == nil {
		t.Fatal("replacing a directory must fail")
	}
	assertNoRuleSetTemporaryFiles(t, filepath.Dir(dir))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	path := filepath.Join(dir, "untouched")
	if err := writeBundledRuleSetSnapshot(ctx, path, snapshot); !errors.Is(err, context.Canceled) {
		t.Fatalf("write error = %v, want canceled", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled write created a file: %v", err)
	}
}

func TestRuleSetWriteBundledSnapshotTimestamp(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "snapshot")
	snapshot := bundledRuleSetSnapshot{Content: []byte("complete"), UpdatedAt: time.Unix(100, 0)}
	if err := writeBundledRuleSetSnapshot(t.Context(), path, snapshot); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(snapshot.Content, data) {
		t.Fatalf("data = %q, error = %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || !snapshot.UpdatedAt.Equal(info.ModTime()) || info.Mode().Perm() != 0600 {
		t.Fatalf("file metadata = %+v, error = %v", info, err)
	}
}

func assertNoRuleSetTemporaryFiles(t *testing.T, dir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, ".ruleset-*"))
	if err != nil || len(paths) != 0 {
		t.Fatalf("temporary files left behind: %v, error = %v", paths, err)
	}
}
