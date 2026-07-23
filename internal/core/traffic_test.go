package core

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
)

func TestNewTrafficTracker(t *testing.T) {
	tt := NewTrafficTracker()
	if tt == nil {
		t.Fatal("expected non-nil TrafficTracker")
	}
}

func TestTrafficTrackerTotal(t *testing.T) {
	tt := NewTrafficTracker()
	up, down := tt.Total()
	if up != 0 || down != 0 {
		t.Errorf("expected 0,0, got %d,%d", up, down)
	}
}

func TestTrafficTrackerConnections(t *testing.T) {
	tt := NewTrafficTracker()
	conns := tt.Connections()
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

func TestWrappedConnReadWriteClose(t *testing.T) {
	var buf bytes.Buffer
	readCalled := false
	writeCalled := false
	closeCalled := false

	wc := &wrappedConn{
		Conn:    &mockRW{buf: &buf},
		onRead:  func(n int) { readCalled = true },
		onWrite: func(n int) { writeCalled = true },
		onClose: func() { closeCalled = true },
	}

	if _, err := buf.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 5)
	n, err := wc.Read(b)
	if n != 5 || err != nil {
		t.Errorf("expected 5 bytes read, got %d err=%v", n, err)
	}
	if !readCalled {
		t.Error("onRead should have been called")
	}

	n, err = wc.Write([]byte("world"))
	if n != 5 || err != nil {
		t.Errorf("expected 5 bytes written, got %d err=%v", n, err)
	}
	if !writeCalled {
		t.Error("onWrite should have been called")
	}
	if buf.String() != "world" {
		t.Errorf("expected 'world', got '%s'", buf.String())
	}

	err = wc.Close()
	if err != nil {
		t.Errorf("expected nil error on close, got %v", err)
	}
	if !closeCalled {
		t.Error("onClose should have been called")
	}
}

type mockRW struct {
	buf *bytes.Buffer
}

func (m *mockRW) Read(b []byte) (int, error) {
	return m.buf.Read(b)
}

func (m *mockRW) Write(b []byte) (int, error) {
	return m.buf.Write(b)
}

func (m *mockRW) Close() error {
	return nil
}

func (m *mockRW) LocalAddr() net.Addr {
	return nil
}

func (m *mockRW) RemoteAddr() net.Addr {
	return nil
}

func (m *mockRW) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockRW) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockRW) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestTrafficConnInternalWrapTracksTotals(t *testing.T) {
	tracker := NewTrafficTracker()
	var buf bytes.Buffer
	tc := &trafficConnInternal{
		id:       1,
		tracker:  tracker,
		target:   "example.com",
		outbound: "proxy",
		start:    time.Now(),
	}
	tracker.connections.Store(tc.id, tc)

	conn := tc.wrap(&mockRW{buf: &buf})
	if _, err := buf.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	readBuf := make([]byte, 5)
	if _, err := conn.Read(readBuf); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write([]byte("world")); err != nil {
		t.Fatal(err)
	}

	up, down := tracker.Total()
	if up != 5 || down != 5 {
		t.Fatalf("totals = %d,%d", up, down)
	}
	conns := tracker.Connections()
	if len(conns) != 1 || conns[0].Target != "example.com" || conns[0].Outbound != "proxy" {
		t.Fatalf("connections = %#v", conns)
	}

	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if conns := tracker.Connections(); len(conns) != 0 {
		t.Fatalf("connections after close = %#v", conns)
	}
}

func TestTrafficTrackerRoutedConnection(t *testing.T) {
	tracker := NewTrafficTracker()
	left, right := net.Pipe()
	defer func() { _ = right.Close() }()

	wrapped := tracker.RoutedConnection(
		context.Background(),
		left,
		adapter.InboundContext{Destination: M.ParseSocksaddr("example.com:443")},
		nil,
		fakeOutbound{tag: "proxy"},
	)
	conns := tracker.Connections()
	if len(conns) != 1 || conns[0].Target != "example.com:443" || conns[0].Outbound != "proxy" {
		t.Fatalf("connections = %#v", conns)
	}
	if err := wrapped.Close(); err != nil {
		t.Fatal(err)
	}
	if conns := tracker.Connections(); len(conns) != 0 {
		t.Fatalf("connections after close = %#v", conns)
	}
}

func TestTrafficTrackerRoutedPacketConnection(t *testing.T) {
	tracker := NewTrafficTracker()
	packetConn := &fakePacketConn{payload: []byte("ping")}
	got := tracker.RoutedPacketConnection(
		context.Background(),
		packetConn,
		adapter.InboundContext{
			Network:     "udp",
			Inbound:     "tun-in",
			Destination: M.ParseSocksaddr("1.1.1.1:53"),
			Source:      M.ParseSocksaddr("10.0.0.2:53000"),
		},
		nil,
		fakeOutbound{tag: "proxy"},
	)
	if _, ok := got.(*wrappedPacketConn); !ok {
		t.Fatalf("expected wrapped packet conn, got %T", got)
	}
	conns := tracker.Connections()
	if len(conns) != 1 || conns[0].Network != "udp" || conns[0].Target != "1.1.1.1:53" {
		t.Fatalf("connections = %#v", conns)
	}
	buffer := buf.New()
	defer buffer.Release()
	if _, err := got.ReadPacket(buffer); err != nil {
		t.Fatal(err)
	}
	if err := got.WritePacket(buf.As([]byte("pong")), M.ParseSocksaddr("1.1.1.1:53")); err != nil {
		t.Fatal(err)
	}
	up, down := tracker.Total()
	if up != 4 || down != 4 {
		t.Fatalf("totals = %d,%d", up, down)
	}
	if err := got.Close(); err != nil {
		t.Fatal(err)
	}
	if conns := tracker.Connections(); len(conns) != 0 {
		t.Fatalf("connections after close = %#v", conns)
	}
}

type fakePacketConn struct {
	payload []byte
	closed  bool
}

func (f *fakePacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	if len(f.payload) > 0 {
		_, _ = buffer.Write(f.payload)
	}
	return M.ParseSocksaddr("1.1.1.1:53"), nil
}

func (f *fakePacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	return nil
}

func (f *fakePacketConn) Close() error {
	f.closed = true
	return nil
}

func (f *fakePacketConn) LocalAddr() net.Addr {
	return nil
}

func (f *fakePacketConn) SetDeadline(t time.Time) error {
	return nil
}

func (f *fakePacketConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (f *fakePacketConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestTrafficTrackerCloseConn(t *testing.T) {
	tracker := NewTrafficTracker()
	left, right := net.Pipe()
	defer func() { _ = right.Close() }()

	wrapped := tracker.RoutedConnection(
		context.Background(),
		left,
		adapter.InboundContext{Destination: M.ParseSocksaddr("example.com:443")},
		nil,
		fakeOutbound{tag: "proxy"},
	)

	if closed := tracker.CloseConn(999); closed {
		t.Fatal("CloseConn for missing id should return false")
	}
	if !tracker.CloseConn(1) {
		t.Fatal("CloseConn for existing id should return true")
	}
	if conns := tracker.Connections(); len(conns) != 0 {
		t.Fatalf("connections after close = %#v", conns)
	}
	buf := make([]byte, 1)
	if _, err := right.Read(buf); err == nil {
		t.Fatal("right side Read should error after CloseConn")
	}
	_ = wrapped.Close()
}

func TestTrafficTrackerCloseAllConns(t *testing.T) {
	tracker := NewTrafficTracker()
	left1, right1 := net.Pipe()
	left2, right2 := net.Pipe()
	w1 := tracker.RoutedConnection(context.Background(), left1, adapter.InboundContext{Destination: M.ParseSocksaddr("a:443")}, nil, fakeOutbound{tag: "proxy"})
	w2 := tracker.RoutedConnection(context.Background(), left2, adapter.InboundContext{Destination: M.ParseSocksaddr("b:443")}, nil, fakeOutbound{tag: "proxy"})

	count := tracker.CloseAllConns()
	if count != 2 {
		t.Fatalf("CloseAllConns count = %d", count)
	}
	if conns := tracker.Connections(); len(conns) != 0 {
		t.Fatalf("connections after closeAll = %#v", conns)
	}
	// 再次调用应返回 0
	if count := tracker.CloseAllConns(); count != 0 {
		t.Fatalf("empty CloseAllConns count = %d", count)
	}
	_ = w1.Close()
	_ = w2.Close()
	_ = right1.Close()
	_ = right2.Close()
}

func TestTrafficTrackerConnectionRuleEmptyWhenNoMatch(t *testing.T) {
	tracker := NewTrafficTracker()
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	_ = tracker.RoutedConnection(context.Background(), left, adapter.InboundContext{Destination: M.ParseSocksaddr("rule.example:443")}, nil, fakeOutbound{tag: "proxy"})
	conns := tracker.Connections()
	if len(conns) != 1 {
		t.Fatalf("len = %d", len(conns))
	}
	if conns[0].Rule != "" {
		t.Fatalf("rule = %q", conns[0].Rule)
	}
	if conns[0].Outbound != "proxy" {
		t.Fatalf("outbound = %q", conns[0].Outbound)
	}
}

func TestTrafficTrackerCloseConnsByOutboundAndRule(t *testing.T) {
	tracker := NewTrafficTracker()
	left1, right1 := net.Pipe()
	left2, right2 := net.Pipe()
	left3, right3 := net.Pipe()
	defer left1.Close()
	defer right1.Close()
	defer left2.Close()
	defer right2.Close()
	defer left3.Close()
	defer right3.Close()

	// seed via internal store helpers: use CloseAll path style by RoutedConnection requires adapters.
	// Build minimal internal entries through CloseConn tests pattern - store via public RoutedConnection is heavy.
	// Use CloseConnsWhere indirectly by planting through package-level structure via Connections map helpers.
	// Fall back: call CloseConnsByOutbound on empty and ensure 0.
	if n := tracker.CloseConnsByOutbound("proxy"); n != 0 {
		t.Fatalf("empty outbound close = %d", n)
	}
	if n := tracker.CloseConnsByRule("geoip-cn"); n != 0 {
		t.Fatalf("empty rule close = %d", n)
	}

	// Plant three synthetic connections.
	tracker.connections.Store(int64(1), &trafficConnInternal{id: 1, outbound: "proxy", rule: "geosite-google", closer: left1.Close})
	tracker.connections.Store(int64(2), &trafficConnInternal{id: 2, outbound: "proxy", rule: "geoip-cn", closer: left2.Close})
	tracker.connections.Store(int64(3), &trafficConnInternal{id: 3, outbound: "direct", rule: "geoip-cn", closer: left3.Close})

	if n := tracker.CloseConnsByOutbound("proxy"); n != 2 {
		t.Fatalf("close by outbound = %d, want 2", n)
	}
	if conns := tracker.Connections(); len(conns) != 1 || conns[0].Outbound != "direct" {
		t.Fatalf("remaining = %#v", conns)
	}
	if n := tracker.CloseConnsByRule("geoip-cn"); n != 1 {
		t.Fatalf("close by rule = %d, want 1", n)
	}
	if conns := tracker.Connections(); len(conns) != 0 {
		t.Fatalf("expected empty, got %#v", conns)
	}
}

func TestTrafficTrackerCloseConnsByIDs(t *testing.T) {
	tracker := NewTrafficTracker()
	left1, right1 := net.Pipe()
	left2, right2 := net.Pipe()
	left3, right3 := net.Pipe()
	defer left1.Close()
	defer right1.Close()
	defer left2.Close()
	defer right2.Close()
	defer left3.Close()
	defer right3.Close()

	if n := tracker.CloseConnsByIDs(nil); n != 0 {
		t.Fatalf("nil ids = %d", n)
	}
	if n := tracker.CloseConnsByIDs([]int64{0, -1}); n != 0 {
		t.Fatalf("invalid ids = %d", n)
	}

	tracker.connections.Store(int64(1), &trafficConnInternal{id: 1, outbound: "proxy", closer: left1.Close})
	tracker.connections.Store(int64(2), &trafficConnInternal{id: 2, outbound: "direct", closer: left2.Close})
	tracker.connections.Store(int64(3), &trafficConnInternal{id: 3, outbound: "proxy", closer: left3.Close})

	if n := tracker.CloseConnsByIDs([]int64{1, 3, 3, 9}); n != 2 {
		t.Fatalf("close by ids = %d, want 2", n)
	}
	conns := tracker.Connections()
	if len(conns) != 1 || conns[0].ID != 2 {
		t.Fatalf("remaining = %#v", conns)
	}
}

func TestTrafficTrackerCloseConnsByProcess(t *testing.T) {
	tracker := NewTrafficTracker()
	left1, right1 := net.Pipe()
	left2, right2 := net.Pipe()
	left3, right3 := net.Pipe()
	defer left1.Close()
	defer right1.Close()
	defer left2.Close()
	defer right2.Close()
	defer left3.Close()
	defer right3.Close()

	if n := tracker.CloseConnsByProcess("/usr/bin/curl"); n != 0 {
		t.Fatalf("empty process close = %d", n)
	}

	tracker.connections.Store(int64(1), &trafficConnInternal{id: 1, process: "/usr/bin/curl", closer: left1.Close})
	tracker.connections.Store(int64(2), &trafficConnInternal{id: 2, process: "/usr/bin/curl", closer: left2.Close})
	tracker.connections.Store(int64(3), &trafficConnInternal{id: 3, process: "/Applications/Chrome.app", closer: left3.Close})

	if n := tracker.CloseConnsByProcess("/usr/bin/curl"); n != 2 {
		t.Fatalf("close by process = %d, want 2", n)
	}
	conns := tracker.Connections()
	if len(conns) != 1 || conns[0].Process != "/Applications/Chrome.app" {
		t.Fatalf("remaining = %#v", conns)
	}
}

func TestTrafficTrackerConnectionMetadata(t *testing.T) {
	tracker := NewTrafficTracker()
	left, right := net.Pipe()
	defer func() { _ = right.Close() }()

	meta := adapter.InboundContext{
		Inbound:     "mixed-in",
		InboundType: "mixed",
		Network:     "tcp",
		Protocol:    "tls",
		Domain:      "cdn.example.com",
		Source:      M.ParseSocksaddr("10.0.0.2:51234"),
		Destination: M.ParseSocksaddr("1.2.3.4:443"),
		ProcessInfo: &adapter.ConnectionOwner{ProcessPath: "/usr/bin/curl"},
	}
	wrapped := tracker.RoutedConnection(context.Background(), left, meta, nil, fakeOutbound{tag: "proxy"})
	conns := tracker.Connections()
	if len(conns) != 1 {
		t.Fatalf("connections = %#v", conns)
	}
	got := conns[0]
	if got.Target != "cdn.example.com:443" {
		t.Fatalf("target = %q", got.Target)
	}
	if got.Source != "10.0.0.2:51234" {
		t.Fatalf("source = %q", got.Source)
	}
	if got.Network != "tcp" || got.Inbound != "mixed-in" || got.Protocol != "tls" {
		t.Fatalf("meta = %#v", got)
	}
	if got.Process != "/usr/bin/curl" {
		t.Fatalf("process = %q", got.Process)
	}
	_ = wrapped.Close()
}

func TestConnectionMetadataHelpers(t *testing.T) {
	if got := connectionTarget(adapter.InboundContext{Destination: M.ParseSocksaddr("example.com:443")}); got != "example.com:443" {
		t.Fatalf("fqdn target = %q", got)
	}
	if got := connectionTarget(adapter.InboundContext{Destination: M.ParseSocksaddr("8.8.8.8:53")}); got != "8.8.8.8:53" {
		t.Fatalf("ip target = %q", got)
	}
	if got := connectionSource(adapter.InboundContext{}); got != "" {
		t.Fatalf("empty source = %q", got)
	}
	if got := connectionInbound(adapter.InboundContext{InboundType: "tun"}); got != "tun" {
		t.Fatalf("inbound type fallback = %q", got)
	}
	if got := connectionProcess(adapter.InboundContext{ProcessInfo: &adapter.ConnectionOwner{UserName: "alice"}}); got != "alice" {
		t.Fatalf("process user = %q", got)
	}
	if got := connectionProcess(adapter.InboundContext{ProcessInfo: &adapter.ConnectionOwner{AndroidPackageNames: []string{"com.app"}}}); got != "com.app" {
		t.Fatalf("process package = %q", got)
	}
	if got := connectionProcess(adapter.InboundContext{}); got != "" {
		t.Fatalf("empty process = %q", got)
	}
}

func TestConnectionTargetFallbacks(t *testing.T) {
	if got := connectionTarget(adapter.InboundContext{
		Domain:      "only.domain",
		Destination: M.Socksaddr{Fqdn: "ignored.example", Port: 0},
	}); got != "only.domain" {
		t.Fatalf("domain without port = %q", got)
	}
	// Invalid destination with Fqdn field set but IsValid false - use zero Socksaddr with Fqdn
	var dest M.Socksaddr
	dest.Fqdn = "bare.fqdn"
	if got := connectionTarget(adapter.InboundContext{Destination: dest}); got != "bare.fqdn" {
		t.Fatalf("fqdn fallback = %q", got)
	}
	if got := connectionTarget(adapter.InboundContext{}); got != "" {
		t.Fatalf("empty target = %q", got)
	}
	if got := connectionProcess(adapter.InboundContext{ProcessInfo: &adapter.ConnectionOwner{}}); got != "" {
		t.Fatalf("empty process info = %q", got)
	}
	if got := ruleName(nil); got != "" {
		t.Fatalf("nil rule = %q", got)
	}
	if got := formatRuleName("default", "rule(default)"); got != "rule(default)" {
		t.Fatalf("format default = %q", got)
	}
	if got := formatRuleName("", " rule "); got != "rule" {
		t.Fatalf("format raw = %q", got)
	}
	if got := formatHostPort(" example.com ", 443); got != "example.com:443" {
		t.Fatalf("format hostport = %q", got)
	}
	if got := formatHostPort("example.com", 0); got != "example.com" {
		t.Fatalf("format host = %q", got)
	}
	if got := formatHostPort("  ", 80); got != "" {
		t.Fatalf("empty host = %q", got)
	}
}

type fakeRule struct {
	typ string
	raw string
}

func (r fakeRule) Match(metadata *adapter.InboundContext) bool { return false }

func (r fakeRule) String() string { return r.raw }

func (r fakeRule) Start() error { return nil }

func (r fakeRule) Close() error { return nil }

func (r fakeRule) Type() string { return r.typ }

func (r fakeRule) Action() adapter.RuleAction { return nil }

func TestRuleNameWithAdapterRule(t *testing.T) {
	if got := ruleName(fakeRule{typ: "logical", raw: "domain_suffix=google.com => proxy"}); got != "domain_suffix=google.com => proxy" {
		t.Fatalf("logical rule = %q", got)
	}
	if got := ruleName(fakeRule{typ: "default", raw: "geoip=cn => direct"}); got != "geoip=cn => direct" {
		t.Fatalf("default rule = %q", got)
	}
	if got := ruleName(fakeRule{typ: "custom", raw: "geoip=cn"}); got != "custom: geoip=cn" {
		t.Fatalf("named rule = %q", got)
	}
	if got := ruleName(fakeRule{typ: "", raw: "fallback-rule"}); got != "fallback-rule" {
		t.Fatalf("raw rule = %q", got)
	}
	if got := ruleName(fakeRule{typ: "default", raw: ""}); got != "default" {
		t.Fatalf("type only = %q", got)
	}
}

func TestFormatRuleNameHelpers(t *testing.T) {
	if got := formatRuleName("default", "  a   b  "); got != "a b" {
		t.Fatalf("compact = %q", got)
	}
	long := strings.Repeat("x", 130)
	if got := formatRuleName("default", long); len([]rune(got)) != 120 || !strings.HasSuffix(got, "…") {
		t.Fatalf("truncate = %q len=%d", got, len([]rune(got)))
	}
	if got := formatRuleName("route", "route"); got != "route" {
		t.Fatalf("equal = %q", got)
	}
	if got := truncateRuleName("ab", 1); got != "a" {
		t.Fatalf("tiny truncate = %q", got)
	}
	if got := truncateRuleName("ab", 0); got != "ab" {
		t.Fatalf("no limit = %q", got)
	}
}

func TestConnectionsSkipsInvalidEntries(t *testing.T) {
	tracker := NewTrafficTracker()
	tracker.connections.Store(int64(99), "not-a-conn")
	if conns := tracker.Connections(); len(conns) != 0 {
		t.Fatalf("expected skip invalid entry, got %#v", conns)
	}
}

func TestConnectionTargetIPOnly(t *testing.T) {
	var dest M.Socksaddr
	dest.Addr = M.ParseSocksaddr("9.9.9.9:53").Addr
	dest.Port = 53
	if got := connectionTarget(adapter.InboundContext{Destination: dest}); got != "9.9.9.9:53" {
		t.Fatalf("ip target = %q", got)
	}
}

func TestCloseConnsWhereNilPred(t *testing.T) {
	tracker := NewTrafficTracker()
	if got := tracker.CloseConnsWhere(nil); got != 0 {
		t.Fatalf("nil pred = %d", got)
	}
}

func TestRoutedConnectionDefaultNetwork(t *testing.T) {
	tracker := NewTrafficTracker()
	left, right := net.Pipe()
	defer func() { _ = right.Close() }()
	wrapped := tracker.RoutedConnection(
		context.Background(),
		left,
		adapter.InboundContext{Destination: M.ParseSocksaddr("example.com:443")},
		nil,
		fakeOutbound{tag: "proxy"},
	)
	conns := tracker.Connections()
	if len(conns) != 1 || conns[0].Network != "tcp" {
		t.Fatalf("connections = %#v", conns)
	}
	_ = wrapped.Close()
}

func TestRoutedPacketConnectionDefaultUDPNetwork(t *testing.T) {
	tracker := NewTrafficTracker()
	got := tracker.RoutedPacketConnection(
		context.Background(),
		&fakePacketConn{},
		adapter.InboundContext{Destination: M.ParseSocksaddr("8.8.8.8:53")},
		nil,
		fakeOutbound{tag: "direct"},
	)
	conns := tracker.Connections()
	if len(conns) != 1 || conns[0].Network != "udp" {
		t.Fatalf("connections = %#v", conns)
	}
	_ = got.Close()
}
