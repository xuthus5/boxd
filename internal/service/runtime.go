package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// runtimeInstance 抽象内核运行时控制能力，便于测试注入。
type runtimeInstance interface {
	OutboundGroups() []core.OutboundGroupInfo
	SelectOutbound(groupTag, outTag string) error
	URLTestDelays(ctx context.Context, groupTag string) (map[string]uint16, error)
	FlushDNS() error
	FlushFakeIP() error
	OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error)
	ClashMode() (core.ClashModeStatus, error)
	SetClashMode(mode string) (core.ClashModeStatus, error)
}

// RuntimeService 提供运行时交互用例逻辑。
type RuntimeService struct {
	instance runtimeInstance
}

// NewRuntimeService 构造运行时用例服务。
func NewRuntimeService(instance runtimeInstance) *RuntimeService {
	return &RuntimeService{instance: instance}
}

// OutboundGroups 返回出站组列表。
func (s *RuntimeService) OutboundGroups(_ context.Context) ([]core.OutboundGroupInfo, error) {
	if s.instance == nil {
		return nil, Errorf(500, model.ErrorInternal, "service is not available")
	}
	groups := s.instance.OutboundGroups()
	if groups == nil {
		groups = []core.OutboundGroupInfo{}
	}
	return groups, nil
}

// SelectOutbound 切换选择器组的选中出站。
func (s *RuntimeService) SelectOutbound(_ context.Context, group, tag string) (string, error) {
	if group == "" {
		return "", Errorf(400, model.ErrorInvalidRequest, "group is required")
	}
	if tag == "" {
		return "", Errorf(400, model.ErrorInvalidRequest, "tag is required")
	}
	if s.instance == nil {
		return "", Errorf(500, model.ErrorInternal, "service is not available")
	}
	if err := s.instance.SelectOutbound(group, tag); err != nil {
		return "", mapRuntimeError(err)
	}
	return tag, nil
}

// URLTestDelays 触发分组 URLTest 延迟测试。
func (s *RuntimeService) URLTestDelays(ctx context.Context, group string) (map[string]uint16, error) {
	if group == "" {
		return nil, Errorf(400, model.ErrorInvalidRequest, "group is required")
	}
	if s.instance == nil {
		return nil, Errorf(500, model.ErrorInternal, "service is not available")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	delays, err := s.instance.URLTestDelays(probeCtx, group)
	if err != nil {
		return nil, mapRuntimeError(err)
	}
	return delays, nil
}

// FlushDNS 清空内核 DNS 缓存。
func (s *RuntimeService) FlushDNS(_ context.Context) error {
	if s.instance == nil {
		return Errorf(500, model.ErrorInternal, "service is not available")
	}
	if err := s.instance.FlushDNS(); err != nil {
		return mapRuntimeError(err)
	}
	return nil
}

// FlushFakeIP 清空 FakeIP 存储。
func (s *RuntimeService) FlushFakeIP(_ context.Context) error {
	if s.instance == nil {
		return Errorf(500, model.ErrorInternal, "service is not available")
	}
	if err := s.instance.FlushFakeIP(); err != nil {
		return mapRuntimeError(err)
	}
	return nil
}

// OutboundDelay 单出站延迟测试。
func (s *RuntimeService) OutboundDelay(ctx context.Context, tag, link string, timeoutMs int64) (uint16, error) {
	if tag == "" {
		return 0, Errorf(400, model.ErrorInvalidRequest, "tag is required")
	}
	if link != "" {
		if err := core.ValidateHTTPURL(link); err != nil {
			return 0, Errorf(400, model.ErrorInvalidRequest, "%v", err)
		}
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if timeoutMs > int64((60*time.Second)/time.Millisecond) {
		return 0, Errorf(400, model.ErrorInvalidRequest, "invalid timeout")
	}
	if s.instance == nil {
		return 0, Errorf(500, model.ErrorInternal, "service is not available")
	}
	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	delay, err := s.instance.OutboundDelay(probeCtx, tag, link, time.Duration(timeoutMs)*time.Millisecond)
	if err != nil {
		return 0, mapRuntimeError(err)
	}
	if delay == 0 {
		return 0, Errorf(502, model.ErrorRuntimeDelayFailed, "delay test failed: no response")
	}
	return delay, nil
}

// GetClashMode 返回 Clash 模式。
func (s *RuntimeService) GetClashMode(_ context.Context) (core.ClashModeStatus, error) {
	if s.instance == nil {
		return core.ClashModeStatus{}, Errorf(500, model.ErrorInternal, "service is not available")
	}
	status, err := s.instance.ClashMode()
	if err != nil {
		return core.ClashModeStatus{}, mapRuntimeError(err)
	}
	return status, nil
}

// SetClashMode 设置 Clash 模式。
func (s *RuntimeService) SetClashMode(_ context.Context, mode string) (core.ClashModeStatus, error) {
	if s.instance == nil {
		return core.ClashModeStatus{}, Errorf(500, model.ErrorInternal, "service is not available")
	}
	status, err := s.instance.SetClashMode(mode)
	if err != nil {
		return core.ClashModeStatus{}, mapRuntimeError(err)
	}
	return status, nil
}

func mapRuntimeError(err error) *DomainError {
	switch {
	case errors.Is(err, core.ErrNotRunning):
		return Errorf(503, model.ErrorUnavailable, "%v", err)
	case errors.Is(err, core.ErrGroupNotFound):
		return Errorf(404, model.ErrorRuntimeGroupNotFound, "%v", err)
	case errors.Is(err, core.ErrNotSelectable), errors.Is(err, core.ErrTagNotInGroup):
		return Errorf(400, model.ErrorRuntimeNotSelectable, "%v", err)
	case errors.Is(err, core.ErrFeatureNotEnabled):
		return Errorf(400, model.ErrorInvalidRequest, "%v", err)
	case errors.Is(err, core.ErrInvalidMode):
		return Errorf(400, model.ErrorInvalidRequest, "%v", err)
	case errors.Is(err, core.ErrOutboundNotFound):
		return Errorf(404, model.ErrorNotFound, "%v", err)
	default:
		return Errorf(500, model.ErrorInternal, "%v", err)
	}
}

// ParseTimeout 解析毫秒超时字符串。
func ParseTimeout(raw string, fallback int64) (int64, error) {
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 || parsed > int64((60*time.Second)/time.Millisecond) {
		return 0, fmt.Errorf("invalid timeout")
	}
	return parsed, nil
}
