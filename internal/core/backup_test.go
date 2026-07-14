package core

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateBackupIncludesSnapshotConfigAndManifest(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	settings := NewSettingsManager(db)
	if err := settings.Set("url_test", "https://example.test/"); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "sing-box.json")
	if err := os.WriteFile(configPath, []byte(`{"log":{"level":"info"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(t.TempDir(), "boxui-backup.tar.gz")

	if err := CreateBackup(db, configPath, backupPath, "test-version"); err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("backup mode = %o", info.Mode().Perm())
	}

	entries := readBackupEntries(t, backupPath)
	if len(entries[backupDatabaseName]) == 0 || string(entries[backupConfigName]) != `{"log":{"level":"info"}}` {
		t.Fatalf("backup entries = %#v", entries)
	}
	var manifest BackupManifest
	if err := json.Unmarshal(entries[backupManifestName], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.FormatVersion != backupFormatVersion || manifest.AppVersion != "test-version" {
		t.Fatalf("manifest = %#v", manifest)
	}
	if err := verifyBackupChecksums(entries, manifest); err != nil {
		t.Fatalf("checksum verification failed: %v", err)
	}
}

func TestCreateBackupAllowsMissingKernelConfig(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	backupPath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := CreateBackup(db, filepath.Join(t.TempDir(), "missing.json"), backupPath, "test"); err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}
	entries := readBackupEntries(t, backupPath)
	if _, found := entries[backupConfigName]; found {
		t.Fatal("missing kernel config was unexpectedly archived")
	}
}

func TestVerifyBackupChecksumsRejectsMissingAndChangedEntries(t *testing.T) {
	manifest := BackupManifest{Checksums: map[string]string{"missing": "value"}}
	if err := verifyBackupChecksums(map[string][]byte{}, manifest); err == nil {
		t.Fatal("expected missing entry error")
	}
	manifest = BackupManifest{Checksums: backupChecksums(map[string][]byte{"file": []byte("expected")})}
	if err := verifyBackupChecksums(map[string][]byte{"file": []byte("changed")}, manifest); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestCreateBackupReportsFilesystemErrors(t *testing.T) {
	db, cleanup := setupSettingsDB(t)
	defer cleanup()
	if err := CreateBackup(db, t.TempDir(), filepath.Join(t.TempDir(), "backup.tar.gz"), "test"); err == nil {
		t.Fatal("expected kernel config read error")
	}
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CreateBackup(db, filepath.Join(t.TempDir(), "missing.json"), filepath.Join(blocked, "backup.tar.gz"), "test"); err == nil {
		t.Fatal("expected backup directory error")
	}
}

func readBackupEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = gz.Close() }()
	entries := make(map[string][]byte)
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		entries[header.Name] = data
	}
	return entries
}
