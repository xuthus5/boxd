package controlplane

import (
	"context"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// BackupService exposes consistent snapshot creation and inspection.
type BackupService struct {
	store      control.BackupStore
	defaultDir string
}

var _ control.BackupAPI = (*BackupService)(nil)

func NewBackupService(store control.BackupStore, defaultDir string) *BackupService {
	return &BackupService{store: store, defaultDir: strings.TrimSpace(defaultDir)}
}

func (s *BackupService) Create(ctx context.Context, options control.BackupCreateOptions) (control.BackupInfo, error) {
	if s.store == nil {
		return control.BackupInfo{}, capabilityUnavailable("backup.create", "backup store")
	}
	directory := strings.TrimSpace(options.DestinationDir)
	if directory == "" {
		directory = s.defaultDir
	}
	if directory == "" {
		return control.BackupInfo{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "backup.create", "backup destination directory is required")
	}
	info, err := s.store.Create(ctx, directory, time.Now().UTC())
	if err != nil {
		return control.BackupInfo{}, wrapStoreError("backup.create", "create database backup", err)
	}
	return info, nil
}

func (s *BackupService) Inspect(ctx context.Context, path string) (control.BackupInfo, error) {
	if s.store == nil {
		return control.BackupInfo{}, capabilityUnavailable("backup.inspect", "backup store")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return control.BackupInfo{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "backup.inspect", "backup path is required")
	}
	info, err := s.store.Inspect(ctx, path)
	if err != nil {
		return control.BackupInfo{}, wrapStoreError("backup.inspect", "inspect database backup", err)
	}
	return info, nil
}

func (s *BackupService) List(ctx context.Context, options control.BackupListOptions) ([]control.BackupInfo, error) {
	if s.store == nil {
		return nil, capabilityUnavailable("backup.list", "backup store")
	}
	directory := strings.TrimSpace(options.DestinationDir)
	if directory == "" {
		directory = s.defaultDir
	}
	if directory == "" {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "backup.list", "backup directory is required")
	}
	infos, err := s.store.List(ctx, directory)
	if err != nil {
		return nil, wrapStoreError("backup.list", "list database backups", err)
	}
	return infos, nil
}
