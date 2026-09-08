package core

import (
	"errors"
	"sync"
	"testing"

	"go.etcd.io/bbolt"
	bboltErrors "go.etcd.io/bbolt/errors"
)

func TestEnsureKernelAutostartDefaultPreservesExplicitChoices(t *testing.T) {
	for _, tt := range []struct {
		name, existing string
		created, want  bool
	}{
		{name: "new defaults", created: true, want: true},
		{name: "existing custom config"},
		{name: "explicit false with new config", existing: "false", created: true},
		{name: "explicit true", existing: "true", want: true},
		{name: "unrecognized explicit choice", existing: "custom", created: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := setupSettingsDB(t)
			t.Cleanup(cleanup)
			settings := NewSettingsManager(db)
			if tt.existing != "" {
				if err := settings.Set("kernel_autostart", tt.existing); err != nil {
					t.Fatal(err)
				}
			}
			enabled, err := settings.EnsureKernelAutostartDefault(tt.created)
			if err != nil || enabled != tt.want {
				t.Fatalf("want enabled=%v, got %v: %v", tt.want, enabled, err)
			}
			if tt.existing != "" && settings.Get("kernel_autostart") != tt.existing {
				t.Fatal("existing autostart choice was overwritten")
			}
			if !tt.created && tt.existing == "" && settings.Get("kernel_autostart") != "" {
				t.Fatal("existing config acquired an implicit autostart setting")
			}
		})
	}
}

func TestEnsureKernelAutostartDefaultReportsIncompatibleStoredValue(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	t.Cleanup(cleanup)
	settings := NewSettingsManager(db)
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.Bucket(settingsBucket).CreateBucket([]byte("kernel_autostart"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.EnsureKernelAutostartDefault(true); !errors.Is(err, bboltErrors.ErrIncompatibleValue) {
		t.Fatalf("invalid stored value must abort initialization: %v", err)
	}
}

func TestEnsureKernelAutostartDefaultIsAtomicWithUserChoice(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	t.Cleanup(cleanup)
	settings := NewSettingsManager(db)
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			if _, err := settings.EnsureKernelAutostartDefault(true); err != nil {
				t.Error(err)
			}
		})
	}
	group.Go(func() {
		if err := settings.Set("kernel_autostart", "false"); err != nil {
			t.Error(err)
		}
	})
	group.Wait()
	if settings.Get("kernel_autostart") != "false" {
		t.Fatal("bootstrap overwrote the concurrent explicit false setting")
	}
}

func TestEnsureKernelAutostartDefaultReportsUnavailableDatabase(t *testing.T) {
	for _, settings := range []*SettingsManager{nil, {}} {
		if _, err := settings.EnsureKernelAutostartDefault(true); err == nil {
			t.Fatal("missing database must fail explicitly")
		}
	}
	db, cleanup := setupSettingsDB(t)
	settings := NewSettingsManager(db)
	cleanup()
	if _, err := settings.EnsureKernelAutostartDefault(true); err == nil {
		t.Fatal("closed database error was lost")
	}
}
