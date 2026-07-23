package core

import (
	"context"
	"net"
	"strconv"
	"strings"
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
			Network:  tc.network,
			Source:   tc.source,
			Inbound:  tc.inbound,
			Protocol: tc.protocol,
			Process:  tc.process,
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
		target:   connectionTarget(metadata),
		outbound: matchOutbound.Tag(),
		rule:     ruleName(matchedRule),
		network:  connectionNetwork(metadata),
		source:   connectionSource(metadata),
		inbound:  connectionInbound(metadata),
		protocol: connectionProtocol(metadata),
		process:  connectionProcess(metadata),
		start:    time.Now(),
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
	network  string
	source   string
	inbound  string
	protocol string
	process  string
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
	Network  string    `json:"network,omitempty"`
	Source   string    `json:"source,omitempty"`
	Inbound  string    `json:"inbound,omitempty"`
	Protocol string    `json:"protocol,omitempty"`
	Process  string    `json:"process,omitempty"`
	Upload   int64     `json:"upload"`
	Download int64     `json:"download"`
	Start    time.Time `json:"start"`
}

func ruleName(rule adapter.Rule) string {
	if rule == nil {
		return ""
	}
	typeName := ""
	if typed, ok := any(rule).(interface{ Type() string }); ok {
		typeName = typed.Type()
	}
	return pickRuleName(typeName, rule.String())
}

// pickRuleName prefers a short rule type when present.
func pickRuleName(typeName, raw string) string {
	if name := strings.TrimSpace(typeName); name != "" {
		return name
	}
	return strings.TrimSpace(raw)
}

func connectionTarget(metadata adapter.InboundContext) string {
	if domain := strings.TrimSpace(metadata.Domain); domain != "" {
		return formatHostPort(domain, metadata.Destination.Port)
	}
	if host := strings.TrimSpace(metadata.Destination.Fqdn); host != "" {
		return formatHostPort(host, metadata.Destination.Port)
	}
	if metadata.Destination.Addr.IsValid() {
		return formatHostPort(metadata.Destination.Addr.String(), metadata.Destination.Port)
	}
	return ""
}

func formatHostPort(host string, port uint16) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if port == 0 {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}

func connectionSource(metadata adapter.InboundContext) string {
	if metadata.Source.IsValid() {
		return metadata.Source.String()
	}
	return ""
}

func connectionNetwork(metadata adapter.InboundContext) string {
	return strings.TrimSpace(metadata.Network)
}

func connectionInbound(metadata adapter.InboundContext) string {
	if tag := strings.TrimSpace(metadata.Inbound); tag != "" {
		return tag
	}
	return strings.TrimSpace(metadata.InboundType)
}

func connectionProtocol(metadata adapter.InboundContext) string {
	return strings.TrimSpace(metadata.Protocol)
}

func connectionProcess(metadata adapter.InboundContext) string {
	info := metadata.ProcessInfo
	if info == nil {
		return ""
	}
	if path := strings.TrimSpace(info.ProcessPath); path != "" {
		return path
	}
	if name := strings.TrimSpace(info.UserName); name != "" {
		return name
	}
	if len(info.AndroidPackageNames) > 0 {
		return strings.TrimSpace(info.AndroidPackageNames[0])
	}
	return ""
}
