package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
)

func TestBackupHandlerExportStreamsArchive(t *testing.T) {
	db := newTestDB(t)
	if err := core.NewSettingsManager(db).Set("backup_export", "yes"); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "sing-box.json")
	if err := os.WriteFile(configPath, []byte(`{"log":{"level":"info"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	handler := NewBackupHandler(db, configPath, "test-version")
	rr := httptest.NewRecorder()
	handler.Export(rr, httptest.NewRequest(http.MethodGet, "/api/settings/backup", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "application/gzip" {
		t.Fatalf("Content-Type = %q", got)
	}
	disposition := rr.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, `filename="boxd-backup-`) || !strings.Contains(disposition, `.tar.gz"`) {
		t.Fatalf("Content-Disposition = %q", disposition)
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", rr.Header().Get("Cache-Control"))
	}
	if rr.Body.Len() == 0 {
		t.Fatal("empty backup body")
	}

	entries := readExportedBackupEntries(t, rr.Body.Bytes())
	if len(entries["boxd.db"]) == 0 {
		t.Fatal("missing database entry")
	}
	if string(entries["sing-box.json"]) != `{"log":{"level":"info"}}` {
		t.Fatalf("config entry = %q", entries["sing-box.json"])
	}
	if len(entries["manifest.json"]) == 0 {
		t.Fatal("missing manifest entry")
	}
}

func TestBackupHandlerExportRequiresConfiguration(t *testing.T) {
	rr := httptest.NewRecorder()
	NewBackupHandler(nil, "", "test").Export(rr, httptest.NewRequest(http.MethodGet, "/api/settings/backup", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	var nilHandler *BackupHandler
	nilHandler.Export(rr, httptest.NewRequest(http.MethodGet, "/api/settings/backup", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil handler status = %d", rr.Code)
	}
}

func TestBackupHandlerExportReportsCreateFailure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "boxd.db")
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	handler := NewBackupHandler(db, filepath.Join(t.TempDir(), "missing.json"), "test")
	rr := httptest.NewRecorder()
	handler.Export(rr, httptest.NewRequest(http.MethodGet, "/api/settings/backup", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "failed to create backup") {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func readExportedBackupEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	defer func() { _ = gzipReader.Close() }()

	tarReader := tar.NewReader(gzipReader)
	entries := make(map[string][]byte)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar: %v", err)
		}
		body, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read entry: %v", err)
		}
		entries[header.Name] = body
	}
	return entries
}
