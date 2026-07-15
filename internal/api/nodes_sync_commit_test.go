package api

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type failingManagedGroupStore struct{}

func (failingManagedGroupStore) SetURLTestManagedGroups([]string) error {
	return errors.New("saving managed groups")
}

type recordingManagedGroupStore struct{ called bool }

func (s *recordingManagedGroupStore) SetURLTestManagedGroups([]string) error {
	s.called = true
	return nil
}

func TestSyncCommitRollsBackConfigWhenManagedGroupSaveFails(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	previous := []byte(`{"outbounds":[],"route":{"final":"direct"}}`)
	if err := os.WriteFile(configPath, previous, 0o600); err != nil {
		t.Fatalf("writing previous config: %v", err)
	}
	commit := syncCommit{path: configPath, previous: previous, groups: failingManagedGroupStore{}}

	if err := commit.write(map[string]any{"outbounds": []any{}, "route": map[string]any{}}, []string{"group"}); err == nil {
		t.Fatal("expected managed group save error")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading rolled back config: %v", err)
	}
	if !bytes.Equal(after, previous) {
		t.Fatalf("config was not rolled back\nwant: %s\ngot: %s", previous, after)
	}
}

func TestSyncCommitPreservesConfigWhenAtomicWriteFails(t *testing.T) {
	const configPath = "/proc/version"
	previous, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading protected config: %v", err)
	}
	store := &recordingManagedGroupStore{}
	commit := syncCommit{path: configPath, previous: previous, groups: store}

	if err := commit.write(map[string]any{"outbounds": []any{}}, []string{"group"}); err == nil {
		t.Fatal("expected atomic write error")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading protected config after failure: %v", err)
	}
	if !bytes.Equal(after, previous) || store.called {
		t.Fatalf("failed write changed config or saved group tracking")
	}
}
