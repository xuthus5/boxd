package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// minimalConfigTemplate 返回一份最小可用的 sing-box 配置。
// 仅包含基础日志、本地 mixed 入站与直连/阻止出站，保证内核可启动。
func minimalConfigTemplate() map[string]any {
	return map[string]any{
		"log": map[string]any{
			"level":     "info",
			"timestamp": true,
		},
		"inbounds": []any{
			map[string]any{
				"type": "mixed", "tag": "mixed-in",
				"listen": "::", "listen_port": 1080,
			},
		},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"route": map[string]any{
			"final": "direct",
		},
	}
}

// EnsureConfigFile 确保配置文件存在；不存在时生成最小可用配置并写入。
// 返回是否本次生成（created=true 表示自动生成了默认配置）。
// 父目录不存在时自动创建。
func EnsureConfigFile(path string) (bool, error) {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return false, fmt.Errorf("config path is a directory: %s", path)
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat config file: %w", err)
	}

	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}

	body, err := json.MarshalIndent(minimalConfigTemplate(), "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode default config: %w", err)
	}
	body = append(body, '\n')

	if err := atomicWriteFile0600(path, body); err != nil {
		return false, fmt.Errorf("write default config: %w", err)
	}
	return true, nil
}
