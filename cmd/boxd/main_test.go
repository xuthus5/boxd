package main

import (
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/adapters/inbound/httpapi"
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

func TestDefaultWebAddressUsesPort7662(t *testing.T) {
	if httpapi.DefaultAddress != "[::]:7662" {
		t.Fatalf("DefaultAddress = %q, want [::]:7662", httpapi.DefaultAddress)
	}
}
