package api

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func (h *ConfigHandler) recordConfigApply(source, status string, body []byte, applyErr error) {
	if h == nil || h.applyHistory == nil {
		return
	}
	event := core.NewConfigApplyEvent(source, status, body, applyErr)
	if err := h.applyHistory.AppendSnapshot(event, body); err != nil {
		slog.Error("record config apply history failed", "err", err, "source", source)
	}
}

// ListConfigApplyHistory returns recent config apply/reload timeline events.
func (h *ConfigHandler) ListConfigApplyHistory(w http.ResponseWriter, _ *http.Request) {
	if h.applyHistory == nil {
		writeJSON(w, http.StatusOK, map[string]any{"events": []model.ConfigApplyEvent{}})
		return
	}
	events, err := h.applyHistory.List(0)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to load config apply history")
		return
	}
	markCurrentConfigApplyEvent(events, h.configPath)
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func markCurrentConfigApplyEvent(events []model.ConfigApplyEvent, configPath string) {
	body, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	hash := core.ConfigBodyHash(body)
	for index := range events {
		events[index].Current = events[index].Status == model.ConfigApplyStatusApplied && events[index].Hash == hash
	}
}

func (h *ConfigHandler) RestoreConfig(w http.ResponseWriter, r *http.Request) {
	if h.applyHistory == nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNotFound, "config snapshot not found")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	body, err := h.applyHistory.GetSnapshot(id)
	if err != nil {
		if errors.Is(err, core.ErrConfigSnapshotNotFound) {
			writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNotFound, "config snapshot not found")
			return
		}
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to load config snapshot")
		return
	}
	if configSnapshotIsCurrent(h.configPath, body) {
		writeJSON(w, http.StatusOK, map[string]any{
			"restored":        false,
			"source_id":       id,
			"already_current": true,
		})
		return
	}
	status, apiErr, err := h.applyConfigBytesWithSource(body, true, "restore")
	if err != nil {
		writeApplyConfigError(w, err)
		return
	}
	writeJSONStatus(w, http.StatusOK, status, map[string]any{
		"restored":  status == model.StatusOK,
		"source_id": id,
	}, apiErr, map[string]any{
		"restored_from": id,
		"rolled_back":   status == model.StatusRolledBack,
	})
}

func configSnapshotIsCurrent(configPath string, snapshot []byte) bool {
	current, err := os.ReadFile(configPath)
	return err == nil && bytes.Equal(current, snapshot)
}
