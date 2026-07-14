package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandlerProbes(t *testing.T) {
	handler := NewHealthHandler(func() error { return nil })

	rr := httptest.NewRecorder()
	handler.Liveness(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("liveness = %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.Readiness(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"ready"`) {
		t.Fatalf("readiness = %d %s", rr.Code, rr.Body.String())
	}
}

func TestHealthHandlerReadinessFailureDoesNotLeakDetails(t *testing.T) {
	handler := NewHealthHandler(func() error { return errors.New("database /secret/path is closed") })
	rr := httptest.NewRecorder()
	handler.Readiness(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "/secret/path") {
		t.Fatalf("readiness response leaked details: %s", rr.Body.String())
	}
}
