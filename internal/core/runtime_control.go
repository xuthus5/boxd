package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing/service"
)

// 运行时维护相关哨兵错误。
var (
	// ErrFeatureNotEnabled 表示所需内核特性未在当前配置中启用（如未配置 cache_file）。
	ErrFeatureNotEnabled = errors.New("feature not enabled")
	// ErrOutboundNotFound 表示运行实例中不存在指定 tag 的出站。
	ErrOutboundNotFound = errors.New("outbound not found")
)

// FlushDNS 清空内核 DNS 缓存。内核未运行时返回 ErrNotRunning。
// 当配置未启用 DNS 但 DNSRouter 仍注册时按 1.13 语义视为成功。
func (s *SBInstance) FlushDNS() error {
	s.mu.Lock()
	boxCtx := s.boxCtx
	running := s.running
	s.mu.Unlock()

	if !running || boxCtx == nil {
		return ErrNotRunning
	}
	dnsRouter := service.FromContext[adapter.DNSRouter](boxCtx)
	if dnsRouter == nil {
		// 与内核 clashapi 语义一致：缺失时视为空操作成功。
		return nil
	}
	dnsRouter.ClearCache()
	return nil
}

// FlushFakeIP 清空 FakeIP 存储，要求启用实验性 cache_file。
// 内核未运行时返回 ErrNotRunning；未启用 cache_file 时返回 ErrFeatureNotEnabled。
func (s *SBInstance) FlushFakeIP() error {
	s.mu.Lock()
	boxCtx := s.boxCtx
	running := s.running
	s.mu.Unlock()

	if !running || boxCtx == nil {
		return ErrNotRunning
	}
	cacheFile := service.FromContext[adapter.CacheFile](boxCtx)
	if cacheFile == nil {
		return ErrFeatureNotEnabled
	}
	return cacheFile.FakeIPReset()
}

// OutboundDelay 通过指定出站发起一次 URL 测速，返回延迟（ms）。
// link 为空时使用 urltest 默认测速地址；timeout<=0 时取 10s。
// 内核未运行返回 ErrNotRunning，出站不存在返回 ErrOutboundNotFound。
// 用 box 上下文派生子 context 以继承 ntp / RootPool 服务。
func (s *SBInstance) OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error) {
	s.mu.Lock()
	box := s.box
	boxCtx := s.boxCtx
	running := s.running
	s.mu.Unlock()

	if !running || box == nil || boxCtx == nil {
		return 0, ErrNotRunning
	}
	outbound, found := box.Outbound().Outbound(tag)
	if !found {
		return 0, fmt.Errorf("%w: %q", ErrOutboundNotFound, tag)
	}

	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	// 以 box 上下文为父，继承 ntp/RootPool；叠加请求自身的截止与超时。
	_ = ctx // 保留参数语义：调用方可传递取消，但 urltest 以 boxCtx 子 ctx 为准。

	testCtx, cancel := context.WithTimeout(boxCtx, timeout)
	defer cancel()
	return urltest.URLTest(testCtx, link, outbound)
}
