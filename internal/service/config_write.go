package service

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type configWriteOptions struct {
	Source       string
	ExpectedHash string
	Prepare      func() error
}

type previousConfig struct {
	body   []byte
	exists bool
}

func (c *Config) writePreparedConfig(ctx context.Context, body []byte, options configWriteOptions) (ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return ApplyResult{}, err
	}
	if err := ValidateRuntimeConfig(ctx, body); err != nil {
		return ApplyResult{}, err
	}
	unlock := core.LockConfig(c.path)
	defer unlock()
	c.applyMu.Lock()
	defer c.applyMu.Unlock()
	previous, err := c.prepareConfigWrite(ctx, options)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := atomicWriteFile(c.path, body); err != nil {
		return ApplyResult{}, Errorf(500, model.ErrorInternal, "failed to write config")
	}
	return c.finishConfigWrite(body, previous, options.Source)
}

func (c *Config) prepareConfigWrite(ctx context.Context, options configWriteOptions) (previousConfig, error) {
	if err := ctx.Err(); err != nil {
		return previousConfig{}, err
	}
	body, err := os.ReadFile(c.path)
	previous := previousConfig{body: body, exists: err == nil}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return previous, Errorf(500, model.ErrorInternal, "failed to read config")
	}
	if options.ExpectedHash != "" && core.ConfigContentHash(body) != options.ExpectedHash {
		return previous, core.ErrSetupStale
	}
	if options.Prepare != nil {
		if err := options.Prepare(); err != nil {
			return previous, err
		}
	}
	return previous, ctx.Err()
}

func (c *Config) finishConfigWrite(body []byte, previous previousConfig, source string) (ApplyResult, error) {
	if c.instance == nil {
		c.recordApply(source, model.StatusOK, body, nil)
		return ApplyResult{Status: model.StatusOK}, nil
	}
	reloadErr := core.ReloadConfig(c.instance)
	if reloadErr == nil {
		c.recordApply(source, model.StatusOK, body, nil)
		return ApplyResult{Status: model.StatusOK}, nil
	}
	slog.Error("kernel reload failed; restoring previous config", "source", source, "err", reloadErr)
	if err := rollbackConfigFile(c.path, previous.body, previous.exists); err != nil {
		return ApplyResult{}, Errorf(500, model.ErrorConfigRestartFailed, "%v; restore config file: %v", reloadErr, err)
	}
	if err := c.instance.Restart(); err != nil {
		return ApplyResult{}, Errorf(500, model.ErrorConfigRestartFailed, "%v; restart restored config: %v", reloadErr, err)
	}
	c.recordApply(source, model.StatusRolledBack, body, reloadErr)
	return ApplyResult{Status: model.StatusRolledBack, RolledBack: true,
		APIError: &model.APIError{Code: model.ErrorConfigRestartFailed, Message: restartFailureMessage(reloadErr)}}, nil
}
