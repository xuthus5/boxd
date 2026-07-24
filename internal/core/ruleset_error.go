package core

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"

	"github.com/xuthus5/boxd/internal/model"
)

// 规则集更新稳定错误码，供前端提示与诊断复制。
const (
	RuleSetErrorNotUpdatable = "not_updatable"
	RuleSetErrorUnsupported  = "unsupported"
	RuleSetErrorInvalidURL   = "invalid_url"
	RuleSetErrorBlockedURL   = "blocked_url"
	RuleSetErrorNetwork      = "network"
	RuleSetErrorTimeout      = "timeout"
	RuleSetErrorHTTP         = "http_status"
	RuleSetErrorEmpty        = "empty_content"
	RuleSetErrorTooLarge     = "content_too_large"
	RuleSetErrorPermission   = "permission"
	RuleSetErrorCache        = "cache"
	RuleSetErrorUnknown      = "unknown"
)

// ClassifyRuleSetUpdateError 将规则集更新失败映射为稳定错误码。
func ClassifyRuleSetUpdateError(msg string, err error) string {
	if code := classifyRuleSetErrorValue(err); code != "" {
		return code
	}
	return classifyRuleSetErrorMessage(msg)
}

func classifyRuleSetErrorValue(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrRuleSetNotUpdatable) {
		return RuleSetErrorNotUpdatable
	}
	if errors.Is(err, errSubscriptionURLBlocked) {
		return RuleSetErrorBlockedURL
	}
	if errors.Is(err, errSubscriptionURLInvalid) {
		return RuleSetErrorInvalidURL
	}
	if errors.Is(err, ErrRuleSetContentTooLarge) {
		return RuleSetErrorTooLarge
	}
	if errors.Is(err, ErrRuleSetCacheDisabled) {
		return RuleSetErrorCache
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return RuleSetErrorTimeout
	}
	if isRuleSetTimeoutError(err) {
		return RuleSetErrorTimeout
	}
	if isRuleSetNetworkError(err) {
		return RuleSetErrorNetwork
	}
	return ""
}

func classifyRuleSetErrorMessage(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return RuleSetErrorUnknown
	}
	switch {
	case strings.Contains(lower, "not updatable"):
		return RuleSetErrorNotUpdatable
	case strings.Contains(lower, "not auto-updated"), strings.Contains(lower, "not supported"):
		return RuleSetErrorUnsupported
	case strings.Contains(lower, "url is empty"), strings.Contains(lower, "invalid url"), strings.Contains(lower, "unsupported protocol"):
		return RuleSetErrorInvalidURL
	case strings.Contains(lower, "private or local address"), strings.Contains(lower, "dial address is not public"):
		return RuleSetErrorBlockedURL
	case strings.Contains(lower, "content is too large"), strings.Contains(lower, "content too large"):
		return RuleSetErrorTooLarge
	case strings.Contains(lower, "empty rule-set"), strings.Contains(lower, "empty body"), strings.Contains(lower, "empty content"):
		return RuleSetErrorEmpty
	case strings.Contains(lower, "unexpected status"):
		return RuleSetErrorHTTP
	case strings.Contains(lower, "permission denied"), strings.Contains(lower, "operation not permitted"):
		return RuleSetErrorPermission
	case strings.Contains(lower, "cache is unavailable"), strings.Contains(lower, "bbolt"):
		return RuleSetErrorCache
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline exceeded"), strings.Contains(lower, "i/o timeout"):
		return RuleSetErrorTimeout
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "network is unreachable"),
		strings.Contains(lower, "network down"),
		strings.Contains(lower, "tls:"),
		strings.Contains(lower, "x509"):
		return RuleSetErrorNetwork
	default:
		return RuleSetErrorUnknown
	}
}

func failRuleSetResult(result model.RuleSetUpdateResult, msg string, err error) model.RuleSetUpdateResult {
	message := strings.TrimSpace(msg)
	if message == "" && err != nil {
		message = err.Error()
	}
	if message == "" {
		message = "rule-set update failed"
	}
	result.OK = false
	result.Error = message
	result.ErrorCode = ClassifyRuleSetUpdateError(message, err)
	return result
}

func isRuleSetTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isRuleSetNetworkError(err error) bool {
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
