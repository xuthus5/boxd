package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/xuthus5/boxui/internal/model"
)

func (h *ConfigHandler) InstallDefaultOutbounds(w http.ResponseWriter, r *http.Request) {
	if h.outboundInstaller == nil {
		writeJSONError(w, http.StatusNotImplemented, "default outbound installer is not configured")
		return
	}

	data, err := os.ReadFile(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "config not found")
		return
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid JSON in config")
		return
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	result, err := h.outboundInstaller.Install(cfg)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg["outbounds"] = result.Outbounds

	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}
	if _, ok := route["final"]; !ok || route["final"] == "" {
		route["final"] = result.Final
	}
	cfg["route"] = route

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode config")
		return
	}
	status, apiErr, err := h.applyConfigBytes(body, false)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
		return
	}

	writeJSONStatus(w, http.StatusOK, status, result.Installed, apiErr, map[string]any{
		"installed_count": len(result.Installed),
		"rolled_back":     status == model.StatusRolledBack,
	})
}

func (h *ConfigHandler) InstallDefaultRouteRules(w http.ResponseWriter, r *http.Request) {
	if h.routeInstaller == nil {
		writeJSONError(w, http.StatusNotImplemented, "default route installer is not configured")
		return
	}

	data, err := os.ReadFile(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "config not found")
		return
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid JSON in config")
		return
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	result, err := h.routeInstaller.Install(cfg)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}
	route["rules"] = result.Rules
	cfg["route"] = route

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode config")
		return
	}
	status, apiErr, err := h.applyConfigBytes(body, false)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
		return
	}

	writeJSONStatus(w, http.StatusOK, status, result.Installed, apiErr, map[string]any{
		"installed_count": len(result.Installed),
		"rolled_back":     status == model.StatusRolledBack,
	})
}

func (h *ConfigHandler) InstallDefaultDNS(w http.ResponseWriter, r *http.Request) {
	if h.dnsInstaller == nil {
		writeJSONError(w, http.StatusNotImplemented, "default dns installer is not configured")
		return
	}

	data, err := os.ReadFile(h.configPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "config not found")
		return
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid JSON in config")
		return
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	result, err := h.dnsInstaller.Install(cfg)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg["dns"] = result.DNS

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode config")
		return
	}
	status, apiErr, err := h.applyConfigBytes(body, false)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
		return
	}

	writeJSONStatus(w, http.StatusOK, status, result.Installed, apiErr, map[string]any{
		"installed_count": len(result.Installed),
		"rolled_back":     status == model.StatusRolledBack,
	})
}

func mergeRuleSets(existing []any, installed []map[string]any) []any {
	indexByTag := make(map[string]int, len(existing))
	merged := make([]any, 0, len(existing)+len(installed))
	for _, item := range existing {
		merged = append(merged, item)
		if m, ok := item.(map[string]any); ok {
			if tag, _ := m["tag"].(string); tag != "" {
				indexByTag[tag] = len(merged) - 1
			}
		}
	}
	for _, item := range installed {
		tag, _ := item["tag"].(string)
		if idx, ok := indexByTag[tag]; ok {
			merged[idx] = item
			continue
		}
		merged = append(merged, item)
	}
	return merged
}
