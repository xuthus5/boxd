package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const legacyBootstrapFixture = `{"log":{"level":"info","timestamp":true},"inbounds":[{"type":"mixed","tag":"mixed-in","listen":"::","listen_port":1080}],"outbounds":[{"type":"direct","tag":"direct"},{"type":"block","tag":"block"}],"route":{"final":"direct"}}`

func TestBootstrapRecoversUninitializedFiles(t *testing.T) {
	for _, tt := range []struct{ name, body string }{
		{name: "zero bytes"},
		{name: "whitespace", body: " \n\t"},
		{name: "empty object", body: "{}"},
		{name: "null", body: "null"},
		{name: "known legacy defaults", body: legacyBootstrapFixture},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
				t.Fatal(err)
			}
			if changed, err := EnsureDefaultConfig(t.Context(), path, filepath.Dir(path)); err != nil || !changed {
				t.Fatalf("recover defaults: changed=%v, error=%v", changed, err)
			}
			cfg := readInboundRegressionConfig(t, path)
			if cfg["inbounds"] == nil || cfg["dns"] == nil || cfg["experimental"] == nil {
				t.Fatalf("recovery did not install complete defaults: %#v", cfg)
			}
			assertInitialRecoveryBackup(t, path, tt.body)
			if changed, err := EnsureDefaultConfig(t.Context(), path, filepath.Dir(path)); err != nil || changed {
				t.Fatalf("recovery must be idempotent: changed=%v, error=%v", changed, err)
			}
		})
	}
}

func assertInitialRecoveryBackup(t *testing.T, path, original string) {
	t.Helper()
	backups, err := filepath.Glob(path + ".boxd-backup-*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("want exactly one original backup: %v, %v", backups, err)
	}
	body, err := os.ReadFile(backups[0])
	if err != nil || string(body) != original {
		t.Fatalf("backup does not preserve original bytes: %q, %v", body, err)
	}
	for _, file := range []string{path, backups[0]} {
		info, err := os.Stat(file)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("config or backup is not private: %v, %v", info, err)
		}
	}
	assertNoInitialConfigTemps(t, filepath.Dir(path))
}

func TestBootstrapPreservesUnmanagedConfigurations(t *testing.T) {
	for _, tt := range []struct{ name, body string }{
		{name: "intentional no inbounds", body: `{"inbounds":[]}`},
		{name: "custom logging", body: `{"log":{"level":"debug"}}`},
		{name: "custom legacy port", body: strings.Replace(legacyBootstrapFixture, "1080", "8080", 1)},
		{name: "malformed JSON", body: "{"},
		{name: "array", body: "[]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
				t.Fatal(err)
			}
			if changed, err := EnsureDefaultConfig(t.Context(), path, "unused"); err != nil || changed {
				t.Fatalf("custom config unexpectedly initialized: %v, %v", changed, err)
			}
			body, err := os.ReadFile(path)
			if err != nil || string(body) != tt.body {
				t.Fatalf("custom config changed: %q, %v", body, err)
			}
		})
	}
}

func TestBootstrapRecoveryRecognizesExactLegacyTemplates(t *testing.T) {
	for _, listen := range []string{"::", "127.0.0.1", "0.0.0.0"} {
		t.Run(listen, func(t *testing.T) {
			body := strings.Replace(legacyBootstrapFixture, `"::"`, `"`+listen+`"`, 1)
			if !bootstrapConfigNeedsRecovery([]byte(body)) {
				t.Fatal("known legacy defaults were not recognized")
			}
			var cfg map[string]any
			if err := json.Unmarshal([]byte(body), &cfg); err != nil {
				t.Fatal(err)
			}
			cfg["dns"] = map[string]any{"servers": []any{}}
			custom, err := json.Marshal(cfg)
			if err != nil || bootstrapConfigNeedsRecovery(custom) {
				t.Fatalf("additional user modules must be preserved: %v", err)
			}
		})
	}
}

func TestBootstrapRecoveryPreservesValidSymlinks(t *testing.T) {
	dir := t.TempDir()
	target, path := filepath.Join(dir, "target.json"), filepath.Join(dir, "config.json")
	if err := os.WriteFile(target, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if changed, err := EnsureDefaultConfig(t.Context(), path, "unused"); err != nil || changed {
		t.Fatalf("symlink must remain user-owned: %v, %v", changed, err)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink was replaced: %v", err)
	}
}
