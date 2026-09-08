package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
)

// ConfigContentHash 绑定预览与原始文件，避免覆盖其他页面刚保存的配置。
func ConfigContentHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// BuildSetupPlan 只生成配置与差异所需数据，不写文件、不下载、不启停内核。
func BuildSetupPlan(body []byte, request SetupRequest, env SetupEnvironment) (*SetupPlan, error) {
	current, err := setupConfigObject(body)
	original := cloneSetupConfig(current)
	warnings := []string{}
	if err == nil && request.ResetInvalid && AnalyzeConfig(body).Summary.Errors > 0 {
		err = &SetupError{Code: "config_invalid", Message: "configuration cannot be loaded by sing-box"}
	}
	if err != nil {
		if !request.ResetInvalid {
			return nil, err
		}
		current = map[string]any{}
		request.Modules = nil
		warnings = append(warnings, "config_reset")
	}
	modules, err := setupModules(request)
	if err != nil {
		return nil, err
	}
	dataDir, err := filepath.Abs(env.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve setup data directory: %w", err)
	}
	env.DataDir = dataDir
	cfg := cloneSetupConfig(current)
	builder := setupPlanBuilder{request: request, env: env}
	for _, module := range modules {
		if err := builder.install(cfg, module); err != nil {
			return nil, err
		}
	}
	if _, err := encodeInitialConfig(cfg); err != nil {
		return nil, &SetupError{Code: "config_invalid", Message: err.Error()}
	}
	warnings = append(warnings, setupWarnings(cfg, request, env)...)
	return &SetupPlan{SourceHash: ConfigContentHash(body), CurrentConfig: original,
		Config: cfg, Modules: modules, Warnings: warnings, WillRestart: env.KernelRunning}, nil
}

func setupConfigObject(body []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, &SetupError{Code: "config_invalid", Message: "configuration JSON is invalid; explicitly choose rebuild to replace it"}
	}
	if value == nil {
		return map[string]any{}, nil
	}
	cfg, ok := value.(map[string]any)
	if !ok {
		return nil, &SetupError{Code: "config_invalid", Message: "configuration must be a JSON object"}
	}
	return cfg, nil
}

func setupModules(request SetupRequest) ([]string, error) {
	selected := make(map[string]bool)
	requested := request.Modules
	if len(requested) == 0 {
		requested = setupModuleOrder
	}
	if request.InboundMode != "" && request.InboundMode != "preserve" {
		requested = append(slices.Clone(requested), "inbounds")
	}
	for _, module := range requested {
		if _, ok := setupModuleDependencies[module]; !ok {
			return nil, &SetupError{Code: "invalid_setup_module", Message: "unknown configuration module: " + module}
		}
		addSetupDependencies(selected, module)
	}
	result := make([]string, 0, len(selected))
	for _, module := range setupModuleOrder {
		if selected[module] {
			result = append(result, module)
		}
	}
	return result, nil
}

func addSetupDependencies(selected map[string]bool, module string) {
	if selected[module] {
		return
	}
	selected[module] = true
	for _, dependency := range setupModuleDependencies[module] {
		addSetupDependencies(selected, dependency)
	}
}

func cloneSetupConfig(cfg map[string]any) map[string]any {
	cloned := make(map[string]any, len(cfg))
	for key, value := range cfg {
		cloned[key] = cloneSetupValue(value)
	}
	return cloned
}

func cloneSetupValue(value any) any {
	switch item := value.(type) {
	case map[string]any:
		return cloneSetupConfig(item)
	case []any:
		result := make([]any, len(item))
		for index, value := range item {
			result[index] = cloneSetupValue(value)
		}
		return result
	default:
		return value
	}
}

func setupWarnings(cfg map[string]any, request SetupRequest, env SetupEnvironment) []string {
	warnings := []string{}
	if env.Capabilities.Container {
		warnings = append(warnings, "container_port_mapping")
	}
	if !setupProxyReady(cfg) {
		warnings = append(warnings, "proxy_nodes_required")
	}
	ipv4Only := request.IPv6 == "off" || (!env.Capabilities.IPv6Available && request.IPv6 != "on")
	if request.InboundMode == "tun" && ipv4Only {
		warnings = append(warnings, "tun_ipv4_only")
	}
	return warnings
}
