package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadAndWriteConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := writeConfigFile(path, []byte(`{"log":{"level":"info"}}`)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	body, err := readConfigFile(path, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"log"`) {
		t.Fatalf("body = %q", body)
	}
	if _, err := readConfigFile(path, 5); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("oversize error = %v", err)
	}
}
