package api

import (
	"encoding/json"
	"net/http"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

type RuleSetHandler struct {
	updater  *core.RuleSetUpdater
	settings *core.SettingsManager
}

func NewRuleSetHandler(updater *core.RuleSetUpdater, settings *core.SettingsManager) *RuleSetHandler {
	return &RuleSetHandler{updater: updater, settings: settings}
}

func (h *RuleSetHandler) Status(w http.ResponseWriter, r *http.Request) {
	if h.updater == nil {
		writeJSONError(w, http.StatusNotImplemented, "rule-set updater is not configured")
		return
	}
	items, err := h.updater.Status(r.Context())
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *RuleSetHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.updater == nil {
		writeJSONError(w, http.StatusNotImplemented, "rule-set updater is not configured")
		return
	}
	var req core.RuleSetUpdateRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
			return
		}
	}
	result, err := h.updater.Update(r.Context(), req)
	if err != nil {
		writeJSONErrorCode(w, http.StatusBadGateway, model.ErrorBadGateway, err.Error())
		return
	}
	status := model.StatusOK
	if result.FailedCount > 0 && result.UpdatedCount > 0 {
		status = model.StatusPartial
	} else if result.FailedCount > 0 && result.UpdatedCount == 0 {
		status = model.StatusError
	}
	writeJSONStatus(w, http.StatusOK, status, result, nil, map[string]any{
		"updated_count": result.UpdatedCount,
		"failed_count":  result.FailedCount,
		"skipped_count": result.SkippedCount,
		"restarted":     result.Restarted,
	})
}

func (h *RuleSetHandler) GetAutoUpdate(w http.ResponseWriter, r *http.Request) {
	if h.settings == nil {
		writeJSONError(w, http.StatusNotImplemented, "settings are not configured")
		return
	}
	cfg, err := h.settings.RuleSetAutoUpdate()
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *RuleSetHandler) SetAutoUpdate(w http.ResponseWriter, r *http.Request) {
	if h.settings == nil {
		writeJSONError(w, http.StatusNotImplemented, "settings are not configured")
		return
	}
	var cfg model.RuleSetAutoUpdate
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}
	if err := h.settings.SetRuleSetAutoUpdate(cfg); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}
	// re-read for normalized defaults
	saved, err := h.settings.RuleSetAutoUpdate()
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}
