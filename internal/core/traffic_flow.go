package core

import (
	"context"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	tun "github.com/sagernet/sing-tun"
)

var _ tun.FlowTracker = (*trafficFlow)(nil)

// RoutedFlow 记录 1.14 的三层转发流，保留统计、连接列表与手动断开能力。
func (t *TrafficTracker) RoutedFlow(_ context.Context, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) tun.FlowTracker {
	flow := &trafficFlow{connection: t.makeTrackedConnection(metadata, matchedRule, matchOutbound)}
	flow.connection.closer = flow.close
	return flow
}

type trafficFlow struct {
	mu         sync.Mutex
	connection *trafficConnInternal
	handle     tun.FlowHandle
	closed     bool
}

func (f *trafficFlow) AttachFlow(handle tun.FlowHandle) {
	if handle == nil {
		return
	}
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		handle.CloseFlow()
		return
	}
	f.handle = handle
	f.connection.tracker.connections.Store(f.connection.id, f.connection)
	f.mu.Unlock()
}

func (f *trafficFlow) CountForward(size int) {
	f.connection.onTraffic(int64(size), 0)
}

func (f *trafficFlow) CountReverse(size int) {
	f.connection.onTraffic(0, int64(size))
}

func (f *trafficFlow) FlowEstablished() {
	// 列表与计数在 AttachFlow 时生效，不将 TCP 握手状态强加给 UDP/ICMP 流。
}

func (f *trafficFlow) CloseFlow(_ tun.FlowCloseReason) {
	f.mu.Lock()
	f.closed = true
	f.handle = nil
	f.connection.tracker.connections.Delete(f.connection.id)
	f.mu.Unlock()
}

func (f *trafficFlow) close() error {
	f.mu.Lock()
	handle := f.handle
	f.handle = nil
	f.closed = true
	f.connection.tracker.connections.Delete(f.connection.id)
	f.mu.Unlock()
	// 内核的 CloseFlow 会回调本 tracker，不能在持锁时调用。
	if handle != nil {
		handle.CloseFlow()
	}
	return nil
}
