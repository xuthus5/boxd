package api

import (
	"net/http"

	"github.com/xuthus5/boxd/internal/core"
)

// ConfigDiagnostics reports the persisted configuration without applying it.
func (h *ConfigHandler) ConfigDiagnostics(w http.ResponseWriter, _ *http.Request) {
	report := core.AnalyzeConfigFile(h.configPath)
	writeJSON(w, http.StatusOK, report)
}
