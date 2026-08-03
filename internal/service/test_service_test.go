package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type fakeDialer struct {
	delay      uint16
	delayErr   error
	dialErr    error
	delayCalls atomic.Int32
}

func (f *fakeDialer) DialOutbound(context.Context, string, string, string) (net.Conn, error) {
	if f.dialErr != nil {
		return nil, f.dialErr
	}
	return &fakeNetConn{}, nil
}

func (f *fakeDialer) OutboundDelay(context.Context, string, string, time.Duration) (uint16, error) {
	f.delayCalls.Add(1)
	return f.delay, f.delayErr
}

type fakeNetConn struct{}

func (fakeNetConn) Read([]byte) (int, error)         { return 0, errors.New("eof") }
func (fakeNetConn) Write([]byte) (int, error)        { return 0, errors.New("eof") }
func (fakeNetConn) Close() error                     { return nil }
func (fakeNetConn) LocalAddr() net.Addr              { return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1} }
func (fakeNetConn) RemoteAddr() net.Addr             { return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 2} }
func (fakeNetConn) SetDeadline(time.Time) error      { return nil }
func (fakeNetConn) SetReadDeadline(time.Time) error  { return nil }
func (fakeNetConn) SetWriteDeadline(time.Time) error { return nil }

func TestTestServiceRun(t *testing.T) {
	nodeMgr, db := newTestManagers(t)
	svc := NewTestService(func() string { return "" }, nodeMgr, &fakeDialer{delay: 100})
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "tcp", Server: "1.1.1.1", Port: 443})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	_ = db
}

func TestTestServiceRunMissingFields(t *testing.T) {
	svc := NewTestService(nil, nil, nil)
	_, err := svc.Run(context.Background(), TestRequest{})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestTestServiceRunUnsupportedType(t *testing.T) {
	svc := NewTestService(nil, nil, nil)
	_, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "udp"})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestTestServiceRunDelayFailure(t *testing.T) {
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(nil, nodeMgr, &fakeDialer{delayErr: errors.New("down")})
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "tcp"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure")
	}
}

func TestTestServiceRunNilInstance(t *testing.T) {
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(nil, nodeMgr, nil)
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "tcp"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for nil instance")
	}
}

func TestTestServiceRunPersistFailure(t *testing.T) {
	svc := NewTestService(nil, nil, &fakeDialer{delay: 50})
	_, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "tcp"})
	if err == nil {
		t.Fatal("expected persist error for nil node manager")
	}
}

func TestTestServiceRunHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	addr := strings.TrimPrefix(server.URL, "http://")
	nodeMgr, _ := newTestManagers(t)
	dialer := &netDialer{addr: addr}
	svc := NewTestService(func() string { return server.URL }, nodeMgr, dialer)
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "http"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
}

type netDialer struct {
	addr string
}

func (d *netDialer) DialOutbound(_ context.Context, _, _, _ string) (net.Conn, error) {
	return net.Dial("tcp", d.addr)
}

func (d *netDialer) OutboundDelay(context.Context, string, string, time.Duration) (uint16, error) {
	return 0, errors.New("not used")
}

func TestTestServiceRunHTTPInvalidURL(t *testing.T) {
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(func() string { return "not a url" }, nodeMgr, &fakeDialer{})
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "http"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for invalid url")
	}
}

func TestTestServiceRunICMP(t *testing.T) {
	original := ICMPEcho
	t.Cleanup(func() { ICMPEcho = original })
	ICMPEcho = func(_ context.Context, _ string) (float64, error) {
		return 12.3, nil
	}
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(nil, nodeMgr, &fakeDialer{})
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "icmp", Server: "1.1.1.1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	if result.LatencyMs != 12.3 {
		t.Fatalf("latency = %v", result.LatencyMs)
	}
}

func TestTestServiceRunICMPInvalidServer(t *testing.T) {
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(nil, nodeMgr, &fakeDialer{})
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "icmp", Server: "bad;server"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for invalid server")
	}
}

func TestTestServiceRunICMPPingFailure(t *testing.T) {
	originalEcho := ICMPEcho
	originalCmd := PingCommandOutput
	t.Cleanup(func() {
		ICMPEcho = originalEcho
		PingCommandOutput = originalCmd
	})
	// icmp 库失败（如无原始 socket 权限），回退到 ping 命令也失败。
	ICMPEcho = func(_ context.Context, _ string) (float64, error) {
		return 0, errors.New("no permission")
	}
	PingCommandOutput = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("ping: unreachable"), errors.New("exit status 1")
	}
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(nil, nodeMgr, &fakeDialer{})
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "icmp", Server: "1.1.1.1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure")
	}
}

func TestTestServiceBatch(t *testing.T) {
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(func() string { return "" }, nodeMgr, &fakeDialer{delay: 10})
	results, err := svc.RunBatch(context.Background(), TestBatchRequest{
		Items: []TestRequest{
			{Tag: "a", TestType: "tcp"},
			{Tag: "b", TestType: "tcp"},
		},
		Concurrency: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d", len(results))
	}
}

func TestTestServiceBatchEmpty(t *testing.T) {
	svc := NewTestService(nil, nil, nil)
	_, err := svc.RunBatch(context.Background(), TestBatchRequest{})
	if err == nil {
		t.Fatal("expected error for empty items")
	}
}

func TestTestServiceBatchTooMany(t *testing.T) {
	items := make([]TestRequest, maxBatchItems+1)
	svc := NewTestService(nil, nil, nil)
	_, err := svc.RunBatch(context.Background(), TestBatchRequest{Items: items})
	if err == nil {
		t.Fatal("expected error for too many items")
	}
}

func TestTestServiceBatchCancellation(t *testing.T) {
	original := commandOutput
	t.Cleanup(func() { commandOutput = original })
	nodeMgr, _ := newTestManagers(t)
	svc := NewTestService(nil, nodeMgr, &fakeDialer{delayErr: errors.New("down")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.RunBatch(ctx, TestBatchRequest{
		Items: []TestRequest{{Tag: "a", TestType: "tcp"}},
	})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestTestServiceBatchWithNilManager(t *testing.T) {
	svc := NewTestService(nil, nil, &fakeDialer{delay: 10})
	results, err := svc.RunBatch(context.Background(), TestBatchRequest{
		Items:       []TestRequest{{Tag: "a", TestType: "tcp"}},
		Concurrency: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d", len(results))
	}
}

func TestTestServiceRunHTTPDialError(t *testing.T) {
	nodeMgr, _ := newTestManagers(t)
	dialer := &fakeDialer{dialErr: errors.New("dial failed")}
	svc := NewTestService(func() string { return "http://example.com/" }, nodeMgr, dialer)
	result, err := svc.Run(context.Background(), TestRequest{Tag: "n", TestType: "http"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for dial error")
	}
}

func TestNonEmpty(t *testing.T) {
	if got := nonEmpty("x", "fallback"); got != "x" {
		t.Fatalf("got %q", got)
	}
	if got := nonEmpty("", "fallback"); got != "fallback" {
		t.Fatalf("got %q", got)
	}
}

func TestTestServiceListResultsAndHistory(t *testing.T) {
	nodeMgr, _ := newTestManagers(t)
	if err := nodeMgr.SaveTestResult("n_test", model.TestResult{Tag: "n", TestType: "tcp", Success: true}); err != nil {
		t.Fatal(err)
	}
	svc := NewTestService(nil, nodeMgr, nil)
	all, err := svc.ListResults(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("no results")
	}
	history, err := svc.ListHistory(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := history["history"]; !ok {
		t.Fatalf("history = %v", history)
	}
	tagged, err := svc.ListHistory(context.Background(), "n")
	if err != nil {
		t.Fatal(err)
	}
	if tagged["tag"] != "n" {
		t.Fatalf("tagged = %v", tagged)
	}
}

func TestTestServiceListResultsNilManager(t *testing.T) {
	svc := NewTestService(nil, nil, nil)
	if _, err := svc.ListResults(context.Background()); err == nil {
		t.Fatal("expected error for nil manager")
	}
}

func TestIsValidPingTarget(t *testing.T) {
	tests := map[string]bool{
		"1.1.1.1":                true,
		"example.com":            true,
		"bad;server":             false,
		"bad server":             false,
		strings.Repeat("a", 300): false,
		"":                       false,
		"no-dot":                 false,
		"-bad.example":           true,
	}
	for server, want := range tests {
		if got := isValidPingTarget(server); got != want {
			t.Fatalf("%q = %v, want %v", server, got, want)
		}
	}
}

func TestParsePingLatency(t *testing.T) {
	if got, ok := parsePingLatency("64 bytes: time=12.3 ms"); !ok || got != 12.3 {
		t.Fatalf("got %v %v", got, ok)
	}
	if _, ok := parsePingLatency("no match"); ok {
		t.Fatal("unexpected match")
	}
	if _, ok := parsePingLatency("time=abc ms"); ok {
		t.Fatal("unexpected match for non-numeric")
	}
	if _, ok := parsePingLatency("time=-5 ms"); ok {
		t.Fatal("unexpected match for negative")
	}
}

func newTestManagers(t *testing.T) (*core.NodeManager, *core.SubscriptionManager) {
	t.Helper()
	db := newTestDB(t)
	return core.NewNodeManager(db), core.NewSubscriptionManager(db, t.TempDir())
}
