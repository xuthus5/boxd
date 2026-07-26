package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

var ErrInvalidRuntimeConfig = errors.New("invalid sing-box config")

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
	h.applyMu.Lock()
	defer h.applyMu.Unlock()
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
