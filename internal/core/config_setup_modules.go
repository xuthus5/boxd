package core

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
)

type setupPlanBuilder struct {
	request SetupRequest
	env     SetupEnvironment
}

func (b setupPlanBuilder) install(cfg map[string]any, module string) error {
	switch module {
	case "outbounds":
		return installSetupOutbounds(cfg)
	case "rule_sets":
		installSetupRuleSets(cfg, b.env.DataDir)
	case "inbounds":
		options := InboundProfileOptions{Mode: b.request.InboundMode, IPv6: b.request.IPv6}
		result, err := BuildInboundProfile(cfg, options, b.env.Capabilities)
		if err != nil {
			return err
		}
		cfg["inbounds"] = result.Inbounds
		ConfigureDefaultTUNRouting(cfg)
	case "dns":
		return setupDNSInstallError(EnsureDefaultDNSForRouting(cfg))
	case "route":
		result, err := NewDefaultRouteInstaller().Install(cfg)
		if err != nil {
			return err
		}
		route := setupRoute(cfg)
		route["rules"] = result.Rules
	case "experimental":
		result, err := NewDefaultExperimentalInstaller().Install(cfg, b.env.DataDir)
		if err != nil {
			return err
		}
		cfg["experimental"] = result.Experimental
	}
	return nil
}

func setupDNSInstallError(err error) error {
	if errors.Is(err, ErrDNSProxyUnsafe) {
		return &SetupError{Code: "config_invalid", Message: err.Error() +
			"; remove direct fallback or missing members from the proxy outbound, then preview DNS setup again"}
	}
	if errors.Is(err, ErrDNSProxyRequired) {
		return &SetupError{Code: "config_invalid", Message: err.Error() +
			"; install proxy outbound defaults or select a valid proxy outbound before installing DNS"}
	}
	return err
}

func installSetupOutbounds(cfg map[string]any) error {
	result, err := NewDefaultOutboundsInstaller().Install(cfg)
	if err != nil {
		return err
	}
	cfg["outbounds"] = result.Outbounds
	ensureDefaultProxyFinal(cfg)
	if cfg["log"] == nil {
		cfg["log"] = map[string]any{"level": "info", "timestamp": true}
	}
	return nil
}

func setupRoute(cfg map[string]any) map[string]any {
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
		cfg["route"] = route
	}
	return route
}

func installSetupRuleSets(cfg map[string]any, dataDir string) {
	route := setupRoute(cfg)
	entries, _ := route["rule_set"].([]any)
	entries = slices.Clone(entries)
	tags := existingRuleSetTags(cfg)
	for _, source := range defaultRuleSetSources() {
		if !tags[source.Tag] {
			entries = append(entries, localRuleSetEntry(filepath.Join(dataDir, "rule-sets"), source))
		}
	}
	route["rule_set"] = entries
}

// InstallSetupAssets 只安装预览实际引用的内置本地规则；已有有效文件保持不变。
func InstallSetupAssets(ctx context.Context, plan *SetupPlan, dataDir string) error {
	if !slices.Contains(plan.Modules, "rule_sets") {
		return ctx.Err()
	}
	absoluteDir, err := filepath.Abs(dataDir)
	if err != nil {
		return err
	}
	installer := NewLoyalsoldierRuleSetInstaller(absoluteDir)
	selected := make([]RuleSetSource, 0)
	route, _ := plan.Config["route"].(map[string]any)
	entries, _ := route["rule_set"].([]any)
	for _, source := range installer.sources {
		if setupReferencesRuleSet(entries, installer.ruleSetDir, source) {
			selected = append(selected, source)
		}
	}
	if len(selected) == 0 {
		return ctx.Err()
	}
	installer.sources = selected
	_, err = installer.InstallBundled(ctx)
	return err
}

func setupReferencesRuleSet(entries []any, dir string, source RuleSetSource) bool {
	for _, item := range entries {
		entry, _ := item.(map[string]any)
		if entry["tag"] == source.Tag && entry["type"] == "local" && entry["path"] == filepath.Join(dir, source.FileName) {
			return true
		}
	}
	return false
}
