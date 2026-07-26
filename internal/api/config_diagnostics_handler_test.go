package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestConfigDiagnosticsReturnsPersistedReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	handler := NewConfigHandler(path, nil, nil, nil, nil, nil)
	rr := httptest.NewRecorder()
	handler.ConfigDiagnostics(rr, httptest.NewRequest(http.MethodGet, "/api/config/diagnostics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	report := decodeBody[model.ConfigDiagnostics](t, rr)
	if report.Counts.Outbounds != 1 || report.Status != model.ConfigDiagnosticsWarning {
		t.Fatalf("report = %+v", report)
	}
}

func TestConfigDiagnosticsReturnsMissingConfigIssue(t *testing.T) {
	handler := NewConfigHandler(filepath.Join(t.TempDir(), "missing.json"), nil, nil, nil, nil, nil)
	rr := httptest.NewRecorder()
	handler.ConfigDiagnostics(rr, httptest.NewRequest(http.MethodGet, "/api/config/diagnostics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	report := decodeBody[model.ConfigDiagnostics](t, rr)
	if report.Status != model.ConfigDiagnosticsError || len(report.Issues) != 1 || report.Issues[0].Code != "config_missing" {
		t.Fatalf("report = %+v", report)
	}
}
