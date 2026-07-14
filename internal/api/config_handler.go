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

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

var ErrInvalidRuntimeConfig = errors.New("invalid sing-box config")

type ConfigHandler struct {
	configPath        string
	instance          restartableInstance
	ruleSetInstaller  core.RuleSetDefaultsInstaller
	outboundInstaller core.OutboundDefaultsInstaller
	routeInstaller    core.RouteDefaultsInstaller
	dnsInstaller      core.DNSDefaultsInstaller
}

type restartableInstance interface {
	Restart() error
}

func NewConfigHandler(configPath string, instance restartableInstance, ruleSetInstaller core.RuleSetDefaultsInstaller, outboundInstaller core.OutboundDefaultsInstaller, routeInstaller core.RouteDefaultsInstaller, dnsInstaller core.DNSDefaultsInstaller) *ConfigHandler {
	return &ConfigHandler{
		configPath:        configPath,
		instance:          instance,
		ruleSetInstaller:  ruleSetInstaller,
		outboundInstaller: outboundInstaller,
		routeInstaller:    routeInstaller,
		dnsInstaller:      dnsInstaller,
	}
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

func (h *ConfigHandler) applyConfigBytes(body []byte, shouldValidate bool) (string, *model.APIError, error) {
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
		return model.StatusOK, nil, nil
	}

	if err := h.instance.Restart(); err == nil {
		return model.StatusOK, nil, nil
	} else {
		slog.Error("auto-restart after config save failed", "err", err)

		var rollbackErr error
		if previousExists {
			rollbackErr = atomicWriteFile(h.configPath, previousBody)
		} else {
			rollbackErr = os.Remove(h.configPath)
			if errors.Is(rollbackErr, os.ErrNotExist) {
				rollbackErr = nil
			}
		}
		if rollbackErr != nil {
			return "", nil, rollbackErr
		}
		if restartRollbackErr := h.instance.Restart(); restartRollbackErr != nil {
			return "", nil, restartRollbackErr
		}

		return model.StatusRolledBack, &model.APIError{
			Code:    model.ErrorConfigRestartFailed,
			Message: "restart failed after config save",
		}, nil
	}
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

	status, apiErr, err := h.applyConfigBytes(body, true)
	if err != nil {
		if errors.Is(err, ErrInvalidRuntimeConfig) {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, "invalid sing-box config")
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

	status, apiErr, err := h.applyConfigBytes(body, true)
	if err != nil {
		if errors.Is(err, ErrInvalidRuntimeConfig) {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, "invalid sing-box config")
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
	status, apiErr, err := h.applyConfigBytes(body, false)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
		return
	}

	writeJSONStatus(w, http.StatusOK, status, entries, apiErr, map[string]any{
		"installed_count": len(entries),
		"rolled_back":     status == model.StatusRolledBack,
	})
}
