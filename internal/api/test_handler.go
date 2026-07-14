package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

type TestRequest struct {
	Tag      string `json:"tag"`
	TestType string `json:"test_type"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
}

type TestHandler struct {
	settingsURL func() string
	nodeManager *core.NodeManager
	instance    outboundDialer
}

var dialTimeout = net.DialTimeout

var commandOutput = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type outboundDialer interface {
	DialOutbound(ctx context.Context, tag, network, addr string) (net.Conn, error)
}

func NewTestHandler(settingsURLFn func() string, nodeManager *core.NodeManager, instance outboundDialer) *TestHandler {
	return &TestHandler{settingsURL: settingsURLFn, nodeManager: nodeManager, instance: instance}
}

// dispatchTest 执行单点测速并返回结果（不持久化）。不支持类型返回 error。
func (h *TestHandler) dispatchTest(req TestRequest) (model.TestResult, error) {
	var result model.TestResult
	switch req.TestType {
	case "tcp":
		result = h.tcpPing(req)
	case "http":
		result = h.httpTest(req)
	case "icmp":
		result = h.icmpPing(req)
	default:
		return model.TestResult{}, fmt.Errorf("unsupported test_type: %s", req.TestType)
	}
	result.Tag = req.Tag
	result.TestType = req.TestType
	return result, nil
}

func (h *TestHandler) Run(w http.ResponseWriter, r *http.Request) {
	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	if req.Tag == "" || req.TestType == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag and test_type are required")
		return
	}

	result, err := h.dispatchTest(req)
	if err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}

	// 持久化结果
	key := req.Tag + "_" + req.TestType
	if err := h.nodeManager.SaveTestResult(key, result); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to save test result")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// RunBatch POST /api/nodes/test-batch：并发对多节点测速，结果统一返回并持久化。
// 失败的节点隔离为独立错误结果，不影响其他节点。
func (h *TestHandler) RunBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items       []TestRequest `json:"items"`
		Concurrency int           `json:"concurrency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request")
		return
	}
	if len(req.Items) == 0 {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "items is empty")
		return
	}
	if req.Concurrency <= 0 {
		req.Concurrency = defaultBatchConcurrency
	}

	results := make([]model.TestResult, len(req.Items))
	sem := make(chan struct{}, req.Concurrency)
	var wg sync.WaitGroup

	for i, item := range req.Items {
		wg.Add(1)
		go func(idx int, it TestRequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			it.Tag = firstNonEmpty(it.Tag, it.Server)
			r, err := h.dispatchTest(it)
			if err != nil {
				r = model.TestResult{Tag: it.Tag, TestType: it.TestType, Error: err.Error()}
			}
			results[idx] = r

			// 持久化（失败不影响整体）
			key := r.Tag + "_" + nonEmpty(r.TestType, "test")
			if h.nodeManager != nil {
				_ = h.nodeManager.SaveTestResult(key, r)
			}
		}(i, item)
	}
	wg.Wait()

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

var defaultBatchConcurrency = 8

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func nonEmpty(a, fallback string) string {
	if a != "" {
		return a
	}
	return fallback
}

func (h *TestHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	all := h.nodeManager.GetAllTestResults()
	writeJSON(w, http.StatusOK, all)
}

func (h *TestHandler) tcpPing(req TestRequest) model.TestResult {
	addr := net.JoinHostPort(req.Server, fmt.Sprintf("%d", req.Port))
	start := time.Now()
	conn, err := dialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return model.TestResult{Error: err.Error()}
	}
	_ = conn.Close()
	latency := time.Since(start).Seconds() * 1000
	return model.TestResult{Success: true, LatencyMs: latency}
}

func (h *TestHandler) httpTest(req TestRequest) model.TestResult {
	target := req.Server
	if h.settingsURL != nil {
		if u := h.settingsURL(); u != "" {
			target = u
		}
	}
	if target == "" {
		target = defaultTestURL
	}

	start := time.Now()

	if h.instance == nil {
		return model.TestResult{Error: "test service not available"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport := &http.Transport{
		DialContext: func(c context.Context, network, addr string) (net.Conn, error) {
			return h.instance.DialOutbound(ctx, req.Tag, network, addr)
		},
	}
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	resp, err := client.Get(target)
	if err != nil {
		return model.TestResult{Error: err.Error()}
	}
	_ = resp.Body.Close()
	latency := time.Since(start).Seconds() * 1000
	return model.TestResult{Success: true, LatencyMs: latency}
}

// isValidPingTarget 使用 allowlist 验证 ping 目标：仅允许 IP 地址或合法域名。
func isValidPingTarget(server string) bool {
	if server == "" || len(server) > 253 {
		return false
	}
	// 拒绝包含 shell 元字符的输入
	if strings.ContainsAny(server, ";&|`$(){}\n\r\t ") {
		return false
	}
	// 允许 IPv4/IPv6 地址
	if net.ParseIP(server) != nil {
		return true
	}
	// 允许合法域名格式（字母、数字、点、连字符）
	for _, c := range server {
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		isDigit := c >= '0' && c <= '9'
		isAllowed := isLower || isUpper || isDigit || c == '.' || c == '-'
		if !isAllowed {
			return false
		}
	}
	// 必须包含至少一个点（防止 localhost 等特殊名称）
	return strings.Contains(server, ".")
}

func (h *TestHandler) icmpPing(req TestRequest) model.TestResult {
	server := strings.TrimSpace(req.Server)
	if server == "" || !isValidPingTarget(server) {
		return model.TestResult{Error: "invalid server address"}
	}

	start := time.Now()
	output, err := commandOutput("ping", "-c", "1", "-W", "3", server)
	latency := time.Since(start).Seconds() * 1000
	if err != nil {
		return model.TestResult{Error: fmt.Sprintf("ping failed: %s", strings.TrimSpace(string(output)))}
	}

	for _, line := range strings.Split(string(output), "\n") {
		if ms, ok := parsePingLatency(line); ok {
			latency = ms
			break
		}
	}
	return model.TestResult{Success: true, LatencyMs: latency}
}

func parsePingLatency(line string) (float64, bool) {
	idx := strings.Index(line, "time=")
	if idx < 0 {
		return 0, false
	}

	value := line[idx+len("time="):]
	value = strings.TrimSpace(value)
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0, false
	}

	raw := strings.TrimSuffix(fields[0], "ms")
	ms, err := strconv.ParseFloat(raw, 64)
	if err != nil || ms <= 0 {
		return 0, false
	}
	return ms, true
}
