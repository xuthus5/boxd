package api

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	sbLog "github.com/sagernet/sing-box/log"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

func TestServiceHandlerStatusAndErrors(t *testing.T) {
	instance := core.NewSBInstance(t.TempDir()+"/missing.json", core.NewLogWriter(5))
	handler := NewServiceHandler(instance)

	rr := httptest.NewRecorder()
	handler.Status(rr, httptest.NewRequest(http.MethodGet, "/api/service/status", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.Stop(rr, httptest.NewRequest(http.MethodPost, "/api/service/stop", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("stop status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.Start(rr, httptest.NewRequest(http.MethodPost, "/api/service/start", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("start status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.Restart(rr, httptest.NewRequest(http.MethodPost, "/api/service/restart", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("restart status = %d", rr.Code)
	}
}

func TestServiceHandlerSuccess(t *testing.T) {
	service := &fakeService{}
	handler := NewServiceHandler(service)

	tests := []struct {
		name   string
		call   func(*httptest.ResponseRecorder)
		status string
	}{
		{name: "start", call: func(rr *httptest.ResponseRecorder) {
			handler.Start(rr, httptest.NewRequest(http.MethodPost, "/start", nil))
		}, status: "started"},
		{name: "stop", call: func(rr *httptest.ResponseRecorder) {
			handler.Stop(rr, httptest.NewRequest(http.MethodPost, "/stop", nil))
		}, status: "stopped"},
		{name: "restart", call: func(rr *httptest.ResponseRecorder) {
			handler.Restart(rr, httptest.NewRequest(http.MethodPost, "/restart", nil))
		}, status: "restarted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.call(rr)
			if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), tt.status) {
				t.Fatalf("status/body = %d %s", rr.Code, rr.Body.String())
			}
		})
	}

	rr := httptest.NewRecorder()
	handler.Status(rr, httptest.NewRequest(http.MethodGet, "/status", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"running":true`) {
		t.Fatalf("status body = %d %s", rr.Code, rr.Body.String())
	}

	service.stopErr = errors.New("stop failed")
	rr = httptest.NewRecorder()
	handler.Stop(rr, httptest.NewRequest(http.MethodPost, "/stop", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("stop error status = %d", rr.Code)
	}
}

func TestTestHandlerRunAndListResults(t *testing.T) {
	nodeMgr, _, _, _ := newAPIManagers(t)
	handler := NewTestHandler(func() string { return "" }, nodeMgr, nil)

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "invalid json", body: `{`, want: http.StatusBadRequest},
		{name: "missing fields", body: `{}`, want: http.StatusBadRequest},
		{name: "unsupported type", body: `{"tag":"n","test_type":"udp"}`, want: http.StatusBadRequest},
		{name: "http without instance", body: `{"tag":"n","test_type":"http","server":"http://example.test"}`, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.Run(rr, jsonRequest(http.MethodPost, "/api/nodes/test", tt.body))
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d", rr.Code, tt.want)
			}
		})
	}

	rr := httptest.NewRecorder()
	handler.ListResults(rr, httptest.NewRequest(http.MethodGet, "/api/nodes/test-results", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "n_http") {
		t.Fatalf("results status/body = %d %s", rr.Code, rr.Body.String())
	}

	result := handler.icmpPing(TestRequest{Server: "bad;host"})
	if result.Error != "invalid server address" {
		t.Fatalf("icmp error = %q", result.Error)
	}
}

func TestTestHandlerTCPPing(t *testing.T) {
	dialer := &fakeDialer{delay: 187}
	handler := NewTestHandler(func() string { return "https://example.com/ping" }, nil, dialer)
	result := handler.tcpPing(TestRequest{Tag: "proxy", Server: "example.com", Port: 443})
	if !result.Success || result.Error != "" || result.LatencyMs != 187 {
		t.Fatalf("tcp result = %#v", result)
	}
	delayTag, delayLink := dialer.lastDelayInput()
	if delayTag != "proxy" || delayLink != "https://example.com/ping" {
		t.Fatalf("delay input = %q %q", delayTag, delayLink)
	}

	dialer.delayErr = errors.New("delay failed")
	result = handler.tcpPing(TestRequest{Server: "example.com", Port: 443})
	if result.Error == "" {
		t.Fatalf("expected delay error, got %#v", result)
	}

	dialer.delayErr = nil
	dialer.delay = 0
	if result = handler.tcpPing(TestRequest{Tag: "proxy"}); result.Error == "" {
		t.Fatalf("expected zero delay error, got %#v", result)
	}
	if result = NewTestHandler(nil, nil, nil).tcpPing(TestRequest{Tag: "proxy"}); result.Error == "" {
		t.Fatalf("expected unavailable error, got %#v", result)
	}
}

func TestTestHandlerHTTPTestWithDialer(t *testing.T) {
	handler := NewTestHandler(
		func() string { return "http://example.test/" },
		nil,
		&fakeDialer{connFactory: func() net.Conn { return newHTTPResponseConn() }},
	)
	result := handler.httpTest(TestRequest{Tag: "proxy", Server: "http://ignored.test/"})
	if !result.Success || result.Error != "" {
		t.Fatalf("http result = %#v", result)
	}

	handler = NewTestHandler(
		func() string { return "http://example.test/" },
		nil,
		&fakeDialer{err: errors.New("dial failed")},
	)
	result = handler.httpTest(TestRequest{Tag: "proxy"})
	if result.Error == "" {
		t.Fatalf("expected dial error, got %#v", result)
	}
}

func TestTestHandlerICMPPingAndLatencyParsing(t *testing.T) {
	previous := commandOutput
	t.Cleanup(func() { commandOutput = previous })

	handler := NewTestHandler(nil, nil, nil)
	commandOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("64 bytes from 1.1.1.1: icmp_seq=1 ttl=57 time=12.34 ms\n"), nil
	}
	result := handler.icmpPing(TestRequest{Server: "1.1.1.1"})
	if !result.Success || result.LatencyMs != 12.34 {
		t.Fatalf("icmp result = %#v", result)
	}

	commandOutput = func(name string, args ...string) ([]byte, error) {
		return []byte("permission denied"), errors.New("failed")
	}
	result = handler.icmpPing(TestRequest{Server: "1.1.1.1"})
	if result.Error == "" {
		t.Fatalf("expected ping error, got %#v", result)
	}

	if ms, ok := parsePingLatency("time=7.5ms"); !ok || ms != 7.5 {
		t.Fatalf("latency = %v,%v", ms, ok)
	}
	if _, ok := parsePingLatency("no latency"); ok {
		t.Fatal("expected parse miss")
	}
	if _, ok := parsePingLatency("time=0ms"); ok {
		t.Fatal("expected zero latency to be ignored")
	}
	if _, ok := parsePingLatency("time=not-a-number ms"); ok {
		t.Fatal("expected invalid latency to be ignored")
	}
}

func TestStatsHandlerHelpersAndSSE(t *testing.T) {
	handler := NewStatsHandler(nil, nil, nil)
	t.Cleanup(handler.stopTrafficSampler)
	up, down := handler.getTraffic()
	if up != 0 || down != 0 {
		t.Fatalf("traffic = %d,%d", up, down)
	}
	count, conns := handler.getConns()
	if count != 0 || conns != nil {
		t.Fatalf("connections = %d,%v", count, conns)
	}

	rr := httptest.NewRecorder()
	writeSSE(rr, map[string]string{"message": "ok"})
	if !strings.Contains(rr.Body.String(), `data: {"message":"ok"}`) {
		t.Fatalf("sse body = %q", rr.Body.String())
	}
	writeSSE(rr, map[string]any{"bad": func() {}})

	req := canceledRequest(http.MethodGet, "/api/stats/traffic")
	rr = httptest.NewRecorder()
	handler.TrafficSSE(rr, req)
	if rr.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("traffic content type = %q", rr.Header().Get("Content-Type"))
	}

	noFlush := newNoFlushWriter()
	handler.TrafficSSE(noFlush, httptest.NewRequest(http.MethodGet, "/api/stats/traffic", nil))
	if noFlush.status != http.StatusInternalServerError {
		t.Fatalf("no flush status = %d", noFlush.status)
	}

	noFlush = newNoFlushWriter()
	handler.LogsSSE(noFlush, httptest.NewRequest(http.MethodGet, "/api/stats/logs", nil))
	if noFlush.status != http.StatusInternalServerError {
		t.Fatalf("logs no flush status = %d", noFlush.status)
	}

	noFlush = newNoFlushWriter()
	handler.ConnectionsSSE(noFlush, httptest.NewRequest(http.MethodGet, "/api/stats/connections", nil))
	if noFlush.status != http.StatusInternalServerError {
		t.Fatalf("connections no flush status = %d", noFlush.status)
	}

	logWriter := core.NewLogWriter(5)
	logWriter.WriteMessage(sbLog.LevelInfo, "booted")
	logHandler := NewStatsHandler(logWriter, logWriter, nil)
	t.Cleanup(logHandler.stopTrafficSampler)
	rr = httptest.NewRecorder()
	logHandler.LogsSSE(rr, canceledRequest(http.MethodGet, "/api/stats/logs"))
	if !strings.Contains(rr.Body.String(), "booted") {
		t.Fatalf("logs body = %q", rr.Body.String())
	}
	rr = httptest.NewRecorder()
	logHandler.AppLogsSSE(rr, canceledRequest(http.MethodGet, "/api/stats/app-logs"))
	if !strings.Contains(rr.Body.String(), "booted") {
		t.Fatalf("app logs body = %q", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	handler.LogsSSE(rr, canceledRequest(http.MethodGet, "/api/stats/logs"))
	if rr.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("logs content type = %q", rr.Header().Get("Content-Type"))
	}

	rr = httptest.NewRecorder()
	handler.ConnectionsSSE(rr, canceledRequest(http.MethodGet, "/api/stats/connections"))
	if rr.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("connections content type = %q", rr.Header().Get("Content-Type"))
	}
}

func TestNetworkHandlerGetInterfaces(t *testing.T) {
	handler := NewNetworkHandler()
	rr := httptest.NewRecorder()
	handler.GetInterfaces(rr, httptest.NewRequest(http.MethodGet, "/api/network/interfaces", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"auto"`) || !strings.Contains(rr.Body.String(), `"0.0.0.0"`) {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestGetDefaultRouteIPsErrors(t *testing.T) {
	previous := defaultRouteOutput
	previousAddrs := interfaceAddrs
	t.Cleanup(func() {
		defaultRouteOutput = previous
		interfaceAddrs = previousAddrs
	})

	defaultRouteOutput = func() ([]byte, error) {
		return nil, errors.New("ip command failed")
	}
	if ips := getDefaultRouteIPs(); ips != nil {
		t.Fatalf("ips = %#v, want nil", ips)
	}

	defaultRouteOutput = func() ([]byte, error) {
		return []byte("default via 10.0.0.1\n"), nil
	}
	if ips := getDefaultRouteIPs(); ips != nil {
		t.Fatalf("ips = %#v, want nil for malformed route", ips)
	}

	defaultRouteOutput = func() ([]byte, error) {
		return []byte("default via 10.0.0.1 dev eth-test proto dhcp\n"), nil
	}
	interfaceAddrs = func(name string) ([]net.Addr, error) {
		if name != "eth-test" {
			t.Fatalf("interface name = %q", name)
		}
		ip, ipNet, err := net.ParseCIDR("192.0.2.10/24")
		if err != nil {
			t.Fatal(err)
		}
		ipNet.IP = ip
		_, loopback, err := net.ParseCIDR("::1/128")
		if err != nil {
			t.Fatal(err)
		}
		return []net.Addr{ipNet, loopback}, nil
	}
	ips := getDefaultRouteIPs()
	if len(ips) != 1 || ips[0] != "192.0.2.10" {
		t.Fatalf("ips = %#v", ips)
	}
}

func canceledRequest(method, target string) *http.Request {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return httptest.NewRequest(method, target, nil).WithContext(ctx)
}

type noFlushWriter struct {
	header http.Header
	status int
	body   strings.Builder
}

func newNoFlushWriter() *noFlushWriter {
	return &noFlushWriter{header: http.Header{}, status: http.StatusOK}
}

func (w *noFlushWriter) Header() http.Header {
	return w.header
}

func (w *noFlushWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *noFlushWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func TestStatsSSEWithTrafficTracker(t *testing.T) {
	tracker := core.NewTrafficTracker()
	instance := &core.SBInstance{Traffic: tracker}
	handler := NewStatsHandler(nil, nil, instance)
	t.Cleanup(handler.stopTrafficSampler)

	up, down := handler.getTraffic()
	if up != 0 || down != 0 {
		t.Fatalf("traffic = %d,%d", up, down)
	}
	count, conns := handler.getConns()
	if count != 0 || len(conns) != 0 {
		t.Fatalf("connections = %d,%v", count, conns)
	}
}

func TestTrafficHistoryBufferOrderAndTrim(t *testing.T) {
	buffer := newTrafficHistoryBuffer(3)

	t0 := time.Date(2026, 7, 6, 20, 0, 0, 0, time.UTC)
	buffer.add(TrafficHistoryPoint{Timestamp: t0, UploadBytes: 1, DownloadBytes: 10})
	buffer.add(TrafficHistoryPoint{Timestamp: t0.Add(time.Second), UploadBytes: 2, DownloadBytes: 20})

	initial := buffer.snapshot()
	if len(initial) != 2 {
		t.Fatalf("initial len = %d", len(initial))
	}
	if !initial[0].Timestamp.Equal(t0) || !initial[1].Timestamp.Equal(t0.Add(time.Second)) {
		t.Fatalf("initial order = %#v", initial)
	}

	buffer.add(TrafficHistoryPoint{Timestamp: t0.Add(2 * time.Second), UploadBytes: 3, DownloadBytes: 30})
	buffer.add(TrafficHistoryPoint{Timestamp: t0.Add(3 * time.Second), UploadBytes: 4, DownloadBytes: 40})

	points := buffer.snapshot()
	if len(points) != 3 {
		t.Fatalf("points len = %d", len(points))
	}
	if !points[0].Timestamp.Equal(t0.Add(time.Second)) || !points[2].Timestamp.Equal(t0.Add(3*time.Second)) {
		t.Fatalf("points order = %#v", points)
	}
	if points[0].UploadBytes != 2 || points[2].DownloadBytes != 40 {
		t.Fatalf("points values = %#v", points)
	}
}

func TestStatsHandlerTrafficHistory(t *testing.T) {
	handler := NewStatsHandler(nil, nil, nil)
	t.Cleanup(handler.stopTrafficSampler)
	handler.history = newTrafficHistoryBuffer(4)

	base := time.Date(2026, 7, 6, 20, 0, 0, 0, time.UTC)
	handler.history.add(TrafficHistoryPoint{Timestamp: base, UploadBytes: 100, DownloadBytes: 200})
	handler.history.add(TrafficHistoryPoint{Timestamp: base.Add(time.Second), UploadBytes: 140, DownloadBytes: 280})

	rr := httptest.NewRecorder()
	handler.TrafficHistory(rr, httptest.NewRequest(http.MethodGet, "/api/stats/traffic/history", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}

	resp := decodeBody[struct {
		Points []TrafficHistoryPoint `json:"points"`
	}](t, rr)
	if len(resp.Points) != 2 {
		t.Fatalf("points len = %d", len(resp.Points))
	}
	if !resp.Points[0].Timestamp.Equal(base) || resp.Points[1].UploadBytes != 140 {
		t.Fatalf("points = %#v", resp.Points)
	}
}

func TestCanceledRequestIsAlreadyDone(t *testing.T) {
	req := canceledRequest(http.MethodGet, "/x")
	select {
	case <-req.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("context was not canceled")
	}
}

type fakeDialer struct {
	mu          sync.Mutex
	conn        net.Conn
	connFactory func() net.Conn
	err         error
	delay       uint16
	delayErr    error
	delayTag    string
	delayLink   string
}

type fakeService struct {
	stopErr error
}

func (s *fakeService) Start() error {
	return nil
}

func (s *fakeService) Stop() error {
	return s.stopErr
}

func (s *fakeService) Restart() error {
	return nil
}

func (s *fakeService) Status() model.ServiceStatus {
	return model.ServiceStatus{Running: true, Version: "test"}
}

func (d *fakeDialer) DialOutbound(ctx context.Context, tag, network, addr string) (net.Conn, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.connFactory != nil {
		return d.connFactory(), nil
	}
	return d.conn, nil
}

func (d *fakeDialer) OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.delayTag = tag
	d.delayLink = link
	return d.delay, d.delayErr
}

func (d *fakeDialer) lastDelayInput() (string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.delayTag, d.delayLink
}

func newHTTPResponseConn() net.Conn {
	client, server := net.Pipe()
	go func() {
		defer func() { _ = server.Close() }()
		buffer := make([]byte, 4096)
		if _, err := server.Read(buffer); err != nil {
			return
		}
		_, _ = io.WriteString(server, "HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	}()
	return client
}

// ---- 新增：关闭连接 handler 测试 ----

type fakeStatsInstance struct {
	tracker    *core.TrafficTracker
	closeOK    bool
	closeCount int
}

func (f *fakeStatsInstance) TrafficTracker() *core.TrafficTracker { return f.tracker }

func (f *fakeStatsInstance) CloseConnection(id int64) bool {
	return f.closeOK
}

func (f *fakeStatsInstance) CloseAllConnections() int { return f.closeCount }

func TestStatsHandlerCloseConnection(t *testing.T) {
	tests := []struct {
		name string
		inst statsInstance
		id   string
		want int
	}{
		{name: "nil instance", inst: nil, id: "1", want: http.StatusServiceUnavailable},
		{name: "invalid id", inst: &fakeStatsInstance{}, id: "abc", want: http.StatusBadRequest},
		{name: "not found", inst: &fakeStatsInstance{closeOK: false}, id: "1", want: http.StatusNotFound},
		{name: "ok", inst: &fakeStatsInstance{closeOK: true}, id: "1", want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStatsHandler(nil, nil, tt.inst)
			req := withURLParam(jsonRequest(http.MethodDelete, "/api/stats/connections/"+tt.id, ""), "id", tt.id)
			rr := httptest.NewRecorder()
			handler.CloseConnection(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}

func TestStatsHandlerCloseAllConnections(t *testing.T) {
	tests := []struct {
		name string
		inst statsInstance
		want int
		body string
	}{
		{name: "nil instance", inst: nil, want: http.StatusServiceUnavailable},
		{name: "ok", inst: &fakeStatsInstance{closeCount: 3}, want: http.StatusOK, body: `"closed":3`},
		{name: "zero", inst: &fakeStatsInstance{closeCount: 0}, want: http.StatusOK, body: `"closed":0`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStatsHandler(nil, nil, tt.inst)
			rr := httptest.NewRecorder()
			handler.CloseAllConnections(rr, httptest.NewRequest(http.MethodDelete, "/api/stats/connections", nil))
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d", rr.Code, tt.want)
			}
			if tt.body != "" && !strings.Contains(rr.Body.String(), tt.body) {
				t.Fatalf("body = %q", rr.Body.String())
			}
		})
	}
}

// ---- 新增：批量测速 handler 测试 ----

func TestTestHandlerRunBatch(t *testing.T) {
	nodeMgr, _, _, _ := newAPIManagers(t)
	handler := NewTestHandler(
		func() string { return "http://example.test/" },
		nodeMgr,
		&fakeDialer{connFactory: func() net.Conn { return newHTTPResponseConn() }, delay: 12},
	)

	t.Run("invalid json", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.RunBatch(rr, jsonRequest(http.MethodPost, "/api/nodes/test-batch", `{`))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rr.Code)
		}
	})

	t.Run("empty items", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.RunBatch(rr, jsonRequest(http.MethodPost, "/api/nodes/test-batch", `{"items":[]}`))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rr.Code)
		}
	})

	t.Run("batch success and isolation", func(t *testing.T) {
		body := `{"items":[
			{"tag":"a","test_type":"tcp","server":"1.1.1.1","port":80},
			{"tag":"b","test_type":"udp","server":"2.2.2.2","port":80},
			{"tag":"c","test_type":"tcp","server":"3.3.3.3","port":80}
		],"concurrency":2}`
		rr := httptest.NewRecorder()
		handler.RunBatch(rr, jsonRequest(http.MethodPost, "/api/nodes/test-batch", body))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d %s", rr.Code, rr.Body.String())
		}
		resp := decodeBody[struct {
			Results []model.TestResult `json:"results"`
		}](t, rr)
		if len(resp.Results) != 3 {
			t.Fatalf("results len = %d", len(resp.Results))
		}
		// a 和 c 走 tcp 成功；b 走 udp 不支持 → 隔离为错误结果
		if !resp.Results[0].Success || resp.Results[0].Error != "" {
			t.Fatalf("results[0] = %#v", resp.Results[0])
		}
		if resp.Results[1].Error == "" {
			t.Fatalf("results[1] should be isolated failure, got %#v", resp.Results[1])
		}
		if resp.Results[1].Tag != "b" {
			t.Fatalf("results[1].Tag = %q", resp.Results[1].Tag)
		}
		if !resp.Results[2].Success {
			t.Fatalf("results[2] = %#v", resp.Results[2])
		}
	})

	t.Run("default concurrency", func(t *testing.T) {
		body := `{"items":[{"tag":"x","test_type":"tcp","server":"1.1.1.1","port":80}]}`
		rr := httptest.NewRecorder()
		handler.RunBatch(rr, jsonRequest(http.MethodPost, "/api/nodes/test-batch", body))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d", rr.Code)
		}
	})
}
