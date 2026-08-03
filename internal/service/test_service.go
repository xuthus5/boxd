package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

// testProbeTimeout 单次测速超时。
const testProbeTimeout = 5 * time.Second

// outboundDialer 抽象内核出站拨号能力，便于测试注入。
type outboundDialer interface {
	DialOutbound(ctx context.Context, tag, network, addr string) (net.Conn, error)
	OutboundDelay(ctx context.Context, tag, link string, timeout time.Duration) (uint16, error)
}

// TestRequest 单次测速请求。
type TestRequest struct {
	Tag      string `json:"tag"`
	TestType string `json:"test_type"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
}

// TestService 提供节点测速用例逻辑。
type TestService struct {
	settingsURL func() string
	nodeManager *core.NodeManager
	instance    outboundDialer
}

// NewTestService 构造测速用例服务。
func NewTestService(settingsURL func() string, nodeManager *core.NodeManager, instance outboundDialer) *TestService {
	return &TestService{settingsURL: settingsURL, nodeManager: nodeManager, instance: instance}
}

// Run 执行单点测速并持久化结果。
func (s *TestService) Run(ctx context.Context, req TestRequest) (model.TestResult, error) {
	if req.Tag == "" || req.TestType == "" {
		return model.TestResult{}, Errorf(400, model.ErrorInvalidRequest, "tag and test_type are required")
	}
	result, err := s.dispatchTest(ctx, req)
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err != nil {
		return result, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	if err := s.persistTestResult(result); err != nil {
		return result, Errorf(500, model.ErrorInternal, "failed to save test result")
	}
	return result, nil
}

// RunBatch 并发测速多个节点。
func (s *TestService) RunBatch(ctx context.Context, req TestBatchRequest) ([]model.TestResult, error) {
	if err := normalizeTestBatchRequest(&req); err != nil {
		return nil, Errorf(400, model.ErrorInvalidRequest, "%v", err)
	}
	results := s.runTestBatch(ctx, req.Items, req.Concurrency)
	if ctx.Err() != nil {
		return results, ctx.Err()
	}
	return results, nil
}

// ListResults 返回全部已保存测速结果。
func (s *TestService) ListResults(_ context.Context) (map[string]map[string]model.TestResult, error) {
	if s.nodeManager == nil {
		return nil, Errorf(500, model.ErrorInternal, "node manager not available")
	}
	return s.nodeManager.GetAllTestResults(), nil
}

// ListHistory 返回节点测速历史。
func (s *TestService) ListHistory(_ context.Context, tag string) (map[string]any, error) {
	if tag != "" {
		return map[string]any{
			"tag":     tag,
			"history": s.nodeManager.GetTestHistory(tag),
		}, nil
	}
	return map[string]any{
		"history": s.nodeManager.GetAllTestHistory(),
	}, nil
}

func (s *TestService) persistTestResult(result model.TestResult) error {
	if s.nodeManager == nil {
		return errors.New("node manager not available")
	}
	key := result.Tag + "_" + nonEmpty(result.TestType, "test")
	return s.nodeManager.SaveTestResult(key, result)
}

func (s *TestService) dispatchTest(ctx context.Context, req TestRequest) (model.TestResult, error) {
	var result model.TestResult
	switch req.TestType {
	case "tcp":
		result = s.tcpPing(ctx, req)
	case "http":
		result = s.httpTest(ctx, req)
	case "icmp":
		result = s.icmpPing(ctx, req)
	default:
		return model.TestResult{}, errors.New("unsupported test_type: " + req.TestType)
	}
	result.Tag = req.Tag
	result.TestType = req.TestType
	return result, nil
}

func (s *TestService) tcpPing(ctx context.Context, req TestRequest) model.TestResult {
	if s.instance == nil {
		return failedTestResult("test service not available", nil)
	}
	link := ""
	if s.settingsURL != nil {
		link = s.settingsURL()
	}
	if link != "" {
		if err := core.ValidateHTTPURL(link); err != nil {
			link = ""
		}
	}
	probeCtx, cancel := context.WithTimeout(ctx, testProbeTimeout)
	defer cancel()
	delay, err := s.instance.OutboundDelay(probeCtx, req.Tag, link, testProbeTimeout)
	if err != nil {
		return failedTestResult(err.Error(), err)
	}
	if delay == 0 {
		return failedTestResult("delay test failed: no response", nil)
	}
	return model.TestResult{Success: true, LatencyMs: float64(delay)}
}

func (s *TestService) httpTest(ctx context.Context, req TestRequest) model.TestResult {
	target := req.Server
	if s.settingsURL != nil {
		if url := s.settingsURL(); url != "" {
			target = url
		}
	}
	if target == "" {
		target = defaultTestURL
	}
	if err := core.ValidateHTTPURL(target); err != nil {
		return failedTestResult(err.Error(), err)
	}
	if s.instance == nil {
		return failedTestResult("test service not available", nil)
	}

	probeCtx, cancel := context.WithTimeout(ctx, testProbeTimeout)
	defer cancel()
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.instance.DialOutbound(ctx, req.Tag, network, addr)
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

func (s *TestService) icmpPing(ctx context.Context, req TestRequest) model.TestResult {
	server := strings.TrimSpace(req.Server)
	if server == "" || !isValidPingTarget(server) {
		return failedTestResult("invalid server address", nil)
	}

	latency, err := ICMPPing(ctx, server)
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

func nonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
