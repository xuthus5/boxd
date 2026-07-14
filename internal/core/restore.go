package core

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

const maxBackupEntryBytes = 512 << 20

func RestoreBackup(archivePath, dataDir, configPath string) error {
	entries, manifest, err := loadBackupArchive(archivePath)
	if err != nil {
		return err
	}
	if err := verifyBackupChecksums(entries, manifest); err != nil {
		return err
	}
	database, found := entries[backupDatabaseName]
	if !found {
		return fmt.Errorf("backup database is missing")
	}
	if err := validateDatabaseSnapshot(database); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	databasePath := filepath.Join(dataDir, "boxui.db")
	previousDatabase, databaseExisted, err := readOptionalFile(databasePath)
	if err != nil {
		return fmt.Errorf("read existing database: %w", err)
	}
	if err := preserveRestoreCopy(databasePath, previousDatabase, databaseExisted); err != nil {
		return fmt.Errorf("preserve existing database: %w", err)
	}
	if err := atomicReplaceFile(databasePath, database); err != nil {
		return fmt.Errorf("restore database: %w", err)
	}
	if config, found := entries[backupConfigName]; found {
		previousConfig, configExisted, err := readOptionalFile(configPath)
		if err != nil {
			_ = rollbackFile(databasePath, previousDatabase, databaseExisted)
			return fmt.Errorf("read existing kernel config: %w", err)
		}
		if err := preserveRestoreCopy(configPath, previousConfig, configExisted); err != nil {
			_ = rollbackFile(databasePath, previousDatabase, databaseExisted)
			return fmt.Errorf("preserve existing kernel config: %w", err)
		}
		if err := atomicReplaceFile(configPath, config); err != nil {
			_ = rollbackFile(databasePath, previousDatabase, databaseExisted)
			return fmt.Errorf("restore kernel config: %w", err)
		}
	}
	return nil
}

func loadBackupArchive(path string) (map[string][]byte, BackupManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, BackupManifest{}, fmt.Errorf("open backup: %w", err)
	}
	defer func() { _ = file.Close() }()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, BackupManifest{}, fmt.Errorf("open backup gzip: %w", err)
	}
	defer func() { _ = gzipReader.Close() }()
	entries, err := readValidatedTar(tar.NewReader(gzipReader))
	if err != nil {
		return nil, BackupManifest{}, err
	}
	var manifest BackupManifest
	if err := json.Unmarshal(entries[backupManifestName], &manifest); err != nil {
		return nil, BackupManifest{}, fmt.Errorf("decode backup manifest: %w", err)
	}
	if manifest.FormatVersion != backupFormatVersion {
		return nil, BackupManifest{}, fmt.Errorf("unsupported backup format version")
	}
	return entries, manifest, nil
}

func readValidatedTar(reader *tar.Reader) (map[string][]byte, error) {
	allowed := map[string]bool{backupDatabaseName: true, backupConfigName: true, backupManifestName: true}
	entries := make(map[string][]byte, len(allowed))
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read backup archive: %w", err)
		}
		if !allowed[header.Name] || header.Typeflag != tar.TypeReg || header.Size > maxBackupEntryBytes {
			return nil, fmt.Errorf("invalid backup entry")
		}
		data, err := io.ReadAll(io.LimitReader(reader, maxBackupEntryBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read backup entry: %w", err)
		}
		if len(data) > maxBackupEntryBytes {
			return nil, fmt.Errorf("backup entry is too large")
		}
		entries[header.Name] = data
	}
	if len(entries[backupManifestName]) == 0 {
		return nil, fmt.Errorf("backup manifest is missing")
	}
	return entries, nil
}

func validateDatabaseSnapshot(data []byte) error {
	dir, err := os.MkdirTemp("", "boxui-restore-*")
	if err != nil {
		return fmt.Errorf("create restore validation directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "boxui.db")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write database validation snapshot: %w", err)
	}
	db, err := bbolt.Open(path, 0600, &bbolt.Options{ReadOnly: true, Timeout: time.Second})
	if err != nil {
		return fmt.Errorf("validate database snapshot: %w", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close database validation snapshot: %w", err)
	}
	return nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, err
}

func preserveRestoreCopy(path string, data []byte, exists bool) error {
	if !exists {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	backupPath := fmt.Sprintf("%s.pre-restore-%s", path, time.Now().UTC().Format("20060102T150405Z"))
	return os.WriteFile(backupPath, data, 0600)
}

func rollbackFile(path string, data []byte, existed bool) error {
	if !existed {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return atomicReplaceFile(path, data)
}

func atomicReplaceFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".restore-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
