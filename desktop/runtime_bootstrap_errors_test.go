package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
	bboltErrors "go.etcd.io/bbolt/errors"

	"github.com/xuthus5/boxd/internal/core"
)

func TestDesktopBootstrapReportsFilesystemFailures(t *testing.T) {
	for _, phase := range []string{"data directory", "config path", "database"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			cfg := desktopConfig{DataDir: dir, ConfigPath: filepath.Join(dir, "config.json"), Username: "admin"}
			if err := os.WriteFile(cfg.ConfigPath, []byte(`{"log":{}}`), 0600); err != nil {
				t.Fatal(err)
			}
			switch phase {
			case "data directory":
				cfg.DataDir = cfg.ConfigPath
			case "config path":
				cfg.ConfigPath = dir
			case "database":
				if err := os.Mkdir(filepath.Join(dir, "boxd.db"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := initRuntime(cfg); err == nil {
				t.Fatalf("%s failure was ignored", phase)
			}
		})
	}
}

func TestDesktopBootstrapReportsCredentialAndAutostartErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	db, err := bbolt.Open(filepath.Join(dir, "boxd.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewSettingsManager(db)
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.Bucket([]byte("settings")).CreateBucket([]byte("kernel_autostart"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := desktopConfig{DataDir: dir, ConfigPath: filepath.Join(dir, "config.json"), Username: "admin"}
	if err := initializeDesktopCredentials(settings, cfg); err == nil {
		t.Fatal("credential database failure was ignored")
	}
	if _, err := initRuntime(cfg); !errors.Is(err, bboltErrors.ErrIncompatibleValue) {
		t.Fatalf("autostart transaction failure was ignored: %v", err)
	}
}

func TestDesktopBackgroundStartupUsesInjectedLifecycle(t *testing.T) {
	for _, fails := range []bool{false, true} {
		name := "starts"
		if fails {
			name = "fails before background start"
		}
		t.Run(name, func(t *testing.T) {
			rt := newTestRuntimeWithService(t)
			cause := errors.New("start failed")
			called := 0
			configureDesktopBackground(rt, func() error {
				called++
				if fails {
					return cause
				}
				return nil
			})
			err := rt.startFn()
			if called != 1 || errors.Is(err, cause) != fails {
				t.Fatalf("unexpected startup: calls=%d, error=%v", called, err)
			}
			rt.backgroundStop()
		})
	}
}
