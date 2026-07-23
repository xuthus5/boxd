package api

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"

	"github.com/xuthus5/boxd/internal/model"
)

// 节点测速 / DNS 探测稳定错误码，供前端提示与诊断复制。
const (
	ProbeErrorUnavailable  = "unavailable"
	ProbeErrorInvalidInput = "invalid_input"
	ProbeErrorTimeout      = "timeout"
	ProbeErrorNetwork      = "network"
	ProbeErrorNoResponse   = "no_response"
	ProbeErrorUnsupported  = "unsupported"
	ProbeErrorDNSRcode     = "dns_rcode"
	ProbeErrorEmpty        = "empty_response"
	ProbeErrorUnknown      = "unknown"
)

// classifyProbeError 将失败原因映射为稳定错误码。
func classifyProbeError(msg string, err error) string {
	if code := classifyProbeErrorValue(err); code != "" {
		return code
	}
	return classifyProbeErrorMessage(msg)
}

func classifyProbeErrorValue(err error) string {
	if err == nil {
		return ""
	}
	if isProbeTimeoutError(err) {
		return ProbeErrorTimeout
	}
	if isProbeNetworkError(err) {
		return ProbeErrorNetwork
	}
	return ""
}

func classifyProbeErrorMessage(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return ProbeErrorUnknown
	}
	switch {
	case strings.Contains(lower, "not available"), strings.Contains(lower, "service not available"):
		return ProbeErrorUnavailable
	case strings.Contains(lower, "unsupported"), strings.Contains(lower, "not probeable"):
		return ProbeErrorUnsupported
	case strings.Contains(lower, "invalid"), strings.Contains(lower, "required"), strings.Contains(lower, "empty address"):
		return ProbeErrorInvalidInput
	case strings.Contains(lower, "no response"):
		return ProbeErrorNoResponse
	case strings.Contains(lower, "empty dns response"), strings.Contains(lower, "empty response"):
		return ProbeErrorEmpty
	case strings.Contains(lower, "dns rcode"):
		return ProbeErrorDNSRcode
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline exceeded"), strings.Contains(lower, "i/o timeout"):
		return ProbeErrorTimeout
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "network is unreachable"),
		strings.Contains(lower, "network down"),
		strings.Contains(lower, "broken pipe"),
		strings.Contains(lower, "ping failed"):
		return ProbeErrorNetwork
	default:
		return ProbeErrorUnknown
	}
}

func failedTestResult(msg string, err error) model.TestResult {
	message := strings.TrimSpace(msg)
	if message == "" && err != nil {
		message = err.Error()
	}
	if message == "" {
		message = "probe failed"
	}
	return model.TestResult{
		Error:     message,
		ErrorCode: classifyProbeError(message, err),
	}
}

func failedDNSProbeResult(base DNSProbeResult, msg string, err error) DNSProbeResult {
	message := strings.TrimSpace(msg)
	if message == "" && err != nil {
		message = err.Error()
	}
	if message == "" {
		message = "probe failed"
	}
	base.Success = false
	base.Error = message
	base.ErrorCode = classifyProbeError(message, err)
	return base
}

func isProbeTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isProbeNetworkError(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH)
}
