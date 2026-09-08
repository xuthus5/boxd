package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestInitialConfigConcurrentCreationPreservesWinner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	const writers = 12
	var created atomic.Int32
	var group sync.WaitGroup
	for index := range writers {
		group.Go(func() {
			body := []byte(fmt.Sprintf(`{"writer":%d}`, index))
			installed, err := writeInitialConfig(path, body)
			if err != nil {
				t.Errorf("write: %v", err)
				return
			}
			if installed {
				created.Add(1)
			}
		})
	}
	group.Wait()
	if created.Load() != 1 {
		t.Fatalf("want exactly one successful creation, got %d", created.Load())
	}
	assertNoInitialConfigTemps(t, filepath.Dir(path))
}

func TestInitialConfigNeverReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	const original = "user config\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	created, err := writeInitialConfig(path, []byte("replacement"))
	if err != nil || created {
		t.Fatalf("existing config: created=%v, err=%v", created, err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != original {
		t.Fatalf("existing config changed: %q, %v", actual, err)
	}
	assertNoInitialConfigTemps(t, filepath.Dir(path))
}

func TestInitialConfigRejectsInvalidPaths(t *testing.T) {
	dir := t.TempDir()
	parentFile := filepath.Join(dir, "file")
	if err := os.WriteFile(parentFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		path string
	}{
		{name: "directory", path: dir},
		{name: "parent is file", path: filepath.Join(parentFile, "config.json")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := writeInitialConfig(tt.path, []byte("{}")); err == nil {
				t.Fatal("expected path error")
			}
		})
	}
	assertNoInitialConfigTemps(t, dir)
}

func TestInitialConfigWritePropagatesClosedFileError(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "closed-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writeAndCloseInitialConfig(file, []byte("{}")); err == nil {
		t.Fatal("expected write error")
	}
}

func TestConfigFileExistsPreservesValidSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	exists, err := configFileExists(link)
	if err != nil || !exists {
		t.Fatalf("existing symlink: exists=%v, err=%v", exists, err)
	}
}

func assertNoInitialConfigTemps(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".boxd-config-") {
			t.Errorf("temporary config was not cleaned up: %s", entry.Name())
		}
	}
}
