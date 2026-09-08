package core

import (
	"errors"
	"path/filepath"
)

// ExperimentalDefaultsInstaller 安装常用 experimental 段默认值。
type ExperimentalDefaultsInstaller interface {
	Install(cfg map[string]any, dataDir string) (*ExperimentalDefaultsResult, error)
}

// ExperimentalDefaultsResult 返回合并后的 experimental 与本次写入的子集。
type ExperimentalDefaultsResult struct {
	Experimental map[string]any
	Installed    map[string]any
}

// DefaultExperimentalInstaller 启用内部 Clash 模式控制与选择状态缓存。
type DefaultExperimentalInstaller struct{}

// NewDefaultExperimentalInstaller 创建默认 experimental 安装器。
func NewDefaultExperimentalInstaller() *DefaultExperimentalInstaller {
	return &DefaultExperimentalInstaller{}
}

// Install 补齐内部 Clash Rule 模式与缓存路径，不额外开放监听端口。
// 明确设置的控制器、缓存路径和禁用状态保持不变。
func (i *DefaultExperimentalInstaller) Install(cfg map[string]any, dataDir string) (*ExperimentalDefaultsResult, error) {
	experimental := copyMap(asMap(cfg["experimental"]))
	installed := map[string]any{}

	clashAPI := copyMap(asMap(experimental["clash_api"]))
	clashInstalled := map[string]any{}
	ensureString(clashAPI, clashInstalled, "default_mode", "rule")
	experimental["clash_api"] = clashAPI
	if len(clashInstalled) > 0 {
		installed["clash_api"] = clashInstalled
	}

	cacheFile := copyMap(asMap(experimental["cache_file"]))
	cacheInstalled := map[string]any{}
	if _, exists := cacheFile["enabled"]; !exists && dataDir != "" {
		cacheFile["enabled"] = true
		cacheInstalled["enabled"] = true
	}
	if enabled, _ := cacheFile["enabled"].(bool); enabled {
		if path, _ := cacheFile["path"].(string); path == "" {
			if dataDir == "" {
				return nil, errors.New("data directory is required for the default cache path")
			}
			cacheFile["path"] = filepath.Join(dataDir, "cache.db")
			cacheInstalled["path"] = cacheFile["path"]
		}
	}
	if len(cacheFile) > 0 {
		experimental["cache_file"] = cacheFile
	} else {
		delete(experimental, "cache_file")
	}
	if len(cacheInstalled) > 0 {
		installed["cache_file"] = cacheInstalled
	}

	return &ExperimentalDefaultsResult{
		Experimental: experimental,
		Installed:    installed,
	}, nil
}

func asMap(value any) map[string]any {
	m, _ := value.(map[string]any)
	return m
}

func ensureString(target, installed map[string]any, key, value string) {
	if existing, ok := target[key].(string); ok && existing != "" {
		return
	}
	target[key] = value
	installed[key] = value
}

func copyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return cloneAnyMap(in)
}
