package service

import (
	"context"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// RuleSetService 封装规则集状态、更新与自动更新配置，供传输层复用。
type RuleSetService struct {
	updater  *core.RuleSetUpdater
	settings *core.SettingsManager
}

// NewRuleSetService 构造规则集服务。
func NewRuleSetService(updater *core.RuleSetUpdater, settings *core.SettingsManager) *RuleSetService {
	return &RuleSetService{updater: updater, settings: settings}
}

// Status 返回规则集状态列表。
func (s *RuleSetService) Status(ctx context.Context) ([]model.RuleSetStatusItem, error) {
	if s.updater == nil {
		return nil, Errorf(501, model.ErrorInternal, "rule-set updater is not configured")
	}
	return s.updater.Status(ctx)
}

// Update 按请求更新规则集。
func (s *RuleSetService) Update(ctx context.Context, req core.RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error) {
	if s.updater == nil {
		return model.RuleSetUpdateResponse{}, Errorf(501, model.ErrorInternal, "rule-set updater is not configured")
	}
	return s.updater.Update(ctx, req)
}

// AutoUpdate 返回规则集自动更新配置。
func (s *RuleSetService) AutoUpdate() (model.RuleSetAutoUpdate, error) {
	if s.settings == nil {
		return model.RuleSetAutoUpdate{}, Errorf(501, model.ErrorInternal, "settings are not configured")
	}
	return s.settings.RuleSetAutoUpdate()
}

// SetAutoUpdate 保存规则集自动更新配置。
func (s *RuleSetService) SetAutoUpdate(cfg model.RuleSetAutoUpdate) error {
	if s.settings == nil {
		return Errorf(501, model.ErrorInternal, "settings are not configured")
	}
	return s.settings.SetRuleSetAutoUpdate(cfg)
}
