package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// BackupHandler 提供已认证的面板备份导出（仅下载，不支持恢复）。
type BackupHandler struct {
	db         *bbolt.DB
	configPath string
	version    string
}

func NewBackupHandler(db *bbolt.DB, configPath, version string) *BackupHandler {
	return &BackupHandler{db: db, configPath: configPath, version: version}
}

// Export GET /api/settings/backup —— 生成并下载 tar.gz 备份。
func (h *BackupHandler) Export(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.db == nil {
		writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, "backup is not configured")
		return
	}

	dir, err := os.MkdirTemp("", "boxd-backup-*")
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to prepare backup")
		return
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := os.Chmod(dir, 0700); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to prepare backup")
		return
	}

	filename := fmt.Sprintf("boxd-backup-%s.tar.gz", time.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(dir, filename)
	if err := core.CreateBackup(h.db, h.configPath, path, h.version); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to create backup")
		return
	}

	file, err := os.Open(path)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to open backup")
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to read backup")
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, file); err != nil {
		// 响应已开始写入，无法再切换为 JSON 错误体。
		return
	}
}
