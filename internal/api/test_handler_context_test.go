package api

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
	"github.com/xuthus5/boxd/internal/service"
)

type testContextKey struct{}

type blockingTestDialer struct {
	release chan struct{}
	started chan struct{}
	starts  atomic.Int32
}

func newBlockingTestDialer(capacity int) *blockingTestDialer {
	return &blockingTestDialer{
		release: make(chan struct{}),
		started: make(chan struct{}, capacity),
	}
}

func (d *blockingTestDialer) DialOutbound(ctx context.Context, _, _, _ string) (net.Conn, error) {
	return nil, d.wait(ctx)
}

func (d *blockingTestDialer) OutboundDelay(
	ctx context.Context,
	_ string,
	_ string,
	_ time.Duration,
) (uint16, error) {
	return 0, d.wait(ctx)
}

func (d *blockingTestDialer) wait(ctx context.Context) error {
	d.starts.Add(1)
	d.started <- struct{}{}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.release:
		return errors.New("test probe released")
	}
}

type recordingTestDialer struct {
	dialContext  chan context.Context
	delayContext chan context.Context
}

func (d *recordingTestDialer) DialOutbound(ctx context.Context, _, _, _ string) (net.Conn, error) {
	d.dialContext <- ctx
	return newHTTPResponseConn(), nil
}

func (d *recordingTestDialer) OutboundDelay(
	ctx context.Context,
	_ string,
	_ string,
	_ time.Duration,
) (uint16, error) {
	d.delayContext <- ctx
	return 12, nil
}

func TestTestHandlerRunStopsAfterCancellation(t *testing.T) {
	nodeManager, _, _, _ := newAPIManagers(t)
	dialer := newBlockingTestDialer(1)
	t.Cleanup(func() { close(dialer.release) })
	handler := NewTestHandler(func() string { return "" }, nodeManager, dialer)
	ctx, cancel := context.WithCancel(t.Context())
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/nodes/test",
		strings.NewReader(`{"tag":"node-a","test_type":"tcp"}`),
	).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.Run(recorder, request)
		close(done)
	}()

	waitForTestSignal(t, dialer.started)
	cancel()
	waitForTestSignal(t, done)
	assertCanceledTestRequest(t, recorder, nodeManager)
}

func TestTestHandlerRunHandlesMissingNodeManager(t *testing.T) {
	handler := NewTestHandler(func() string { return "" }, nil, &fakeDialer{delay: 12})
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/nodes/test",
		strings.NewReader(`{"tag":"node-a","test_type":"tcp"}`),
	)
	recorder := httptest.NewRecorder()

	handler.Run(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestTestHandlerRunBatchLimitsCanceledWork(t *testing.T) {
	nodeManager, _, _, _ := newAPIManagers(t)
	dialer := newBlockingTestDialer(8)
	t.Cleanup(func() { close(dialer.release) })
	handler := NewTestHandler(func() string { return "" }, nodeManager, dialer)
	ctx, cancel := context.WithCancel(t.Context())
	body := `{"items":[
		{"tag":"a","test_type":"tcp"},{"tag":"b","test_type":"tcp"},
		{"tag":"c","test_type":"tcp"},{"tag":"d","test_type":"tcp"},
		{"tag":"e","test_type":"tcp"},{"tag":"f","test_type":"tcp"},
		{"tag":"g","test_type":"tcp"},{"tag":"h","test_type":"tcp"}
	],"concurrency":2}`
	request := httptest.NewRequest(http.MethodPost, "/api/nodes/test-batch", strings.NewReader(body)).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.RunBatch(recorder, request)
		close(done)
	}()

	waitForTestSignal(t, dialer.started)
	waitForTestSignal(t, dialer.started)
	cancel()
	waitForTestSignal(t, done)
	if got := dialer.starts.Load(); got != 2 {
		t.Fatalf("started probes = %d, want 2", got)
	}
	assertCanceledTestRequest(t, recorder, nodeManager)
}

func TestTestHandlerProbeContextPropagation(t *testing.T) {
	ctx := context.WithValue(t.Context(), testContextKey{}, "request")
	dialer := &recordingTestDialer{
		dialContext:  make(chan context.Context, 1),
		delayContext: make(chan context.Context, 1),
	}
	handler := NewTestHandler(func() string { return "http://example.test/" }, nil, dialer)

	if result := handler.tcpPing(ctx, TestRequest{Tag: "proxy"}); !result.Success {
		t.Fatalf("tcp result = %#v", result)
	}
	assertTestContextValue(t, <-dialer.delayContext)

	if result := handler.httpTest(ctx, TestRequest{Tag: "proxy"}); !result.Success {
		t.Fatalf("http result = %#v", result)
	}
	assertTestContextValue(t, <-dialer.dialContext)

	previous := service.ICMPEcho
	t.Cleanup(func() { service.ICMPEcho = previous })
	var commandContext context.Context
	service.ICMPEcho = func(ctx context.Context, _ string) (float64, error) {
		commandContext = ctx
		return 3.5, nil
	}
	if result := handler.icmpPing(ctx, TestRequest{Server: "1.1.1.1"}); !result.Success {
		t.Fatalf("icmp result = %#v", result)
	}
	assertTestContextValue(t, commandContext)
}

func waitForTestSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for test signal")
	}
}

func assertCanceledTestRequest(t *testing.T, recorder *httptest.ResponseRecorder, nodeManager *core.NodeManager) {
	t.Helper()
	if recorder.Body.Len() != 0 {
		t.Fatalf("response body = %q, want empty", recorder.Body.String())
	}
	if results := nodeManager.GetAllTestResults(); len(results) != 0 {
		t.Fatalf("persisted results = %#v, want empty", results)
	}
}

func assertTestContextValue(t *testing.T, ctx context.Context) {
	t.Helper()
	if ctx == nil || ctx.Value(testContextKey{}) != "request" {
		t.Fatalf("probe context value = %v, want request", ctx)
	}
}
