package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

func TestCoreRestarterWithNilInstance(t *testing.T) {
	var restarter coreRestarter
	if err := restarter.Restart(); err != nil {
		t.Fatal(err)
	}
	if err := core.ReloadConfig(restarter); err != nil {
		t.Fatal(err)
	}
}

func TestCoreRestarterReloadPreservesStoppedKernel(t *testing.T) {
	instance := core.NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), core.NewLogWriter(5))
	restarter := coreRestarter{instance: instance}
	if err := restarter.Restart(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("explicit restart must attempt to load the config, got %v", err)
	}
	before := instance.Status()
	if err := core.ReloadConfig(restarter); err != nil {
		t.Fatalf("stopped reload must not attempt a start: %v", err)
	}
	if after := instance.Status(); !reflect.DeepEqual(before, after) {
		t.Fatalf("stopped kernel diagnostics changed: before=%#v after=%#v", before, after)
	}
}
