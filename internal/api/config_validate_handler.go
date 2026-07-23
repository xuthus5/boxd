package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/xuthus5/boxd/internal/model"
)

// ValidateConfig dry-runs sing-box option unmarshalling without writing or restarting.
func (h *ConfigHandler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(body) == 0 {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "empty request body")
		return
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid JSON")
		return
	}
	if err := validateRuntimeConfig(body); err != nil {
		if errors.Is(err, ErrInvalidRuntimeConfig) {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, runtimeConfigErrorMessage(err))
			return
		}
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to validate config")
		return
	}
	writeJSONWithMeta(w, http.StatusOK, map[string]any{"valid": true}, map[string]any{
		"validated": true,
		"applied":   false,
	})
}
