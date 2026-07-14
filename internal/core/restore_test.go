package core

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
)

func TestRestoreBackupRestoresDatabaseAndConfig(t *testing.T) {
	sourceDB, cleanup := setupSettingsDB(t)
	defer cleanup()
	settings := NewSettingsManager(sourceDB)
	if err := settings.Set("restored", "yes"); err != nil {
		t.Fatal(err)
	}
	sourceConfig := filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(sourceConfig, []byte(`{"restored":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := CreateBackup(sourceDB, sourceConfig, archive, "test"); err != nil {
		t.Fatal(err)
	}

	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := filepath.Join(t.TempDir(), "config", "sing-box.json")
	if err := RestoreBackup(archive, dataDir, configPath); err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}
	restoredDB, err := bbolt.Open(filepath.Join(dataDir, "boxui.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = restoredDB.Close() }()
	if got := NewSettingsManager(restoredDB).Get("restored"); got != "yes" {
		t.Fatalf("restored setting = %q", got)
	}
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(config) != `{"restored":true}` {
		t.Fatalf("restored config = %q", config)
	}
}

func TestRestoreBackupRejectsChecksumMismatch(t *testing.T) {
	entries := map[string][]byte{
		backupDatabaseName: []byte("not-a-database"),
		backupManifestName: []byte(`{"formatVersion":1,"checksums":{"boxui.db":"bad"}}`),
	}
	archive := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := writeBackupArchive(archive, entries); err != nil {
		t.Fatal(err)
	}
	if err := RestoreBackup(archive, t.TempDir(), filepath.Join(t.TempDir(), "config.json")); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestRestoreBackupRollsBackDatabaseWhenConfigRestoreFails(t *testing.T) {
	sourceDB, sourceCleanup := setupSettingsDB(t)
	defer sourceCleanup()
	if err := NewSettingsManager(sourceDB).Set("state", "new"); err != nil {
		t.Fatal(err)
	}
	sourceConfig := filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(sourceConfig, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := CreateBackup(sourceDB, sourceConfig, archive, "test"); err != nil {
		t.Fatal(err)
	}

	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	targetDB, err := bbolt.Open(filepath.Join(dataDir, "boxui.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewSettingsManager(targetDB).Set("state", "old"); err != nil {
		t.Fatal(err)
	}
	if err := targetDB.Close(); err != nil {
		t.Fatal(err)
	}
	blockedParent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedParent, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RestoreBackup(archive, dataDir, filepath.Join(blockedParent, "config.json")); err == nil {
		t.Fatal("expected config restore failure")
	}

	targetDB, err = bbolt.Open(filepath.Join(dataDir, "boxui.db"), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = targetDB.Close() }()
	if got := NewSettingsManager(targetDB).Get("state"); got != "old" {
		t.Fatalf("database was not rolled back, state = %q", got)
	}
}

func TestRestoreBackupRejectsMalformedArchives(t *testing.T) {
	plain := filepath.Join(t.TempDir(), "plain.tar.gz")
	if err := os.WriteFile(plain, []byte("not gzip"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RestoreBackup(plain, t.TempDir(), filepath.Join(t.TempDir(), "config.json")); err == nil {
		t.Fatal("expected invalid gzip error")
	}

	for name, entries := range map[string]map[string][]byte{
		"path traversal": {"../boxui.db": []byte("bad")},
		"bad manifest":   {backupManifestName: []byte("not json")},
		"bad version":    {backupManifestName: []byte(`{"formatVersion":99}`)},
	} {
		t.Run(name, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "bad.tar.gz")
			writeRawArchive(t, archive, entries)
			if err := RestoreBackup(archive, t.TempDir(), filepath.Join(t.TempDir(), "config.json")); err == nil {
				t.Fatal("expected malformed archive error")
			}
		})
	}
}

func writeRawArchive(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, data := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRollbackFileRestoresAndRemovesTargets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target")
	if err := rollbackFile(path, nil, false); err != nil {
		t.Fatalf("remove absent target: %v", err)
	}
	if err := os.WriteFile(path, []byte("temporary"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rollbackFile(path, nil, false); err != nil {
		t.Fatalf("remove existing target: %v", err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rollbackFile(path, []byte("original"), true); err != nil {
		t.Fatalf("restore target: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("rollback data = %q", data)
	}
}
