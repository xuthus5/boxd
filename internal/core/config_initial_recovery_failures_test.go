package core

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInitialRecoveryFailuresKeepTheOriginalConfig(t *testing.T) {
	cause := errors.New("injected filesystem failure")
	for _, phase := range []string{"compare", "prepare", "claim", "secure", "publish"} {
		t.Run(phase, func(t *testing.T) {
			path, snapshot := initialRecoveryFixture(t)
			ops := defaultInitialRecoveryIO()
			switch phase {
			case "compare":
				ops.match = func(string, initialConfigSnapshot) (bool, error) { return false, cause }
			case "prepare":
				ops.prepare = func(string, []byte) (string, error) { return "", cause }
			case "claim":
				ops.claim = func(string) (string, error) { return "", cause }
			case "secure":
				ops.secure = func(string) error { return cause }
			case "publish":
				ops.publish = func(string, string) error { return cause }
			}
			recovery := initialRecovery{path: path, snapshot: snapshot, io: ops}
			if changed, err := recovery.apply([]byte("new")); changed || !errors.Is(err, cause) {
				t.Fatalf("want recoverable filesystem error: %v, %v", changed, err)
			}
			body, err := os.ReadFile(path)
			if err != nil || string(body) != "{}" {
				t.Fatalf("filesystem failure lost original config: %q, %v", body, err)
			}
			assertNoInitialConfigTemps(t, filepath.Dir(path))
		})
	}
}

func TestInitialRecoveryRechecksTheClaimedFile(t *testing.T) {
	path, snapshot := initialRecoveryFixture(t)
	const custom = `{"custom":"edited during preparation"}`
	ops := defaultInitialRecoveryIO()
	ops.claim = func(path string) (string, error) {
		if err := os.WriteFile(path, []byte(custom), 0600); err != nil {
			return "", err
		}
		return claimInitialConfig(path)
	}
	recovery := initialRecovery{path: path, snapshot: snapshot, io: ops}
	if changed, err := recovery.apply([]byte("new")); changed || err != nil {
		t.Fatalf("concurrent edit should be restored: %v, %v", changed, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != custom {
		t.Fatalf("post-claim CAS lost user data: %q, %v", body, err)
	}
	assertNoInitialConfigTemps(t, filepath.Dir(path))
}

func TestInitialRecoveryPublicationCannotOverwriteAnotherWriter(t *testing.T) {
	path, snapshot := initialRecoveryFixture(t)
	const custom = `{"custom":"published concurrently"}`
	ops := defaultInitialRecoveryIO()
	ops.publish = func(staged, path string) error {
		if err := os.WriteFile(path, []byte(custom), 0600); err != nil {
			return err
		}
		return os.Link(staged, path)
	}
	recovery := initialRecovery{path: path, snapshot: snapshot, io: ops}
	if changed, err := recovery.apply([]byte("new")); changed || err != nil {
		t.Fatalf("concurrent publication should win: %v, %v", changed, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != custom {
		t.Fatalf("exclusive publication overwrote user data: %q, %v", body, err)
	}
	assertInitialRecoveryBackup(t, path, "{}")
}

func TestInitialConfigPropagatesSyncFailureAndClosesFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows pipe flushing requires a concurrent reader")
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	if err := writeAndCloseInitialConfig(writer, []byte("{}")); err == nil {
		t.Fatal("a pipe cannot provide durable config storage")
	}
	body, err := io.ReadAll(reader)
	if err != nil || string(body) != "{}" {
		t.Fatalf("sync failure must still close the file: %q, %v", body, err)
	}
}
