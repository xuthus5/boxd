package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

var ErrInvalidRuntimeConfig = errors.New("invalid sing-box config")

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

func validateRuntimeConfig(body []byte) error {
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	defer cancel()

	var cfg option.Options
	if err := cfg.UnmarshalJSONContext(ctx, body); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRuntimeConfig, err)
	}
	return nil
}

func runtimeConfigErrorMessage(err error) string {
	msg := strings.TrimSpace(err.Error())
	prefix := ErrInvalidRuntimeConfig.Error() + ": "
	if detail, ok := strings.CutPrefix(msg, prefix); ok && detail != "" {
		msg = strings.TrimSpace(detail)
	}
	// Collapse multi-line decoder noise to a single readable line.
	if lines := strings.Split(msg, "\n"); len(lines) > 1 {
		first := strings.TrimSpace(lines[0])
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, ".") || strings.Contains(line, "[") {
				msg = first + ": " + line
				break
			}
		}
		if !strings.Contains(msg, ":") {
			msg = first
		}
	}
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "invalid sing-box config"
	}
	return msg
}

func restartFailureMessage(err error) string {
	detail := strings.TrimSpace(err.Error())
	if detail == "" {
		return "restart failed after config save"
	}
	return "restart failed after config save: " + detail
}

func atomicWriteFile(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err := tempFile.Chmod(0600); err != nil {
		return err
	}
	if _, err := tempFile.Write(body); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

func writeApplyConfigError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrInvalidRuntimeConfig) {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, runtimeConfigErrorMessage(err))
		return
	}
	writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
}

func (h *ConfigHandler) applyConfigBytesWithSource(body []byte, shouldValidate bool, source string) (string, *model.APIError, error) {
	if shouldValidate {
		if err := validateRuntimeConfig(body); err != nil {
			return "", nil, err
		}
	}
	previousBody, err := os.ReadFile(h.configPath)
	previousExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", nil, err
	}
	if err := atomicWriteFile(h.configPath, body); err != nil {
		return "", nil, err
	}
	if h.instance == nil {
		h.recordConfigApply(source, model.StatusOK, body, nil)
		return model.StatusOK, nil, nil
	}
	restartErr := h.instance.Restart()
	if restartErr == nil {
		h.recordConfigApply(source, model.StatusOK, body, nil)
		return model.StatusOK, nil, nil
	}
	slog.Error("auto-restart after config save failed", "err", restartErr)
	if err := rollbackConfigFile(h.configPath, previousBody, previousExists); err != nil {
		return "", nil, err
	}
	if err := h.instance.Restart(); err != nil {
		return "", nil, err
	}
	h.recordConfigApply(source, model.StatusRolledBack, body, restartErr)
	return model.StatusRolledBack, &model.APIError{
		Code:    model.ErrorConfigRestartFailed,
		Message: restartFailureMessage(restartErr),
	}, nil
}

func rollbackConfigFile(path string, previous []byte, previousExists bool) error {
	if previousExists {
		return atomicWriteFile(path, previous)
	}
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
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
