package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type ConfigHandler struct {
	configPath            string
	instance              restartableInstance
	ruleSetInstaller      core.RuleSetDefaultsInstaller
	outboundInstaller     core.OutboundDefaultsInstaller
	inboundInstaller      core.InboundDefaultsInstaller
	routeInstaller        core.RouteDefaultsInstaller
	dnsInstaller          core.DNSDefaultsInstaller
	experimentalInstaller core.ExperimentalDefaultsInstaller
	routeMetadata         *core.RouteRuleMetadataManager
	applyHistory          *core.ConfigApplyHistoryManager
	applyMu               sync.Mutex
}

type restartableInstance interface {
	Restart() error
}

func NewConfigHandler(configPath string, instance restartableInstance, ruleSetInstaller core.RuleSetDefaultsInstaller, outboundInstaller core.OutboundDefaultsInstaller, routeInstaller core.RouteDefaultsInstaller, dnsInstaller core.DNSDefaultsInstaller, routeMetadata ...*core.RouteRuleMetadataManager) *ConfigHandler {
	return NewConfigHandlerWithHistory(configPath, instance, ruleSetInstaller, outboundInstaller, routeInstaller, dnsInstaller, nil, routeMetadata...)
}

// NewConfigHandlerWithHistory wires optional config apply timeline storage.
func NewConfigHandlerWithHistory(configPath string, instance restartableInstance, ruleSetInstaller core.RuleSetDefaultsInstaller, outboundInstaller core.OutboundDefaultsInstaller, routeInstaller core.RouteDefaultsInstaller, dnsInstaller core.DNSDefaultsInstaller, applyHistory *core.ConfigApplyHistoryManager, routeMetadata ...*core.RouteRuleMetadataManager) *ConfigHandler {
	handler := &ConfigHandler{
		configPath:            configPath,
		instance:              instance,
		ruleSetInstaller:      ruleSetInstaller,
		outboundInstaller:     outboundInstaller,
		inboundInstaller:      core.NewDefaultInboundsInstaller(),
		routeInstaller:        routeInstaller,
		dnsInstaller:          dnsInstaller,
		experimentalInstaller: core.NewDefaultExperimentalInstaller(),
		applyHistory:          applyHistory,
	}
	if len(routeMetadata) > 0 {
		handler.routeMetadata = routeMetadata[0]
	}
	return handler
}

func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNotFound, "config not found")
		return
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "invalid JSON in config")
		return
	}

	writeJSON(w, http.StatusOK, parsed)
}

func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	status, apiErr, err := h.applyConfigBytesWithSource(body, true, "update")
	if err != nil {
		if errors.Is(err, ErrInvalidRuntimeConfig) {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, runtimeConfigErrorMessage(err))
			return
		}
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
		return
	}
	writeJSONStatus(w, http.StatusOK, status, nil, apiErr, map[string]any{
		"rolled_back": status == model.StatusRolledBack,
	})
}

func (h *ConfigHandler) GetRawConfig(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNotFound, "config not found")
		return
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "invalid JSON in config")
		return
	}

	writeJSON(w, http.StatusOK, parsed)
}

func (h *ConfigHandler) UpdateRawConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	status, apiErr, err := h.applyConfigBytesWithSource(body, true, "raw")
	if err != nil {
		if errors.Is(err, ErrInvalidRuntimeConfig) {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, runtimeConfigErrorMessage(err))
			return
		}
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
		return
	}
	writeJSONStatus(w, http.StatusOK, status, nil, apiErr, map[string]any{
		"rolled_back": status == model.StatusRolledBack,
	})
}

func (h *ConfigHandler) InstallDefaultRuleSets(w http.ResponseWriter, r *http.Request) {
	if h.ruleSetInstaller == nil {
		writeJSONError(w, http.StatusNotImplemented, "default rule-set installer is not configured")
		return
	}

	entries, err := h.ruleSetInstaller.Install(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
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

	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}

	existing, _ := route["rule_set"].([]any)
	merged := mergeRuleSets(existing, entries)
	if len(merged) > 0 {
		route["rule_set"] = merged
	} else {
		delete(route, "rule_set")
	}
	cfg["route"] = route

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode config")
		return
	}
	status, apiErr, err := h.applyConfigBytesWithSource(body, true, "rule_sets_defaults")
	if err != nil {
		writeApplyConfigError(w, err)
		return
	}

	writeJSONStatus(w, http.StatusOK, status, entries, apiErr, map[string]any{
		"installed_count": len(entries),
		"rolled_back":     status == model.StatusRolledBack,
	})
}
