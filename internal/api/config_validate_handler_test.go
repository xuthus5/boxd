package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

func TestValidateConfigAcceptsValidDocument(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	rr := httptest.NewRecorder()
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate", `{"outbounds":[{"type":"direct","tag":"direct"}]}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var envelope model.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Status != model.StatusOK {
		t.Fatalf("status = %s", envelope.Status)
	}
}

func TestValidateConfigRejectsInvalidRuntimeConfig(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	rr := httptest.NewRecorder()
	// missing required type fields should fail sing-box option validation
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate", `{"inbounds":[{"tag":"broken"}]}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var envelope model.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error == nil || envelope.Error.Code != model.ErrorConfigInvalidRuntime {
		t.Fatalf("error = %#v", envelope.Error)
	}
}

func TestValidateConfigRejectsInvalidJSON(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	rr := httptest.NewRecorder()
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate", `not-json`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestValidateConfigRejectsEmptyBody(t *testing.T) {
	handler, _ := newConfigHandlerWithFile(t)
	rr := httptest.NewRecorder()
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate", ``))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestValidateConfigRecordsApplyHistory(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	body := `{"log":{"level":"info"},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"direct"}}`
	rr := httptest.NewRecorder()
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	events, err := history.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Source != "validate" || events[0].Status != "validated" {
		t.Fatalf("events = %+v", events)
	}
}

func TestValidateConfigFailureRecordsApplyHistory(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	rr := httptest.NewRecorder()
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate", `{"inbounds":[{"tag":"broken"}]}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	events, err := history.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Source != "validate" || events[0].Status != "validate_failed" {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Error == "" {
		t.Fatal("expected error text")
	}
}

func TestValidateConfigAcceptsSourceQuery(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	body := `{"log":{"level":"info"},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"direct"}}`
	rr := httptest.NewRecorder()
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate?source=validate_raw", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	events, err := history.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Source != "validate_raw" || events[0].Status != "validated" {
		t.Fatalf("events = %+v", events)
	}
}

func TestValidateConfigUnknownSourceFallsBack(t *testing.T) {
	handler, history, _ := setupApplyHistoryHandler(t)
	body := `{"log":{"level":"info"},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"direct"}}`
	rr := httptest.NewRecorder()
	handler.ValidateConfig(rr, jsonRequest(http.MethodPost, "/api/config/validate?source=evil;drop", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	events, err := history.List(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Source != "validate" {
		t.Fatalf("events = %+v", events)
	}
}

func TestNormalizeValidateSource(t *testing.T) {
	if got := normalizeValidateSource("validate_route"); got != "validate_route" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeValidateSource(" "); got != "validate" {
		t.Fatalf("blank = %q", got)
	}
	if got := normalizeValidateSource("nope"); got != "validate" {
		t.Fatalf("unknown = %q", got)
	}
}
