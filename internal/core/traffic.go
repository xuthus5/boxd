package core

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/adapter"
	N "github.com/sagernet/sing/common/network"
)

type TrafficTracker struct {
	uploadTotal   atomic.Int64
	downloadTotal atomic.Int64
	connections   sync.Map
	nextID        atomic.Int64
}

func NewTrafficTracker() *TrafficTracker {
	return &TrafficTracker{}
}

func (t *TrafficTracker) Total() (up, down int64) {
	return t.uploadTotal.Load(), t.downloadTotal.Load()
}

func (t *TrafficTracker) Connections() []TrafficConn {
	var list []TrafficConn
	t.connections.Range(func(key, value any) bool {
		tc, ok := value.(*trafficConnInternal)
		if !ok {
			return true
		}
		list = append(list, TrafficConn{
			ID:       tc.id,
			Target:   tc.target,
			Outbound: tc.outbound,
			Rule:     tc.rule,
			Upload:   tc.upload.Load(),
			Download: tc.download.Load(),
			Start:    tc.start,
		})
		return true
	})
	return list
}

func (t *TrafficTracker) CloseConn(id int64) bool {
	v, ok := t.connections.LoadAndDelete(id)
	if !ok {
		return false
	}
	if tc, ok := v.(*trafficConnInternal); ok && tc.conn != nil {
		_ = tc.conn.Close()
	}
	return true
}

func (t *TrafficTracker) CloseAllConns() int {
	return t.CloseConnsWhere(func(*trafficConnInternal) bool { return true })
}

// CloseConnsWhere closes connections matching pred and returns the closed count.
func (t *TrafficTracker) CloseConnsWhere(pred func(*trafficConnInternal) bool) int {
	if pred == nil {
		return 0
	}
	count := 0
	t.connections.Range(func(key, value any) bool {
		tc, ok := value.(*trafficConnInternal)
		if !ok || !pred(tc) {
			return true
		}
		t.connections.Delete(key)
		if tc.conn != nil {
			_ = tc.conn.Close()
		}
		count++
		return true
	})
	return count
}

// CloseConnsByOutbound closes connections for the given outbound tag.
func (t *TrafficTracker) CloseConnsByOutbound(outbound string) int {
	return t.CloseConnsWhere(func(tc *trafficConnInternal) bool {
		return tc.outbound == outbound
	})
}

// CloseConnsByRule closes connections matched by the given rule name.
func (t *TrafficTracker) CloseConnsByRule(rule string) int {
	return t.CloseConnsWhere(func(tc *trafficConnInternal) bool {
		return tc.rule == rule
	})
}

func (t *TrafficTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	id := t.nextID.Add(1)
	tc := &trafficConnInternal{
		id:       id,
		tracker:  t,
		target:   metadata.Destination.Fqdn,
		outbound: matchOutbound.Tag(),
		rule:     ruleName(matchedRule),
		start:    time.Now(),
	}
	if metadata.Destination.Fqdn == "" {
		tc.target = metadata.Destination.Addr.String()
	}
	t.connections.Store(id, tc)
	return tc.wrap(conn)
}

func (t *TrafficTracker) RoutedPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) N.PacketConn {
	return conn
}

type trafficConnInternal struct {
	id       int64
	tracker  *TrafficTracker
	target   string
	outbound string
	rule     string
	upload   atomic.Int64
	download atomic.Int64
	start    time.Time
	conn     net.Conn // 底层连接，用于按 id 关闭
}

func (tc *trafficConnInternal) wrap(conn net.Conn) net.Conn {
	tc.conn = conn
	return &wrappedConn{
		Conn: conn,
		onRead: func(n int) {
			val := int64(n)
			tc.download.Add(val)
			tc.tracker.downloadTotal.Add(val)
		},
		onWrite: func(n int) {
			val := int64(n)
			tc.upload.Add(val)
			tc.tracker.uploadTotal.Add(val)
		},
		onClose: func() {
			tc.tracker.connections.Delete(tc.id)
		},
	}
}

type wrappedConn struct {
	net.Conn
	onRead  func(int)
	onWrite func(int)
	onClose func()
}

func (w *wrappedConn) Read(b []byte) (int, error) {
	n, err := w.Conn.Read(b)
	if n > 0 {
		w.onRead(n)
	}
	return n, err
}

func (w *wrappedConn) Write(b []byte) (int, error) {
	n, err := w.Conn.Write(b)
	if n > 0 {
		w.onWrite(n)
	}
	return n, err
}

func (w *wrappedConn) Close() error {
	w.onClose()
	return w.Conn.Close()
}

type TrafficConn struct {
	ID       int64     `json:"id"`
	Target   string    `json:"target"`
	Outbound string    `json:"outbound"`
	Rule     string    `json:"rule,omitempty"`
	Upload   int64     `json:"upload"`
	Download int64     `json:"download"`
	Start    time.Time `json:"start"`
}

func ruleName(rule adapter.Rule) string {
	if rule == nil {
		return ""
	}
	// Rule.String() 通常包含类型信息；优先用 Type() 保持短可读。
	if typed, ok := any(rule).(interface{ Type() string }); ok {
		if name := typed.Type(); name != "" {
			return name
		}
	}
	return rule.String()
}
