package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// Diagnostics 返回当前配置的诊断报告（不应用配置）。
func (c *Config) Diagnostics() model.ConfigDiagnostics {
	if c == nil {
		return model.ConfigDiagnostics{}
	}
	return core.AnalyzeConfigFile(c.path)
}

// ApplyHistory 返回最近的配置应用事件时间线。
func (c *Config) ApplyHistory() ([]model.ConfigApplyEvent, error) {
	if c == nil || c.applyHistory == nil {
		return []model.ConfigApplyEvent{}, nil
	}
	return c.applyHistory.List(0)
}

// ConfigSnapshot 返回指定历史快照的原始配置内容。
func (c *Config) ConfigSnapshot(id string) ([]byte, error) {
	if c == nil || c.applyHistory == nil {
		return nil, Errorf(404, model.ErrorNotFound, "config snapshot not found")
	}
	body, err := c.applyHistory.GetSnapshot(id)
	if err != nil {
		if errors.Is(err, core.ErrConfigSnapshotNotFound) {
			return nil, Errorf(404, model.ErrorNotFound, "config snapshot not found")
		}
		return nil, Errorf(500, model.ErrorInternal, "failed to load config snapshot")
	}
	return body, nil
}

// RestoreConfigSnapshot 从历史快照恢复配置并重启内核。
func (c *Config) RestoreConfigSnapshot(ctx context.Context, id string) (ApplyResult, error) {
	if c == nil || c.applyHistory == nil {
		return ApplyResult{}, Errorf(404, model.ErrorNotFound, "config snapshot not found")
	}
	body, err := c.applyHistory.GetSnapshot(id)
	if err != nil {
		if errors.Is(err, core.ErrConfigSnapshotNotFound) {
			return ApplyResult{}, Errorf(404, model.ErrorNotFound, "config snapshot not found")
		}
		return ApplyResult{}, Errorf(500, model.ErrorInternal, "failed to load config snapshot")
	}
	return c.writeConfigFile(ctx, body, "restore")
}

// RouteRules 返回配置中的路由规则列表。
func (c *Config) RouteRules() ([]any, error) {
	if c == nil {
		return nil, Errorf(500, model.ErrorInternal, "config service is not available")
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to read config")
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to decode config")
	}
	route, _ := config["route"].(map[string]any)
	rules, _ := route["rules"].([]any)
	return rules, nil
}

// GetRouteRuleMetadata 返回路由规则元数据。
func (c *Config) GetRouteRuleMetadata() ([]model.RouteRuleMetadata, error) {
	if c == nil || c.routeMetadata == nil {
		return nil, Errorf(501, model.ErrorInternal, "route rule metadata is not configured")
	}
	rules, err := c.RouteRules()
	if err != nil {
		return nil, err
	}
	if err := c.routeMetadata.InitializeDefaultNames(rules); err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to initialize route rule metadata")
	}
	return c.routeMetadata.List(rules)
}

// UpdateRouteRuleMetadata 保存路由规则元数据并返回最新列表。
func (c *Config) UpdateRouteRuleMetadata(metadata []model.RouteRuleMetadata) ([]model.RouteRuleMetadata, error) {
	if c == nil || c.routeMetadata == nil {
		return nil, Errorf(501, model.ErrorInternal, "route rule metadata is not configured")
	}
	rules, err := c.RouteRules()
	if err != nil {
		return nil, err
	}
	if err := c.routeMetadata.Save(rules, metadata); err != nil {
		if errors.Is(err, core.ErrInvalidRouteRuleMetadata) {
			return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
		}
		return nil, Errorf(500, model.ErrorInternal, "failed to save route rule metadata")
	}
	return c.routeMetadata.List(rules)
}
