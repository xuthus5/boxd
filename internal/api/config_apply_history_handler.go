package api

import (
	"log/slog"
	"net/http"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func (h *ConfigHandler) recordConfigApply(source, status string, body []byte, applyErr error) {
	if h == nil || h.applyHistory == nil {
		return
	}
	event := core.NewConfigApplyEvent(source, status, body, applyErr)
	if err := h.applyHistory.Append(event); err != nil {
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
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}
