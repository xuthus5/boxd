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
		list = append(list, tc.snapshot())
		return true
	})
	return list
}

func (t *TrafficTracker) CloseConn(id int64) bool {
	v, ok := t.connections.LoadAndDelete(id)
	if !ok {
		return false
	}
	if tc, ok := v.(*trafficConnInternal); ok {
		tc.closeTracked()
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
		tc.closeTracked()
		count++
		return true
	})
	return count
}

// CloseConnsByIDs closes connections whose IDs are listed.
func (t *TrafficTracker) CloseConnsByIDs(ids []int64) int {
	if len(ids) == 0 {
		return 0
	}
	wanted := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id > 0 {
			wanted[id] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return 0
	}
	return t.CloseConnsWhere(func(tc *trafficConnInternal) bool {
		_, ok := wanted[tc.id]
		return ok
	})
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

// CloseConnsByProcess closes connections matched by the process path/name.
func (t *TrafficTracker) CloseConnsByProcess(process string) int {
	return t.CloseConnsWhere(func(tc *trafficConnInternal) bool {
		return tc.process == process
	})
}

func (t *TrafficTracker) newTrackedConnection(metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) *trafficConnInternal {
	id := t.nextID.Add(1)
	network := connectionNetwork(metadata)
	if network == "" {
		network = "tcp"
	}
	tc := &trafficConnInternal{
		id:       id,
		tracker:  t,
		target:   connectionTarget(metadata),
		outbound: matchOutbound.Tag(),
		rule:     ruleName(matchedRule),
		network:  network,
		source:   connectionSource(metadata),
		inbound:  connectionInbound(metadata),
		protocol: connectionProtocol(metadata),
		process:  connectionProcess(metadata),
		start:    time.Now(),
	}
	t.connections.Store(id, tc)
	return tc
}

func (t *TrafficTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	return t.newTrackedConnection(metadata, matchedRule, matchOutbound).wrap(conn)
}

func (t *TrafficTracker) RoutedPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) N.PacketConn {
	tc := t.newTrackedConnection(metadata, matchedRule, matchOutbound)
	if tc.network == "" || tc.network == "tcp" {
		tc.network = "udp"
	}
	return tc.wrapPacket(conn)
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
	closer   func() error
}

func (tc *trafficConnInternal) snapshot() TrafficConn {
	return TrafficConn{
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
	}
}

func (tc *trafficConnInternal) closeTracked() {
	if tc.closer != nil {
		_ = tc.closer()
	}
}

func (tc *trafficConnInternal) onTraffic(upload, download int64) {
	if upload > 0 {
		tc.upload.Add(upload)
		tc.tracker.uploadTotal.Add(upload)
	}
	if download > 0 {
		tc.download.Add(download)
		tc.tracker.downloadTotal.Add(download)
	}
}

func (tc *trafficConnInternal) wrap(conn net.Conn) net.Conn {
	tc.closer = conn.Close
	return &wrappedConn{
		Conn: conn,
		onRead: func(n int) {
			tc.onTraffic(0, int64(n))
		},
		onWrite: func(n int) {
			tc.onTraffic(int64(n), 0)
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
