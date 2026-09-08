package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

func TestSetupStatusAndRequestErrors(t *testing.T) {
	handler, _ := setupHandler(t, `{`)
	status := httptest.NewRecorder()
	handler.GetSetup(status, httptest.NewRequest(http.MethodGet, "/", nil))
	var data struct {
		Data core.SetupStatus `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if status.Code != http.StatusOK || data.Data.ConfigError == "" || data.Data.Listeners == nil {
		t.Fatalf("broken config must remain diagnosable: %d %s", status.Code, status.Body.String())
	}
	for _, body := range []string{`{`, `{"unknown":true}`, `{} {}`, `{"modules":["unknown"]}`} {
		response := httptest.NewRecorder()
		handler.PreviewSetup(response, jsonRequest(http.MethodPost, "/", body))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid preview request %q: %d %s", body, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.ApplySetup(response, jsonRequest(http.MethodPost, "/", `{}`))
	if response.Code != http.StatusBadRequest {
		t.Fatal("apply must require a preview hash")
	}
}

func TestSetupReadAndPreparationErrorsDoNotWrite(t *testing.T) {
	for _, action := range []string{"status", "preview", "apply"} {
		t.Run(action, func(t *testing.T) {
			handler, _ := setupHandler(t, `{}`)
			handler.configPath = t.TempDir()
			response := httptest.NewRecorder()
			request := jsonRequest(http.MethodPost, "/", `{"source_hash":"test"}`)
			switch action {
			case "status":
				handler.GetSetup(response, request)
			case "preview":
				handler.PreviewSetup(response, request)
			case "apply":
				handler.ApplySetup(response, request)
			}
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("read error: %d %s", response.Code, response.Body.String())
			}
		})
	}
	handler, path := setupHandler(t, `{}`)
	handler.ruleSetInstaller = core.NewLoyalsoldierRuleSetInstaller(path)
	body, err := json.Marshal(core.SetupRequest{SourceHash: core.ConfigContentHash([]byte(`{}`))})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ApplySetup(response, jsonRequest(http.MethodPost, "/", string(body)))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("backup error: %d %s", response.Code, response.Body.String())
	}
	if cfg := decodeConfigFile(t, path); len(cfg) != 0 {
		t.Fatal("failed preparation changed the configuration")
	}
}

func TestSetupCASInsideWriterPreventsPreparation(t *testing.T) {
	handler, _ := setupHandler(t, `{"log":{"level":"warn"}}`)
	prepared := false
	_, _, err := handler.applyPreparedConfig(configApplyRequest{Body: []byte(`{}`),
		ExpectedHash: core.ConfigContentHash([]byte(`{}`)), Prepare: func() error { prepared = true; return nil }})
	if !errors.Is(err, core.ErrSetupStale) || prepared {
		t.Fatalf("CAS must precede asset preparation: %v, prepared=%v", err, prepared)
	}
}

func TestSetupProfileErrorsExposeActionableCodes(t *testing.T) {
	response := httptest.NewRecorder()
	writeSetupError(response, &core.InboundProfileError{Code: core.InboundProfileIPv6, Message: "select IPv4 or enable system IPv6"})
	if response.Code != http.StatusBadRequest {
		t.Fatal("profile errors must be actionable client errors")
	}
	handler, _ := setupHandler(t, `{}`)
	handler.instance = &setupKernel{running: true}
	response = httptest.NewRecorder()
	handler.PreviewSetup(response, jsonRequest(http.MethodPost, "/", `{"modules":["unknown"]}`))
	if response.Code != http.StatusBadRequest {
		t.Fatal("unknown modules must not reach the writer")
	}
}

func TestSetupInvalidPlanCannotBeApplied(t *testing.T) {
	handler, path := setupHandler(t, `{}`)
	response := httptest.NewRecorder()
	handler.applySetupPlan(response, httptest.NewRequest(http.MethodPost, "/", nil), &core.SetupPlan{
		Config: map[string]any{"invalid": func() {}},
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatal("unencodable plans must fail")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "data")); !os.IsNotExist(err) {
		t.Fatal("invalid plan created assets")
	}
}
