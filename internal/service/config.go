package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// restartable 抽象内核重启能力，便于测试注入。
type restartable interface {
	Restart() error
}

// Config 封装 sing-box 配置的读取、应用、回滚与默认安装等用例逻辑。
type Config struct {
	path                  string
	instance              restartable
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

// newConfig 构造配置用例服务。
func newConfig(
	path string,
	instance restartable,
	installers ConfigInstaller,
) *Config {
	return &Config{
		path:                  path,
		instance:              instance,
		ruleSetInstaller:      installers.RuleSetInstaller,
		outboundInstaller:     installers.OutboundInstaller,
		inboundInstaller:      installers.InboundInstaller,
		routeInstaller:        installers.RouteInstaller,
		dnsInstaller:          installers.DNSInstaller,
		experimentalInstaller: installers.ExperimentalInstaller,
		applyHistory:          installers.ApplyHistory,
		routeMetadata:         installers.RouteMetadata,
	}
}

// ConfigInstaller 聚合各默认安装器依赖。
type ConfigInstaller struct {
	RuleSetInstaller      core.RuleSetDefaultsInstaller
	OutboundInstaller     core.OutboundDefaultsInstaller
	InboundInstaller      core.InboundDefaultsInstaller
	RouteInstaller        core.RouteDefaultsInstaller
	DNSInstaller          core.DNSDefaultsInstaller
	ExperimentalInstaller core.ExperimentalDefaultsInstaller
	ApplyHistory          *core.ConfigApplyHistoryManager
	RouteMetadata         *core.RouteRuleMetadataManager
}

// readConfigObject 读取配置文件并解析为任意 JSON 值。
func (c *Config) readConfigObject(ctx context.Context) (any, *DomainError) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, Errorf(404, model.ErrorNotFound, "config not found")
		}
		return nil, Errorf(500, model.ErrorInternal, "failed to read config")
	}
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, Errorf(500, model.ErrorInternal, "invalid JSON in config")
	}
	return parsed, nil
}

// GetConfig 返回完整配置 JSON。
func (c *Config) GetConfig(ctx context.Context) (any, error) {
	parsed, apiErr := c.readConfigObject(ctx)
	if apiErr != nil {
		return nil, apiErr
	}
	return parsed, nil
}

// ApplyResult 描述一次配置写入/重启的结果。
type ApplyResult struct {
	Status     string          `json:"status"`
	APIError   *model.APIError `json:"api_error,omitempty"`
	RolledBack bool            `json:"rolled_back"`
}

// writeConfigFile 原子写入配置文件并重启内核，失败时回滚。
func (c *Config) writeConfigFile(ctx context.Context, body []byte, source string) (ApplyResult, error) {
	if err := ValidateRuntimeConfig(ctx, body); err != nil {
		return ApplyResult{}, err
	}
	c.applyMu.Lock()
	defer c.applyMu.Unlock()

	previousBody, err := os.ReadFile(c.path)
	previousExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return ApplyResult{}, Errorf(500, model.ErrorInternal, "failed to read config")
	}
	if err := atomicWriteFile(c.path, body); err != nil {
		return ApplyResult{}, Errorf(500, model.ErrorInternal, "failed to write config")
	}
	if c.instance == nil {
		c.recordApply(source, model.StatusOK, body, nil)
		return ApplyResult{Status: model.StatusOK}, nil
	}
	restartErr := c.instance.Restart()
	if restartErr == nil {
		c.recordApply(source, model.StatusOK, body, nil)
		return ApplyResult{Status: model.StatusOK}, nil
	}
	if err := rollbackConfigFile(c.path, previousBody, previousExists); err != nil {
		return ApplyResult{}, Errorf(500, model.ErrorInternal, "failed to write config")
	}
	if err := c.instance.Restart(); err != nil {
		return ApplyResult{}, Errorf(500, model.ErrorInternal, "failed to write config")
	}
	c.recordApply(source, model.StatusRolledBack, body, restartErr)
	return ApplyResult{
		Status: model.StatusRolledBack,
		APIError: &model.APIError{
			Code:    model.ErrorConfigRestartFailed,
			Message: restartFailureMessage(restartErr),
		},
		RolledBack: true,
	}, nil
}

// recordApply 记录一次配置应用事件。
func (c *Config) recordApply(source, status string, body []byte, applyErr error) {
	if c == nil || c.applyHistory == nil {
		return
	}
	event := core.NewConfigApplyEvent(source, status, body, applyErr)
	if err := c.applyHistory.AppendSnapshot(event, body); err != nil {
		// 历史记录失败不影响配置写入结果。
		_ = err
	}
}

// ApplyConfig 校验并写入配置，失败时回滚。
func (c *Config) ApplyConfig(ctx context.Context, body []byte, source string) (ApplyResult, error) {
	return c.writeConfigFile(ctx, body, source)
}

// ValidateConfig dry-run 校验配置但不写入。
func (c *Config) ValidateConfig(ctx context.Context, body []byte, source string) error {
	if len(body) == 0 {
		return Errorf(400, model.ErrorInvalidRequest, "empty request body")
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Errorf(400, model.ErrorInvalidRequest, "invalid JSON")
	}
	if err := ValidateRuntimeConfig(ctx, body); err != nil {
		var invalid *ErrInvalidRuntime
		if errors.As(err, &invalid) {
			c.recordApply(source, "validate_failed", body, errors.New(invalid.Message()))
			return invalid
		}
		return Errorf(500, model.ErrorInternal, "failed to validate config")
	}
	c.recordApply(source, "validated", body, nil)
	return nil
}

// InstallResult 描述默认安装操作的结果。
// 与 ApplyResult 保持相同的 JSON 字段命名，便于桌面 bridge 与前端识别回滚语义。
type InstallResult struct {
	Status         string          `json:"status"`
	Installed      any             `json:"installed"`
	APIError       *model.APIError `json:"api_error,omitempty"`
	InstalledCount int             `json:"installed_count"`
	RolledBack     bool            `json:"rolled_back"`
}

func (c *Config) applyInstalledConfig(ctx context.Context, cfg map[string]any, source string, installed any) (InstallResult, error) {
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return InstallResult{}, Errorf(500, model.ErrorInternal, "failed to encode config")
	}
	result, err := c.writeConfigFile(ctx, body, source)
	if err != nil {
		var invalid *ErrInvalidRuntime
		if errors.As(err, &invalid) {
			return InstallResult{}, invalid
		}
		return InstallResult{}, Errorf(500, model.ErrorInternal, "failed to write config")
	}
	return InstallResult{
		Status:         result.Status,
		Installed:      installed,
		APIError:       result.APIError,
		InstalledCount: installedCount(installed),
		RolledBack:     result.RolledBack,
	}, nil
}

func installedCount(installed any) int {
	switch v := installed.(type) {
	case []map[string]any:
		return len(v)
	case []model.RuleSetUpdateResult:
		return len(v)
	default:
		return 0
	}
}

// InstallDefaultRuleSets 安装默认规则集并合并到现有配置。
func (c *Config) InstallDefaultRuleSets(ctx context.Context) (InstallResult, error) {
	if c.ruleSetInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default rule-set installer is not configured")
	}
	entries, err := c.ruleSetInstaller.Install(ctx)
	if err != nil {
		return InstallResult{}, Errorf(502, model.ErrorBadGateway, "%v", err)
	}
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
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
	return c.applyInstalledConfig(ctx, cfg, "rule_sets_defaults", entries)
}

func (c *Config) readConfigMap(ctx context.Context) (map[string]any, *DomainError) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, Errorf(500, model.ErrorInternal, "failed to read config")
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, Errorf(500, model.ErrorInternal, "invalid JSON in config")
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

// InstallDefaultOutbounds 安装默认出站配置。
func (c *Config) InstallDefaultOutbounds(ctx context.Context) (InstallResult, error) {
	if c.outboundInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default outbound installer is not configured")
	}
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.outboundInstaller.Install(cfg)
	if err != nil {
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
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
	return c.applyInstalledConfig(ctx, cfg, "outbounds_defaults", result.Installed)
}

// InstallDefaultRouteRules 安装默认路由规则。
func (c *Config) InstallDefaultRouteRules(ctx context.Context) (InstallResult, error) {
	if c.routeInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default route installer is not configured")
	}
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.routeInstaller.Install(cfg)
	if err != nil {
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
	}
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}
	route["rules"] = result.Rules
	cfg["route"] = route
	installResult, err := c.applyInstalledConfig(ctx, cfg, "route_defaults", result.Installed)
	if err != nil {
		return installResult, err
	}
	if installResult.Status != model.StatusRolledBack && c.routeMetadata != nil {
		if err := c.routeMetadata.ApplyDefaultNames(result.Rules); err != nil {
			return InstallResult{}, Errorf(500, model.ErrorInternal, "failed to save default route rule metadata")
		}
	}
	return installResult, nil
}

// InstallDefaultDNS 安装默认 DNS 配置。
func (c *Config) InstallDefaultDNS(ctx context.Context) (InstallResult, error) {
	if c.dnsInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default dns installer is not configured")
	}
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.dnsInstaller.Install(cfg)
	if err != nil {
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
	}
	applyDNSDefaults(cfg, result)
	return c.applyInstalledConfig(ctx, cfg, "dns_defaults", result.Installed)
}

// InstallDefaultInbounds 安装默认入站配置。
func (c *Config) InstallDefaultInbounds(ctx context.Context) (InstallResult, error) {
	if c.inboundInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default inbound installer is not configured")
	}
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.inboundInstaller.Install(cfg)
	if err != nil {
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
	}
	cfg["inbounds"] = result.Inbounds
	return c.applyInstalledConfig(ctx, cfg, "inbounds_defaults", result.Installed)
}

// InstallDefaultExperimental 安装默认 experimental 配置。
func (c *Config) InstallDefaultExperimental(ctx context.Context) (InstallResult, error) {
	if c.experimentalInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default experimental installer is not configured")
	}
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.experimentalInstaller.Install(cfg)
	if err != nil {
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
	}
	cfg["experimental"] = result.Experimental
	return c.applyInstalledConfig(ctx, cfg, "experimental_defaults", result.Installed)
}
