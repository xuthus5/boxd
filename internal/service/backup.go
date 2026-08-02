package service

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// BackupService 提供面板备份导出用例逻辑。
type BackupService struct {
	db         *bbolt.DB
	configPath string
	version    string
}

// NewBackupService 构造备份用例服务。
func NewBackupService(db *bbolt.DB, configPath, version string) *BackupService {
	return &BackupService{db: db, configPath: configPath, version: version}
}

// CreateBackupArchive 生成 tar.gz 备份到指定路径，返回文件名。
func (s *BackupService) CreateBackupArchive(_ context.Context, targetDir string) (string, error) {
	if s == nil || s.db == nil {
		return "", Errorf(503, model.ErrorUnavailable, "backup is not configured")
	}
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return "", Errorf(500, model.ErrorInternal, "failed to prepare backup directory")
	}
	filename := "boxd-backup-" + time.Now().UTC().Format("20060102T150405Z") + ".tar.gz"
	path := filepath.Join(targetDir, filename)
	if err := core.CreateBackup(s.db, s.configPath, path, s.version); err != nil {
		return "", Errorf(500, model.ErrorInternal, "failed to create backup")
	}
	return filename, nil
}

// CreateBackupArchiveTo 生成备份到显式目标路径（桌面端文件对话框用）。
func (s *BackupService) CreateBackupArchiveTo(_ context.Context, path string) error {
	if s == nil || s.db == nil {
		return Errorf(503, model.ErrorUnavailable, "backup is not configured")
	}
	if err := core.CreateBackup(s.db, s.configPath, path, s.version); err != nil {
		return Errorf(500, model.ErrorInternal, "failed to create backup")
	}
	return nil
}
