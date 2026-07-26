package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

// knownValidateSources are dry-run entry labels stored on the apply timeline.
var knownValidateSources = map[string]struct{}{
	"validate":              {},
	"validate_raw":          {},
	"validate_endpoints":    {},
	"validate_certificate":  {},
	"validate_services":     {},
	"validate_log":          {},
	"validate_ntp":          {},
	"validate_experimental": {},
	"validate_inbounds":     {},
	"validate_outbounds":    {},
	"validate_route":        {},
	"validate_dns":          {},
}

func normalizeValidateSource(raw string) string {
	source := strings.TrimSpace(raw)
	if source == "" {
		return "validate"
	}
	if _, ok := knownValidateSources[source]; ok {
		return source
	}
	return "validate"
}

// ValidateConfig dry-runs sing-box option unmarshalling without writing or restarting.
// Optional query ?source= labels the editor entry for the apply timeline.
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
	source := normalizeValidateSource(r.URL.Query().Get("source"))
	if err := validateRuntimeConfig(body); err != nil {
		if errors.Is(err, ErrInvalidRuntimeConfig) {
			msg := runtimeConfigErrorMessage(err)
			h.recordConfigApply(source, "validate_failed", body, errors.New(msg))
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, msg)
			return
		}
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to validate config")
		return
	}
	h.recordConfigApply(source, "validated", body, nil)
	writeJSONWithMeta(w, http.StatusOK, map[string]any{"valid": true}, map[string]any{
		"validated": true,
		"applied":   false,
		"source":    source,
	})
}
