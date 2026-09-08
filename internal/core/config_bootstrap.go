package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuthus5/boxd/internal/model"
)

// EnsureConfigFile 在配置目录内初始化完整默认配置；已有文件保持不变。
// 应用入口使用 EnsureDefaultConfig 传入独立的数据目录。
func EnsureConfigFile(path string) (bool, error) {
	return EnsureDefaultConfig(context.Background(), path, filepath.Dir(path))
}

// EnsureDefaultConfig 仅在首次启动时从离线规则快照生成全部必要模块。
// 返回 true 表示本次创建；取消或失败不会留下半成品配置。
func EnsureDefaultConfig(ctx context.Context, path, dataDir string) (bool, error) {
	if exists, err := configFileExists(path); exists || err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	absoluteDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return false, fmt.Errorf("resolve data directory: %w", err)
	}
	entries, err := NewLoyalsoldierRuleSetInstaller(absoluteDataDir).InstallBundled(ctx)
	if err != nil {
		return false, fmt.Errorf("initialize rule sets: %w", err)
	}
	cfg, err := defaultConfigTemplate(absoluteDataDir, entries, bootstrapTUNAvailable())
	if err != nil {
		return false, err
	}
	body, err := encodeInitialConfig(cfg)
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return writeInitialConfig(path, body)
}

func configFileExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat config file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		info, err = os.Stat(path)
		if err != nil {
			return false, fmt.Errorf("stat config symlink target: %w", err)
		}
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("config path is not a regular file: %s", path)
	}
	return true, nil
}

func defaultConfigTemplate(dataDir string, entries []map[string]any, enableTUN bool) (map[string]any, error) {
	ruleSets := make([]any, 0, len(entries))
	for _, entry := range entries {
		ruleSets = append(ruleSets, entry)
	}
	cfg := map[string]any{
		"log":      map[string]any{"level": "info", "timestamp": true},
		"inbounds": initialInbounds(enableTUN),
		"route":    map[string]any{"rule_set": ruleSets, "final": "proxy"},
	}
	outbounds, err := NewDefaultOutboundsInstaller().Install(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize outbounds: %w", err)
	}
	cfg["outbounds"] = outbounds.Outbounds
	if err := installInitialPolicy(cfg); err != nil {
		return nil, err
	}
	experimental, err := NewDefaultExperimentalInstaller().Install(cfg, dataDir)
	if err != nil {
		return nil, fmt.Errorf("initialize experimental options: %w", err)
	}
	cfg["experimental"] = experimental.Experimental
	ConfigureDefaultTUNRouting(cfg)
	return cfg, nil
}

func installInitialPolicy(cfg map[string]any) error {
	rules, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		return fmt.Errorf("initialize routing: %w", err)
	}
	dns, err := NewDefaultDNSInstaller().Install(cfg)
	if err != nil {
		return fmt.Errorf("initialize DNS: %w", err)
	}
	cfg["dns"] = dns.DNS
	route, _ := cfg["route"].(map[string]any)
	route["rules"] = rules.Rules
	route["default_domain_resolver"] = dns.DefaultDomainResolver
	return nil
}

func encodeInitialConfig(cfg map[string]any) ([]byte, error) {
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode default config: %w", err)
	}
	for _, issue := range AnalyzeConfig(body).Issues {
		if issue.Severity == model.ConfigDiagnosticSeverityError {
			return nil, fmt.Errorf("validate default config: %s: %s", issue.Path, issue.Code)
		}
	}
	return append(body, '\n'), nil
}
