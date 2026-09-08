package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func initialRecoveryFixture(t *testing.T) (string, initialConfigSnapshot) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, initialize, err := inspectBootstrapConfig(path)
	if err != nil || !initialize || snapshot == nil {
		t.Fatalf("cannot inspect fixture: %v, %v", initialize, err)
	}
	return path, *snapshot
}

func TestRecoverInitialConfigPreservesConcurrentEdit(t *testing.T) {
	path, snapshot := initialRecoveryFixture(t)
	const custom = `{"log":{"level":"debug"}}`
	if err := os.WriteFile(path, []byte(custom), 0600); err != nil {
		t.Fatal(err)
	}
	// 即使修改时间保持不变，内容 CAS 仍须发现并发写入。
	if err := os.Chtimes(path, snapshot.info.ModTime(), snapshot.info.ModTime()); err != nil {
		t.Fatal(err)
	}
	changed, err := recoverInitialConfig(path, []byte(`{"inbounds":[]}`), snapshot)
	if err != nil || changed {
		t.Fatalf("concurrent user edit was replaced: %v, %v", changed, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != custom {
		t.Fatalf("user edit lost: %q, %v", body, err)
	}
	assertNoInitialConfigTemps(t, filepath.Dir(path))
}

func TestRecoverInitialConfigPreservesConcurrentReplacement(t *testing.T) {
	path, snapshot := initialRecoveryFixture(t)
	replacement := filepath.Join(filepath.Dir(path), "replacement.json")
	if err := os.WriteFile(replacement, snapshot.body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	if changed, err := recoverInitialConfig(path, []byte("new"), snapshot); err != nil || changed {
		t.Fatalf("same bytes on another inode must not match: %v, %v", changed, err)
	}
}

func TestRecoverInitialConfigConcurrentWritersPreserveWinner(t *testing.T) {
	path, snapshot := initialRecoveryFixture(t)
	const writers = 12
	var changed atomic.Int32
	var group sync.WaitGroup
	for writer := range writers {
		group.Go(func() {
			body := []byte(fmt.Sprintf(`{"writer":%d}`, writer))
			ok, err := recoverInitialConfig(path, body, snapshot)
			if err != nil {
				t.Errorf("recover: %v", err)
			}
			if ok {
				changed.Add(1)
			}
		})
	}
	group.Wait()
	if changed.Load() != 1 {
		t.Fatalf("want one recovery winner, got %d", changed.Load())
	}
	assertInitialRecoveryBackup(t, path, "{}")
}

func TestInitialRecoveryRestoresClaimWithoutOverwritingNewWriter(t *testing.T) {
	path, _ := initialRecoveryFixture(t)
	backup, err := claimInitialConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	const custom = `{"custom":"winner"}`
	if err := os.WriteFile(path, []byte(custom), 0600); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("config changed while preparing recovery")
	if changed, err := restoreInitialClaim(path, backup, cause); changed || !errors.Is(err, cause) {
		t.Fatalf("restore must preserve concurrent winner: %v, %v", changed, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != custom {
		t.Fatalf("concurrent winner lost: %q, %v", body, err)
	}
	assertInitialRecoveryBackup(t, path, "{}")
}

func TestInitialRecoveryRestoresClaimAfterPublishFailure(t *testing.T) {
	path, _ := initialRecoveryFixture(t)
	backup, err := claimInitialConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("publish failed")
	if changed, err := restoreInitialClaim(path, backup, cause); changed || !errors.Is(err, cause) {
		t.Fatalf("restore failed: %v, %v", changed, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "{}" {
		t.Fatalf("original config was not restored: %q, %v", body, err)
	}
	if _, err := os.Stat(backup); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restored claim left a redundant backup: %v", err)
	}
}

func TestInitialRecoveryReportsFilesystemFailures(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	if _, err := claimInitialConfig(missing); err == nil {
		t.Fatal("claiming a missing config must fail")
	}
	if _, err := claimInitialConfig(filepath.Join(missing, "config.json")); err == nil {
		t.Fatal("claiming within a missing directory must fail")
	}
	if err := secureInitialBackup(missing); err == nil {
		t.Fatal("securing a missing backup must fail")
	}
	if _, err := restoreInitialClaim(missing, missing+".backup", nil); err == nil {
		t.Fatal("restoring a missing backup must fail")
	}
	entries, err := os.ReadDir(filepath.Dir(missing))
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed claim left temporary files: %v, %v", entries, err)
	}
}

func TestInitialRecoverySnapshotRejectsMissingOrInvalidPaths(t *testing.T) {
	path, snapshot := initialRecoveryFixture(t)
	for _, tt := range []struct{ name, path string }{
		{name: "missing", path: path + ".missing"},
		{name: "directory", path: filepath.Dir(path)},
		{name: "invalid parent", path: filepath.Join(path, "config.json")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if matches, _ := initialSnapshotMatches(tt.path, snapshot); matches {
				t.Fatal("invalid path must not match original config")
			}
		})
	}
}
