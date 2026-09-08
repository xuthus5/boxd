package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

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
	if errors.Is(err, core.ErrSetupStale) {
		writeJSONErrorCode(w, http.StatusConflict, "config_changed", err.Error())
		return
	}
	if errors.Is(err, ErrInvalidRuntimeConfig) {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorConfigInvalidRuntime, runtimeConfigErrorMessage(err))
		return
	}
	var reloadErr *configReloadError
	if errors.As(err, &reloadErr) {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorConfigRestartFailed, reloadErr.Error())
		return
	}
	writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to write config")
}

func (h *ConfigHandler) applyConfigBytesWithSource(
	body []byte,
	shouldValidate bool,
	source string,
) (string, *model.APIError, error) {
	return h.applyPreparedConfig(configApplyRequest{Body: body, Validate: shouldValidate, Source: source})
}

type configApplyRequest struct {
	Body         []byte
	Validate     bool
	Source       string
	ExpectedHash string
	Prepare      func() error
}

type configReloadError struct{ cause error }

func (e *configReloadError) Error() string { return e.cause.Error() }

func (e *configReloadError) Unwrap() error { return e.cause }

func (h *ConfigHandler) applyPreparedConfig(request configApplyRequest) (string, *model.APIError, error) {
	if request.Validate {
		if err := validateRuntimeConfig(request.Body); err != nil {
			return "", nil, err
		}
	}
	unlock := core.LockConfig(h.configPath)
	defer unlock()
	h.applyMu.Lock()
	defer h.applyMu.Unlock()
	previousBody, err := os.ReadFile(h.configPath)
	previousExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", nil, err
	}
	if request.ExpectedHash != "" && core.ConfigContentHash(previousBody) != request.ExpectedHash {
		return "", nil, core.ErrSetupStale
	}
	if request.Prepare != nil {
		if err := request.Prepare(); err != nil {
			return "", nil, err
		}
	}
	if err := atomicWriteFile(h.configPath, request.Body); err != nil {
		return "", nil, err
	}
	return h.finishConfigApply(request, previousBody, previousExists)
}

func (h *ConfigHandler) finishConfigApply(request configApplyRequest, previousBody []byte, previousExists bool) (string, *model.APIError, error) {
	if h.instance == nil {
		h.recordConfigApply(request.Source, model.StatusOK, request.Body, nil)
		return model.StatusOK, nil, nil
	}
	restartErr := core.ReloadConfig(h.instance)
	if restartErr == nil {
		h.recordConfigApply(request.Source, model.StatusOK, request.Body, nil)
		return model.StatusOK, nil, nil
	}
	slog.Error("auto-restart after config save failed", "err", restartErr)
	if err := rollbackConfigFile(h.configPath, previousBody, previousExists); err != nil {
		return "", nil, &configReloadError{cause: fmt.Errorf("%w; restore config file: %v", restartErr, err)}
	}
	if err := h.instance.Restart(); err != nil {
		return "", nil, &configReloadError{cause: fmt.Errorf("%w; restart restored config: %v", restartErr, err)}
	}
	h.recordConfigApply(request.Source, model.StatusRolledBack, request.Body, restartErr)
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
