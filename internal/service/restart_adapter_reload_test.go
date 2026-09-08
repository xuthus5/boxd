package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestRestartAdapterReloadAndStatusWithNilInstance(t *testing.T) {
	var adapter restartAdapter
	if err := core.ReloadConfig(adapter); err != nil {
		t.Fatal(err)
	}
	if got := adapter.Status(); !reflect.DeepEqual(model.ServiceStatus{}, got) {
		t.Fatalf("want zero status, got %#v", got)
	}
}

func TestRestartAdapterReloadPreservesStoppedDiagnostics(t *testing.T) {
	instance := core.NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), core.NewLogWriter(5))
	adapter := restartAdapter{instance: instance}
	if err := adapter.Restart(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want explicit restart to read the config, got %v", err)
	}
	before := instance.Status()
	if err := core.ReloadConfig(adapter); err != nil {
		t.Fatalf("adapter must forward stopped reload without starting: %v", err)
	}
	if got := adapter.Status(); !reflect.DeepEqual(before, got) {
		t.Fatalf("adapter status must match kernel diagnostics: want=%#v got=%#v", before, got)
	}
}
