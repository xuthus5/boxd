package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

const testProbeTimeout = 5 * time.Second

// dispatchTest 执行单点测速并返回结果（不持久化）。
func (h *TestHandler) dispatchTest(ctx context.Context, req TestRequest) (model.TestResult, error) {
	var result model.TestResult
	switch req.TestType {
	case "tcp":
		result = h.tcpPing(ctx, req)
	case "http":
		result = h.httpTest(ctx, req)
	case "icmp":
		result = h.icmpPing(ctx, req)
	default:
		return model.TestResult{}, fmt.Errorf("unsupported test_type: %s", req.TestType)
	}
	result.Tag = req.Tag
	result.TestType = req.TestType
	return result, nil
}

func (h *TestHandler) tcpPing(ctx context.Context, req TestRequest) model.TestResult {
	if h.instance == nil {
		return failedTestResult("test service not available", nil)
	}
	link := ""
	if h.settingsURL != nil {
		link = h.settingsURL()
	}
	if link != "" {
		if err := core.ValidateHTTPURL(link); err != nil {
			link = ""
		}
	}
	probeCtx, cancel := context.WithTimeout(ctx, testProbeTimeout)
	defer cancel()
	delay, err := h.instance.OutboundDelay(probeCtx, req.Tag, link, testProbeTimeout)
	if err != nil {
		return failedTestResult(err.Error(), err)
	}
	if delay == 0 {
		return failedTestResult("delay test failed: no response", nil)
	}
	return model.TestResult{Success: true, LatencyMs: float64(delay)}
}

func (h *TestHandler) httpTest(ctx context.Context, req TestRequest) model.TestResult {
	target := req.Server
	if h.settingsURL != nil {
		if url := h.settingsURL(); url != "" {
			target = url
		}
	}
	if target == "" {
		target = defaultTestURL
	}
	if err := core.ValidateHTTPURL(target); err != nil {
		return failedTestResult(err.Error(), err)
	}
	if h.instance == nil {
		return failedTestResult("test service not available", nil)
	}

	probeCtx, cancel := context.WithTimeout(ctx, testProbeTimeout)
	defer cancel()
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return h.instance.DialOutbound(ctx, req.Tag, network, addr)
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   testProbeTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	httpReq, err := http.NewRequestWithContext(probeCtx, http.MethodGet, target, nil)
	if err != nil {
		return failedTestResult(err.Error(), err)
	}
	startedAt := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return failedTestResult(err.Error(), err)
	}
	if resp == nil || resp.Body == nil {
		return failedTestResult("http test response body is nil", nil)
	}
	defer func() { _ = resp.Body.Close() }()
	return model.TestResult{Success: true, LatencyMs: time.Since(startedAt).Seconds() * 1000}
}

func (h *TestHandler) icmpPing(ctx context.Context, req TestRequest) model.TestResult {
	server := strings.TrimSpace(req.Server)
	if server == "" || !isValidPingTarget(server) {
		return failedTestResult("invalid server address", nil)
	}

	latency, err := service.ICMPPing(ctx, server)
	if err != nil {
		return failedTestResult(err.Error(), err)
	}
	return model.TestResult{Success: true, LatencyMs: latency}
}

// isValidPingTarget 使用 allowlist 验证 ping 目标：仅允许 IP 地址或合法域名。
func isValidPingTarget(server string) bool {
	if server == "" || len(server) > 253 {
		return false
	}
	if strings.ContainsAny(server, ";&|`$(){}\n\r\t ") {
		return false
	}
	if net.ParseIP(server) != nil {
		return true
	}
	for _, character := range server {
		isLower := character >= 'a' && character <= 'z'
		isUpper := character >= 'A' && character <= 'Z'
		isDigit := character >= '0' && character <= '9'
		if !isLower && !isUpper && !isDigit && character != '.' && character != '-' {
			return false
		}
	}
	return strings.Contains(server, ".")
}

func parsePingLatency(line string) (float64, bool) {
	index := strings.Index(line, "time=")
	if index < 0 {
		return 0, false
	}
	fields := strings.Fields(strings.TrimSpace(line[index+len("time="):]))
	if len(fields) == 0 {
		return 0, false
	}
	raw := strings.TrimSuffix(fields[0], "ms")
	milliseconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || milliseconds <= 0 {
		return 0, false
	}
	return milliseconds, true
}
