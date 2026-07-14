package core

import (
	"bytes"
	"context"
	"net"
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
	if len(conns) != 1 || conns[0].Target != "example.com" || conns[0].Outbound != "proxy" {
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
	packetConn := fakePacketConn{}
	got := tracker.RoutedPacketConnection(
		context.Background(),
		packetConn,
		adapter.InboundContext{},
		nil,
		fakeOutbound{tag: "proxy"},
	)
	if got != packetConn {
		t.Fatal("packet connection should be returned unchanged")
	}
}

type fakePacketConn struct{}

func (fakePacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	return M.Socksaddr{}, nil
}

func (fakePacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	return nil
}

func (fakePacketConn) Close() error {
	return nil
}

func (fakePacketConn) LocalAddr() net.Addr {
	return nil
}

func (fakePacketConn) SetDeadline(t time.Time) error {
	return nil
}

func (fakePacketConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (fakePacketConn) SetWriteDeadline(t time.Time) error {
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
