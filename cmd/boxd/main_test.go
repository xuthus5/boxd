package main

import (
	"path/filepath"
	"testing"
)

func TestResolveDataDirUsesAbsoluteConfiguredPath(t *testing.T) {
	configured := filepath.Join(t.TempDir(), "profile")
	resolved, err := resolveDataDir(configured)
	if err != nil {
		t.Fatalf("resolveDataDir() error = %v", err)
	}
	want, err := filepath.Abs(configured)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if resolved != want {
		t.Fatalf("resolveDataDir() = %q, want %q", resolved, want)
	}
}
