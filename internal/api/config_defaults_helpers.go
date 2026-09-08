package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/xuthus5/boxd/internal/core"
)

func (h *ConfigHandler) readDefaultConfig(w http.ResponseWriter) (map[string]any, bool) {
	body, err := os.ReadFile(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "config not found")
		return nil, false
	}
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid JSON in config")
		return nil, false
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, true
}

func (h *ConfigHandler) defaultDataDir() string {
	if installer, ok := h.ruleSetInstaller.(interface{ RuleSetDir() string }); ok {
		return filepath.Dir(installer.RuleSetDir())
	}
	return filepath.Dir(h.configPath)
}

func writeDefaultDNSInstallError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, core.ErrDNSProxyUnsafe) || errors.Is(err, core.ErrDNSProxyRequired) {
		status = http.StatusBadRequest
	}
	writeJSONError(w, status, err.Error())
}
