package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// InstallDefaultOutbounds 安装默认出站配置。
func (c *Config) InstallDefaultOutbounds(ctx context.Context) (InstallResult, error) {
	if c.outboundInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default outbound installer is not configured")
	}
	slog.Info("installing default outbounds")
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.outboundInstaller.Install(cfg)
	if err != nil {
		slog.Error("outbound install failed", "err", err)
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
	installResult, err := c.applyInstalledConfig(ctx, cfg, "outbounds_defaults", result.Installed)
	if err != nil {
		return installResult, err
	}
	slog.Info("default outbounds installed", "status", installResult.Status, "rolled_back", installResult.RolledBack)
	return installResult, nil
}

// InstallDefaultRouteRules 安装默认路由规则。
func (c *Config) InstallDefaultRouteRules(ctx context.Context) (InstallResult, error) {
	if c.routeInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default route installer is not configured")
	}
	slog.Info("installing default route rules")
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	if err := core.EnsureDefaultDNSForRouting(cfg); err != nil {
		return InstallResult{}, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	result, err := c.routeInstaller.Install(cfg)
	if err != nil {
		slog.Error("route rule install failed", "err", err)
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
	slog.Info("default route rules installed", "status", installResult.Status, "rolled_back", installResult.RolledBack)
	return installResult, nil
}

// InstallDefaultDNS 安装默认 DNS 配置。
func (c *Config) InstallDefaultDNS(ctx context.Context) (InstallResult, error) {
	if c.dnsInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default dns installer is not configured")
	}
	slog.Info("installing default DNS")
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := core.PrepareDNSDefaults(cfg, c.dnsInstaller, c.outboundInstaller)
	if err != nil {
		slog.Error("DNS install failed", "err", err)
		if errors.Is(err, core.ErrDNSProxyUnsafe) || errors.Is(err, core.ErrDNSProxyRequired) {
			return InstallResult{}, Errorf(400, model.ErrorInvalidRequest, "%v", err)
		}
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
	}
	applyDNSDefaults(cfg, result)
	installResult, err := c.applyInstalledConfig(ctx, cfg, "dns_defaults", result.Installed)
	if err != nil {
		return installResult, err
	}
	slog.Info("default DNS installed", "status", installResult.Status, "rolled_back", installResult.RolledBack)
	return installResult, nil
}

// InstallDefaultInbounds 安装默认入站配置。
func (c *Config) InstallDefaultInbounds(ctx context.Context) (InstallResult, error) {
	if c.inboundInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default inbound installer is not configured")
	}
	slog.Info("installing default inbounds")
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.inboundInstaller.Install(cfg)
	if err != nil {
		slog.Error("inbound install failed", "err", err)
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
	}
	cfg["inbounds"] = result.Inbounds
	core.ConfigureDefaultTUNRouting(cfg)
	installResult, err := c.applyInstalledConfig(ctx, cfg, "inbounds_defaults", result.Installed)
	if err != nil {
		return installResult, err
	}
	slog.Info("default inbounds installed", "status", installResult.Status, "rolled_back", installResult.RolledBack)
	return installResult, nil
}

// InstallDefaultExperimental 安装默认 experimental 配置。
func (c *Config) InstallDefaultExperimental(ctx context.Context) (InstallResult, error) {
	if c.experimentalInstaller == nil {
		return InstallResult{}, Errorf(501, model.ErrorInternal, "default experimental installer is not configured")
	}
	slog.Info("installing default experimental")
	cfg, apiErr := c.readConfigMap(ctx)
	if apiErr != nil {
		return InstallResult{}, apiErr
	}
	result, err := c.experimentalInstaller.Install(cfg, c.dataDir)
	if err != nil {
		slog.Error("experimental install failed", "err", err)
		return InstallResult{}, Errorf(500, model.ErrorInternal, "%v", err)
	}
	cfg["experimental"] = result.Experimental
	installResult, err := c.applyInstalledConfig(ctx, cfg, "experimental_defaults", result.Installed)
	if err != nil {
		return installResult, err
	}
	slog.Info("default experimental installed", "status", installResult.Status, "rolled_back", installResult.RolledBack)
	return installResult, nil
}
