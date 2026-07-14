package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

const (
	backupFormatVersion = 1
	backupDatabaseName  = "boxui.db"
	backupConfigName    = "sing-box.json"
	backupManifestName  = "manifest.json"
)

type BackupManifest struct {
	FormatVersion int               `json:"formatVersion"`
	AppVersion    string            `json:"appVersion"`
	CreatedAt     time.Time         `json:"createdAt"`
	Checksums     map[string]string `json:"checksums"`
}

func CreateBackup(db *bbolt.DB, configPath, outputPath, version string) error {
	entries := make(map[string][]byte, 2)
	var database bytes.Buffer
	if err := db.View(func(tx *bbolt.Tx) error {
		_, err := tx.WriteTo(&database)
		return err
	}); err != nil {
		return fmt.Errorf("snapshot database: %w", err)
	}
	entries[backupDatabaseName] = database.Bytes()

	config, err := os.ReadFile(configPath)
	if err == nil {
		entries[backupConfigName] = config
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read kernel config: %w", err)
	}

	manifest := BackupManifest{
		FormatVersion: backupFormatVersion,
		AppVersion:    version,
		CreatedAt:     time.Now().UTC(),
		Checksums:     backupChecksums(entries),
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode backup manifest: %w", err)
	}
	entries[backupManifestName] = manifestData
	return writeBackupArchive(outputPath, entries)
}

func backupChecksums(entries map[string][]byte) map[string]string {
	checksums := make(map[string]string, len(entries))
	for name, data := range entries {
		sum := sha256.Sum256(data)
		checksums[name] = hex.EncodeToString(sum[:])
	}
	return checksums
}

func verifyBackupChecksums(entries map[string][]byte, manifest BackupManifest) error {
	for name, expected := range manifest.Checksums {
		data, found := entries[name]
		if !found {
			return fmt.Errorf("backup entry %q is missing", name)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != expected {
			return fmt.Errorf("backup entry %q checksum mismatch", name)
		}
	}
	return nil
}

func writeBackupArchive(outputPath string, entries map[string][]byte) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(outputPath), filepath.Base(outputPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary backup: %w", err)
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0600); err != nil {
		return fmt.Errorf("set backup permissions: %w", err)
	}
	gzipWriter := gzip.NewWriter(temp)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, name := range []string{backupDatabaseName, backupConfigName, backupManifestName} {
		if data, found := entries[name]; found {
			if err := writeBackupEntry(tarWriter, name, data); err != nil {
				return err
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("close backup tar: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close backup gzip: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync backup: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close backup: %w", err)
	}
	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf("commit backup: %w", err)
	}
	committed = true
	return nil
}

func writeBackupEntry(writer *tar.Writer, name string, data []byte) error {
	header := &tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), ModTime: time.Now().UTC()}
	if err := writer.WriteHeader(header); err != nil {
		return fmt.Errorf("write backup header %q: %w", name, err)
	}
	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("write backup entry %q: %w", name, err)
	}
	return nil
}
