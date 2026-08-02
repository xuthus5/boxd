package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupCreateArchive(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := writeTestJSONFile(configPath, map[string]any{"outbounds": []any{}}); err != nil {
		t.Fatal(err)
	}
	svc := NewBackupService(db, configPath, "test")
	targetDir := t.TempDir()
	filename, err := svc.CreateBackupArchive(context.Background(), targetDir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(targetDir, filename)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("backup not created: %v", err)
	}
}

func TestBackupCreateArchiveNilDB(t *testing.T) {
	svc := NewBackupService(nil, "", "")
	if _, err := svc.CreateBackupArchive(context.Background(), t.TempDir()); err == nil {
		t.Fatal("expected error for nil db")
	}
	var nilSvc *BackupService
	if _, err := nilSvc.CreateBackupArchive(context.Background(), t.TempDir()); err == nil {
		t.Fatal("expected error for nil service")
	}
}

func TestBackupCreateArchiveTo(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := writeTestJSONFile(configPath, map[string]any{"outbounds": []any{}}); err != nil {
		t.Fatal(err)
	}
	svc := NewBackupService(db, configPath, "test")
	path := filepath.Join(t.TempDir(), "custom.tar.gz")
	if err := svc.CreateBackupArchiveTo(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("backup not created: %v", err)
	}
}

func TestBackupCreateArchiveToNilDB(t *testing.T) {
	svc := NewBackupService(nil, "", "")
	if err := svc.CreateBackupArchiveTo(context.Background(), filepath.Join(t.TempDir(), "x.tar.gz")); err == nil {
		t.Fatal("expected error for nil db")
	}
}
