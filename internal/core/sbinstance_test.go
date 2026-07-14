package core

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func TestSBInstanceStatusStopAndDialWhenNotRunning(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))
	if instance == nil {
		t.Fatal("expected instance")
	}

	status := instance.Status()
	if status.Running {
		t.Fatal("new instance should not be running")
	}
	if status.Version == "" {
		t.Fatal("version should not be empty")
	}

	if err := instance.Stop(); err != nil {
		t.Fatal(err)
	}

	_, err := instance.DialOutbound(context.Background(), "proxy", "tcp", "example.com:443")
	if err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("DialOutbound error = %v", err)
	}
}

func TestSBInstanceStartErrors(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))
		if err := instance.Start(); err == nil {
			t.Fatal("expected missing config error")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(configPath, []byte(`{`), 0600); err != nil {
			t.Fatal(err)
		}
		instance := NewSBInstance(configPath, NewLogWriter(5))
		if err := instance.Start(); err == nil {
			t.Fatal("expected invalid config error")
		}
	})
}

func TestSBInstanceRestartPropagatesStartError(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))
	if err := instance.Restart(); err == nil {
		t.Fatal("expected restart error")
	}
}

func TestSBInstanceStatusRunningUptime(t *testing.T) {
	instance := NewSBInstance("config.json", NewLogWriter(5))
	instance.running = true
	instance.startTime = time.Now().Add(-2 * time.Second)

	status := instance.Status()
	if !status.Running {
		t.Fatal("status should report running")
	}
	if status.Uptime == "" {
		t.Fatal("uptime should not be empty")
	}
}

func TestSBInstanceStartStopAndDialWithFactory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	fake := newFakeBox()
	withNewBox(t, func(options box.Options) (boxInstance, error) {
		return fake, nil
	})

	instance := NewSBInstance(configPath, NewLogWriter(5))
	if err := instance.Start(); err != nil {
		t.Fatal(err)
	}
	if !instance.running || instance.Traffic == nil {
		t.Fatalf("instance state running=%v traffic=%v", instance.running, instance.Traffic)
	}
	if !fake.router.trackerAppended {
		t.Fatal("traffic tracker was not registered")
	}

	conn, err := instance.DialOutbound(context.Background(), "proxy", "tcp", "example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	_, err = instance.DialOutbound(context.Background(), "missing", "tcp", "example.com:443")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing outbound error = %v", err)
	}

	if err := instance.Stop(); err != nil {
		t.Fatal(err)
	}
	if !fake.closed || instance.running {
		t.Fatalf("closed=%v running=%v", fake.closed, instance.running)
	}
}

func TestSBInstanceStartFactoryErrors(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	withNewBox(t, func(options box.Options) (boxInstance, error) {
		return nil, errors.New("factory failed")
	})
	instance := NewSBInstance(configPath, NewLogWriter(5))
	if err := instance.Start(); err == nil {
		t.Fatal("expected factory error")
	}

	fake := newFakeBox()
	fake.startErr = errors.New("start failed")
	withNewBox(t, func(options box.Options) (boxInstance, error) {
		return fake, nil
	})
	instance = NewSBInstance(configPath, NewLogWriter(5))
	if err := instance.Start(); err == nil {
		t.Fatal("expected start error")
	}
	if !fake.closed {
		t.Fatal("failed start should close instance")
	}
}

func withNewBox(t *testing.T, factory func(box.Options) (boxInstance, error)) {
	t.Helper()

	previous := newBox
	newBox = factory
	t.Cleanup(func() { newBox = previous })
}

type fakeBox struct {
	router   fakeRouter
	manager  fakeOutboundManager
	startErr error
	closeErr error
	closed   bool
}

func newFakeBox() *fakeBox {
	return &fakeBox{
		manager: fakeOutboundManager{
			outbounds: map[string]adapter.Outbound{
				"proxy": fakeOutbound{tag: "proxy"},
			},
		},
	}
}

func (b *fakeBox) Start() error {
	return b.startErr
}

func (b *fakeBox) Close() error {
	b.closed = true
	return b.closeErr
}

func (b *fakeBox) Router() boxRouter {
	return &b.router
}

func (b *fakeBox) Outbound() boxOutboundManager {
	return &b.manager
}

type fakeRouter struct {
	trackerAppended bool
}

func (r *fakeRouter) AppendTracker(tracker adapter.ConnectionTracker) {
	r.trackerAppended = tracker != nil
}

type fakeOutboundManager struct {
	outbounds map[string]adapter.Outbound
}

func (m *fakeOutboundManager) Outbound(tag string) (adapter.Outbound, bool) {
	outbound, ok := m.outbounds[tag]
	return outbound, ok
}

func (m *fakeOutboundManager) Outbounds() []adapter.Outbound {
	result := make([]adapter.Outbound, 0, len(m.outbounds))
	for _, ob := range m.outbounds {
		result = append(result, ob)
	}
	return result
}

type fakeOutbound struct {
	tag string
}

func (o fakeOutbound) Type() string {
	return "direct"
}

func (o fakeOutbound) Tag() string {
	return o.tag
}

func (o fakeOutbound) Network() []string {
	return []string{"tcp"}
}

func (o fakeOutbound) Dependencies() []string {
	return nil
}

func (o fakeOutbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return fakeNetConn{}, nil
}

func (o fakeOutbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("not implemented")
}

type fakeNetConn struct{}

func (fakeNetConn) Read(b []byte) (int, error) {
	return 0, errors.New("not implemented")
}

func (fakeNetConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func (fakeNetConn) Close() error {
	return nil
}

func (fakeNetConn) LocalAddr() net.Addr {
	return nil
}

func (fakeNetConn) RemoteAddr() net.Addr {
	return nil
}

func (fakeNetConn) SetDeadline(t time.Time) error {
	return nil
}

func (fakeNetConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (fakeNetConn) SetWriteDeadline(t time.Time) error {
	return nil
}

var _ N.Dialer = fakeOutbound{}

// ---- 新增：分组查询、selector 切换、连接关闭 ----

type fakeSelectorOutbound struct {
	tag     string
	outType string
	all     []string
	now     string
}

func (s *fakeSelectorOutbound) Type() string { return s.outType }

func (s *fakeSelectorOutbound) Tag() string { return s.tag }

func (s *fakeSelectorOutbound) Network() []string { return []string{"tcp"} }

func (s *fakeSelectorOutbound) Dependencies() []string { return nil }

func (s *fakeSelectorOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeSelectorOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("not implemented")
}

var _ selectableOutbound = (*fakeSelectorOutbound)(nil)

func (s *fakeSelectorOutbound) Now() string { return s.now }

func (s *fakeSelectorOutbound) All() []string { return s.all }

func (s *fakeSelectorOutbound) SelectOutbound(tag string) bool {
	for _, t := range s.all {
		if t == tag {
			s.now = tag
			return true
		}
	}
	return false
}

func newSelectorFakeBox(selector *fakeSelectorOutbound) *fakeBox {
	return &fakeBox{
		manager: fakeOutboundManager{
			outbounds: map[string]adapter.Outbound{
				selector.tag: selector,
				"direct":     fakeOutbound{tag: "direct"},
			},
		},
	}
}

func startInstanceWithBox(t *testing.T, fake boxInstance) *SBInstance {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	previous := newBox
	newBox = func(box.Options) (boxInstance, error) { return fake, nil }
	t.Cleanup(func() { newBox = previous })
	instance := NewSBInstance(configPath, NewLogWriter(5))
	if err := instance.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = instance.Stop() })
	return instance
}

func TestOutboundGroupsEmptyWhenNotRunning(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "x.json"), NewLogWriter(5))
	if groups := instance.OutboundGroups(); groups != nil {
		t.Fatalf("expected nil, got %#v", groups)
	}
}

func TestOutboundGroupsListsSelectors(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a", "b"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))

	groups := instance.OutboundGroups()
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	g := groups[0]
	if g.Tag != "proxy" || g.Type != "selector" || g.Now != "a" || len(g.All) != 2 || g.All[0] != "a" {
		t.Fatalf("group = %#v", g)
	}
}

func TestSelectOutboundSuccess(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a", "b"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))

	if err := instance.SelectOutbound("proxy", "b"); err != nil {
		t.Fatalf("select: %v", err)
	}
	if selector.now != "b" {
		t.Fatalf("selector.now = %q, want b", selector.now)
	}
}

func TestSelectOutboundNotRunning(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "x.json"), NewLogWriter(5))
	err := instance.SelectOutbound("proxy", "b")
	if err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("err = %v", err)
	}
}

func TestSelectOutboundGroupNotFound(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a", "b"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))
	err := instance.SelectOutbound("missing", "a")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestSelectOutboundNotSelectable(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a", "b"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))
	err := instance.SelectOutbound("direct", "a")
	if err == nil || !strings.Contains(err.Error(), "not a selectable") {
		t.Fatalf("err = %v", err)
	}
}

func TestSelectOutboundTagNotInGroup(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a", "b"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))
	err := instance.SelectOutbound("proxy", "c")
	if err == nil || !strings.Contains(err.Error(), "not in group") {
		t.Fatalf("err = %v", err)
	}
	if selector.now != "a" {
		t.Fatalf("now changed to %q despite invalid select", selector.now)
	}
}

func TestCloseConnectionWiring(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a", "b"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))

	left, right := net.Pipe()
	defer func() { _ = right.Close() }()
	wrapped := instance.Traffic.RoutedConnection(
		context.Background(), left,
		adapter.InboundContext{Destination: M.ParseSocksaddr("example.com:443")},
		nil, fakeOutbound{tag: "proxy"})

	if !instance.CloseConnection(1) {
		t.Fatal("CloseConnection for existing id should return true")
	}
	if instance.CloseConnection(999) {
		t.Fatal("CloseConnection for missing id should return false")
	}
	buf := make([]byte, 1)
	if _, err := right.Read(buf); err == nil {
		t.Fatal("right should error after close")
	}
	_ = wrapped.Close()
}

func TestCloseAllConnectionsWiring(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a", "b"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))

	l1, r1 := net.Pipe()
	l2, r2 := net.Pipe()
	w1 := instance.Traffic.RoutedConnection(context.Background(), l1, adapter.InboundContext{Destination: M.ParseSocksaddr("a:443")}, nil, fakeOutbound{tag: "proxy"})
	w2 := instance.Traffic.RoutedConnection(context.Background(), l2, adapter.InboundContext{Destination: M.ParseSocksaddr("b:443")}, nil, fakeOutbound{tag: "proxy"})

	if n := instance.CloseAllConnections(); n != 2 {
		t.Fatalf("CloseAllConnections = %d, want 2", n)
	}
	if n := instance.CloseAllConnections(); n != 0 {
		t.Fatalf("empty CloseAllConnections = %d, want 0", n)
	}
	_ = r1.Close()
	_ = r2.Close()
	_ = w1.Close()
	_ = w2.Close()
}

func TestCloseConnectionsOnStoppedInstance(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "x.json"), NewLogWriter(5))
	if instance.CloseConnection(1) {
		t.Fatal("CloseConnection on stopped instance should return false")
	}
	if n := instance.CloseAllConnections(); n != 0 {
		t.Fatalf("CloseAllConnections = %d, want 0", n)
	}
}

// ---- T5: URLTest 分组延迟 ----

type fakeURLTestOutbound struct {
	tag     string
	all     []string
	now     string
	delays  map[string]uint16
	testErr error
}

func (u *fakeURLTestOutbound) Type() string { return "urltest" }

func (u *fakeURLTestOutbound) Tag() string { return u.tag }

func (u *fakeURLTestOutbound) Network() []string { return []string{"tcp"} }

func (u *fakeURLTestOutbound) Dependencies() []string { return nil }

func (u *fakeURLTestOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (u *fakeURLTestOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("not implemented")
}

func (u *fakeURLTestOutbound) Now() string { return u.now }

func (u *fakeURLTestOutbound) All() []string { return u.all }

func (u *fakeURLTestOutbound) SelectOutbound(string) bool { return false }

func (u *fakeURLTestOutbound) URLTest(ctx context.Context) (map[string]uint16, error) {
	return u.delays, u.testErr
}

func newURLTestFakeBox(ut *fakeURLTestOutbound) *fakeBox {
	return &fakeBox{
		manager: fakeOutboundManager{
			outbounds: map[string]adapter.Outbound{
				ut.tag: ut,
			},
		},
	}
}

func TestURLTestDelaysSuccess(t *testing.T) {
	ut := &fakeURLTestOutbound{
		tag:    "auto",
		all:    []string{"a", "b"},
		now:    "a",
		delays: map[string]uint16{"a": 120, "b": 250},
	}
	instance := startInstanceWithBox(t, newURLTestFakeBox(ut))

	delays, err := instance.URLTestDelays(context.Background(), "auto")
	if err != nil {
		t.Fatalf("URLTestDelays: %v", err)
	}
	if delays["a"] != 120 || delays["b"] != 250 {
		t.Fatalf("delays = %v", delays)
	}
}

func TestURLTestDelaysNotRunning(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "x.json"), NewLogWriter(5))
	_, err := instance.URLTestDelays(context.Background(), "auto")
	if err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("err = %v", err)
	}
}

func TestURLTestDelaysGroupNotFound(t *testing.T) {
	ut := &fakeURLTestOutbound{tag: "auto", all: []string{"a"}}
	instance := startInstanceWithBox(t, newURLTestFakeBox(ut))
	_, err := instance.URLTestDelays(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestURLTestDelaysNotURLTest(t *testing.T) {
	selector := &fakeSelectorOutbound{tag: "proxy", outType: "selector", all: []string{"a"}, now: "a"}
	instance := startInstanceWithBox(t, newSelectorFakeBox(selector))
	_, err := instance.URLTestDelays(context.Background(), "proxy")
	if err == nil || !strings.Contains(err.Error(), "not a urltest") {
		t.Fatalf("err = %v", err)
	}
}

func TestURLTestDelaysInnerError(t *testing.T) {
	ut := &fakeURLTestOutbound{tag: "auto", all: []string{"a"}, testErr: errors.New("boom")}
	instance := startInstanceWithBox(t, newURLTestFakeBox(ut))
	_, err := instance.URLTestDelays(context.Background(), "auto")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}
