package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func (h *ConfigHandler) setupEnvironment() core.SetupEnvironment {
	running := false
	if instance, ok := h.instance.(interface{ Status() model.ServiceStatus }); ok {
		running = instance.Status().Running
	}
	return core.SetupEnvironment{DataDir: h.defaultDataDir(),
		Capabilities: core.DetectNetworkCapabilities(), KernelRunning: running}
}

func (h *ConfigHandler) GetSetup(w http.ResponseWriter, r *http.Request) {
	body, err := core.ReadSetupSource(h.configPath)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to read config")
		return
	}
	writeJSON(w, http.StatusOK, core.InspectSetup(body, h.setupEnvironment()))
}

func (h *ConfigHandler) PreviewSetup(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeSetupRequest(w, r)
	if !ok {
		return
	}
	body, err := core.ReadSetupSource(h.configPath)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to read config")
		return
	}
	plan, err := core.BuildSetupPlan(body, request, h.setupEnvironment())
	if err != nil {
		writeSetupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *ConfigHandler) ApplySetup(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeSetupRequest(w, r)
	if !ok {
		return
	}
	if request.SourceHash == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, "preview_required", "preview the module changes before applying")
		return
	}
	source, err := core.ReadSetupSource(h.configPath)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to read config")
		return
	}
	if core.ConfigContentHash(source) != request.SourceHash {
		writeSetupError(w, core.ErrSetupStale)
		return
	}
	env := h.setupEnvironment()
	plan, err := core.BuildSetupPlan(source, request, env)
	if err != nil {
		writeSetupError(w, err)
		return
	}
	h.applySetupPlan(w, r, plan)
}

func (h *ConfigHandler) applySetupPlan(w http.ResponseWriter, r *http.Request, plan *core.SetupPlan) {
	body, err := json.MarshalIndent(plan.Config, "", "  ")
	if err != nil {
		writeSetupError(w, err)
		return
	}
	prepare := func() error {
		if err := core.BackupSetupSource(h.configPath, h.defaultDataDir(), plan.SourceHash); err != nil {
			return err
		}
		return core.InstallSetupAssets(r.Context(), plan, h.defaultDataDir())
	}
	status, apiErr, err := h.applyPreparedConfig(configApplyRequest{Body: body, Validate: true,
		Source: "setup_modules", ExpectedHash: plan.SourceHash, Prepare: prepare})
	if err != nil {
		writeSetupError(w, err)
		return
	}
	writeJSONStatus(w, http.StatusOK, status, map[string]any{
		"modules": plan.Modules, "config_hash": core.ConfigContentHash(body),
	}, apiErr, map[string]any{"rolled_back": status == model.StatusRolledBack})
}

func decodeSetupRequest(w http.ResponseWriter, r *http.Request) (core.SetupRequest, bool) {
	var request core.SetupRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid module setup request")
		return request, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "request must contain one JSON object")
		return request, false
	}
	return request, true
}

func writeSetupError(w http.ResponseWriter, err error) {
	var setupErr *core.SetupError
	var inboundErr *core.InboundProfileError
	switch {
	case errors.As(err, &setupErr):
		writeJSONErrorCode(w, http.StatusBadRequest, setupErr.Code, setupErr.Message)
	case errors.As(err, &inboundErr):
		writeJSONErrorCode(w, http.StatusBadRequest, inboundErr.Code, inboundErr.Message)
	default:
		writeApplyConfigError(w, err)
	}
}
