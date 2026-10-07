package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// outboundSelector 抽象内核运行时的出口切换能力，便于按需注入与测试。
type outboundSelector interface {
	SelectOutbound(groupTag, outTag string) error
}

// SubscriptionInput 订阅创建/更新请求。
type SubscriptionInput struct {
	Name        string                  `json:"name"`
	URL         string                  `json:"url"`
	IntervalMin int                     `json:"interval_min"`
	URLTest     *model.URLTestOverrides `json:"urltest"`
}

func (r SubscriptionInput) params() core.SubscriptionParams {
	return core.SubscriptionParams{
		Name:        r.Name,
		URL:         r.URL,
		IntervalMin: r.IntervalMin,
		URLTest:     r.URLTest,
	}
}

// SubscriptionService 提供订阅管理用例逻辑。
type SubscriptionService struct {
	manager    *core.SubscriptionManager
	nodeMgr    *core.NodeManager
	configPath string
	instance   restartable
}

// NewSubscriptionService 构造订阅用例服务。
func NewSubscriptionService(manager *core.SubscriptionManager, nodeMgr *core.NodeManager, configPath string, instance restartable) *SubscriptionService {
	return &SubscriptionService{manager: manager, nodeMgr: nodeMgr, configPath: configPath, instance: instance}
}

// List 返回全部订阅。
func (s *SubscriptionService) List(_ context.Context) ([]model.Subscription, error) {
	subs, err := s.manager.List()
	if err != nil {
		return nil, Errorf(500, model.ErrorInternal, "failed to load subscriptions")
	}
	return subs, nil
}

// Create 创建订阅。
func (s *SubscriptionService) Create(_ context.Context, input SubscriptionInput) (*model.Subscription, error) {
	if input.Name == "" || input.URL == "" {
		return nil, Errorf(400, model.ErrorInvalidRequest, "name and url are required")
	}
	if err := core.ValidateSubscriptionURL(input.URL); err != nil {
		return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if input.IntervalMin <= 0 {
		input.IntervalMin = 60
	}
	if err := core.ValidateURLTestOverrides(input.URLTest); err != nil {
		return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	sub, err := s.manager.Create(input.params())
	if err != nil {
		slog.Error("subscription create failed", "name", input.Name, "err", err)
		return nil, Errorf(500, model.ErrorInternal, "%v", err)
	}
	slog.Info("subscription created", "id", sub.ID, "name", input.Name, "interval", input.IntervalMin)
	return sub, nil
}

// Get 返回单个订阅。
func (s *SubscriptionService) Get(_ context.Context, id string) (*model.Subscription, error) {
	sub := s.manager.Get(id)
	if sub == nil {
		return nil, Errorf(404, model.ErrorSubscriptionNotFound, "subscription not found")
	}
	return sub, nil
}

// Update 更新订阅。
func (s *SubscriptionService) Update(_ context.Context, id string, input SubscriptionInput) error {
	if input.URL != "" {
		if err := core.ValidateSubscriptionURL(input.URL); err != nil {
			return Errorf(400, model.ErrorInvalidRequest, "%v", err)
		}
	}
	if err := core.ValidateURLTestOverrides(input.URLTest); err != nil {
		return Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if err := s.manager.Update(id, input.params()); err != nil {
		slog.Error("subscription update failed", "id", id, "err", err)
		return Errorf(404, model.ErrorSubscriptionNotFound, "%v", err)
	}
	slog.Info("subscription updated", "id", id, "name", input.Name)
	return nil
}

// Delete 删除订阅。
func (s *SubscriptionService) Delete(_ context.Context, id string) error {
	if err := s.manager.Delete(id); err != nil {
		slog.Error("subscription delete failed", "id", id, "err", err)
		return Errorf(404, model.ErrorSubscriptionNotFound, "%v", err)
	}
	slog.Info("subscription deleted", "id", id)
	return nil
}

// SetSelected 持久化订阅在 selector 模式下的出口节点并同步配置；内核运行时会即时切换。
func (s *SubscriptionService) SetSelected(_ context.Context, id, tag string) error {
	subscription := s.manager.Get(id)
	if subscription == nil {
		return Errorf(404, model.ErrorSubscriptionNotFound, "subscription not found")
	}
	if err := core.ValidateSubscriptionSelection(*subscription, tag); err != nil {
		return Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if _, err := s.manager.SetSelected(id, tag); err != nil {
		slog.Error("subscription selection update failed", "id", id, "tag", tag, "err", err)
		return Errorf(500, model.ErrorInternal, "%v", err)
	}
	if s.nodeMgr != nil {
		if err := SyncOutboundsToConfig(s.nodeMgr, s.manager, s.configPath); err != nil {
			if _, restoreErr := s.manager.SetSelected(id, subscription.Selected); restoreErr != nil {
				slog.Error("restoring previous subscription selection failed", "id", id, "err", restoreErr)
			}
			slog.Error("subscription selection config sync failed", "id", id, "err", err)
			return Errorf(500, model.ErrorSubscriptionSync, "%v", subscriptionSyncErrorMessage(err))
		}
	}
	s.applyRuntimeSelection(subscription.Name, tag)
	slog.Info("subscription selection updated", "id", id, "name", subscription.Name, "tag", tag)
	return nil
}

// applyRuntimeSelection 在内核运行时即时切换出口；未运行时留待内核启动后读取配置默认值。
func (s *SubscriptionService) applyRuntimeSelection(group, tag string) {
	if tag == "" {
		return
	}
	selector, ok := s.instance.(outboundSelector)
	if !ok {
		return
	}
	if err := selector.SelectOutbound(group, tag); err != nil && !errors.Is(err, core.ErrNotRunning) {
		slog.Warn("subscription selection saved but runtime switch failed",
			"group", group, "tag", tag, "err", err,
		)
	}
}

// Refresh 刷新单个订阅并同步配置。
func (s *SubscriptionService) Refresh(ctx context.Context, id string) error {
	slog.Info("subscription refresh started", "id", id)
	if err := s.manager.RefreshContext(ctx, id); err != nil {
		slog.Error("subscription refresh failed", "id", id, "err", err)
		return Errorf(500, model.ErrorSubscriptionRefresh, "%v", err)
	}
	if err := s.syncConfig(); err != nil {
		slog.Error("subscription config sync failed after refresh", "id", id, "err", err)
		return Errorf(500, model.ErrorSubscriptionSync, "%v", subscriptionSyncErrorMessage(err))
	}
	slog.Info("subscription refresh completed", "id", id)
	return nil
}

// RefreshAll 刷新全部订阅并同步配置。
func (s *SubscriptionService) RefreshAll(ctx context.Context) (RefreshAllResult, error) {
	slog.Info("subscription refresh-all started")
	failures := s.manager.RefreshAllContext(ctx)
	if failures == nil {
		failures = []core.SubscriptionRefreshFailure{}
	}
	syncErr := s.syncConfig()
	syncMessage := ""
	if syncErr != nil {
		syncMessage = subscriptionSyncErrorMessage(syncErr)
	}
	if len(failures) > 0 || syncErr != nil {
		slog.Warn("subscription refresh-all completed with issues",
			"failed", len(failures),
			"sync_failed", syncErr != nil,
		)
	} else {
		slog.Info("subscription refresh-all completed")
	}
	return RefreshAllResult{
		Failures:    failures,
		SyncError:   syncMessage,
		FailedCount: len(failures),
		SyncFailed:  syncErr != nil,
	}, nil
}

// RefreshAllResult 描述批量刷新结果。
type RefreshAllResult struct {
	Failures    []core.SubscriptionRefreshFailure `json:"failed"`
	SyncError   string                            `json:"sync_error,omitempty"`
	FailedCount int                               `json:"failed_count"`
	SyncFailed  bool                              `json:"sync_failed"`
}

func (s *SubscriptionService) syncConfig() error {
	if s.nodeMgr == nil || s.manager == nil {
		return nil
	}
	return SyncOutboundsAndRestart(s.nodeMgr, s.manager, s.configPath, s.instance)
}

func subscriptionSyncErrorMessage(err error) string {
	detail := ""
	if err != nil {
		detail = strings.TrimSpace(err.Error())
	}
	if detail == "" {
		return "subscription refreshed but configuration sync failed"
	}
	return "subscription refreshed but configuration sync failed: " + detail
}
