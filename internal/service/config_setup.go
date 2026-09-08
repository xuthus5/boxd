package service

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// SetupApplyResult 保持与 HTTP 向导相同的模块数据及既有回滚语义。
type SetupApplyResult struct {
	ApplyResult
	Modules    []string `json:"modules"`
	ConfigHash string   `json:"config_hash"`
}

func (c *Config) setupEnvironment() core.SetupEnvironment {
	dataDir := c.dataDir
	if dataDir == "" {
		dataDir = filepath.Dir(c.path)
	}
	running := false
	if instance, ok := c.instance.(interface{ Status() model.ServiceStatus }); ok {
		running = instance.Status().Running
	}
	return core.SetupEnvironment{DataDir: dataDir, Capabilities: core.DetectNetworkCapabilities(), KernelRunning: running}
}

// GetSetup 检查模块及当前运行环境，不写配置。
func (c *Config) GetSetup(ctx context.Context) (core.SetupStatus, error) {
	body, err := c.readSetupSource(ctx)
	if err != nil {
		return core.SetupStatus{}, err
	}
	return core.InspectSetup(body, c.setupEnvironment()), nil
}

// PreviewSetup 只生成模块预览，不安装规则或修改运行状态。
func (c *Config) PreviewSetup(ctx context.Context, request core.SetupRequest) (*core.SetupPlan, error) {
	body, err := c.readSetupSource(ctx)
	if err != nil {
		return nil, err
	}
	plan, err := core.BuildSetupPlan(body, request, c.setupEnvironment())
	return plan, setupDomainError(err)
}

// ApplySetup 校验预览版本后备份原文件、安装依赖并提交配置。
func (c *Config) ApplySetup(ctx context.Context, request core.SetupRequest) (SetupApplyResult, error) {
	if request.SourceHash == "" {
		return SetupApplyResult{}, Errorf(400, "preview_required", "preview the module changes before applying")
	}
	source, err := c.readSetupSource(ctx)
	if err != nil {
		return SetupApplyResult{}, err
	}
	if core.ConfigContentHash(source) != request.SourceHash {
		return SetupApplyResult{}, setupDomainError(core.ErrSetupStale)
	}
	env := c.setupEnvironment()
	plan, err := core.BuildSetupPlan(source, request, env)
	if err != nil {
		return SetupApplyResult{}, setupDomainError(err)
	}
	return c.applySetupPlan(ctx, plan, env.DataDir)
}

func (c *Config) applySetupPlan(ctx context.Context, plan *core.SetupPlan, dataDir string) (SetupApplyResult, error) {
	body, err := json.MarshalIndent(plan.Config, "", "  ")
	if err != nil {
		return SetupApplyResult{}, Errorf(500, model.ErrorInternal, "failed to encode config")
	}
	prepare := func() error {
		if err := core.BackupSetupSource(c.path, dataDir, plan.SourceHash); err != nil {
			return err
		}
		return core.InstallSetupAssets(ctx, plan, dataDir)
	}
	result, err := c.writePreparedConfig(ctx, body, configWriteOptions{
		Source: "setup_modules", ExpectedHash: plan.SourceHash, Prepare: prepare,
	})
	return SetupApplyResult{ApplyResult: result, Modules: plan.Modules, ConfigHash: core.ConfigContentHash(body)}, setupDomainError(err)
}

func (c *Config) readSetupSource(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	body, err := core.ReadSetupSource(c.path)
	if err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to read config")
	}
	return body, nil
}

func setupDomainError(err error) error {
	if err == nil {
		return nil
	}
	var setupErr *core.SetupError
	var inboundErr *core.InboundProfileError
	var domainErr *DomainError
	var invalidErr *ErrInvalidRuntime
	switch {
	case errors.Is(err, core.ErrSetupStale):
		return Errorf(409, "config_changed", "%v", err)
	case errors.As(err, &setupErr):
		return Errorf(400, setupErr.Code, "%s", setupErr.Message)
	case errors.As(err, &inboundErr):
		return Errorf(400, inboundErr.Code, "%s", inboundErr.Message)
	case errors.As(err, &domainErr):
		return err
	case errors.As(err, &invalidErr):
		return Errorf(400, model.ErrorConfigInvalidRuntime, "%s", invalidErr.Message())
	default:
		return Errorf(500, model.ErrorInternal, "failed to prepare module configuration: %v", err)
	}
}
