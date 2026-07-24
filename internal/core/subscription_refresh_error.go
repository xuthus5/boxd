package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
)

// 订阅刷新错误码：前端可据此展示可操作提示。
const (
	SubRefreshInvalidURL      = "invalid_url"
	SubRefreshNetwork         = "network"
	SubRefreshTimeout         = "timeout"
	SubRefreshUnauthorized    = "unauthorized"
	SubRefreshForbidden       = "forbidden"
	SubRefreshHTTP            = "http_status"
	SubRefreshEmpty           = "empty_content"
	SubRefreshContentTooLarge = "content_too_large"
	SubRefreshNotFound        = "not_found"
	SubRefreshUnknown         = "unknown"
)

// SubscriptionRefreshError 带稳定错误码的订阅刷新失败。
type SubscriptionRefreshError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

func (e *SubscriptionRefreshError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// SubscriptionRefreshFailure 批量刷新时的单项失败摘要。
type SubscriptionRefreshFailure struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newSubscriptionRefreshError(code, message string, status int) *SubscriptionRefreshError {
	return &SubscriptionRefreshError{Code: code, Message: message, Status: status}
}

func classifySubscriptionRefreshError(err error) *SubscriptionRefreshError {
	if err == nil {
		return nil
	}
	var refreshErr *SubscriptionRefreshError
	if errors.As(err, &refreshErr) {
		if refreshErr.Code == "" {
			refreshErr.Code = SubRefreshUnknown
		}
		return refreshErr
	}

	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "subscription not found"):
		return newSubscriptionRefreshError(SubRefreshNotFound, msg, 0)
	case strings.Contains(lower, "content too large") || strings.Contains(lower, "too large"):
		return newSubscriptionRefreshError(SubRefreshContentTooLarge, msg, 0)
	case isTimeoutError(err) || strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return newSubscriptionRefreshError(SubRefreshTimeout, msg, 0)
	case isNetworkError(err) || strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host"):
		return newSubscriptionRefreshError(SubRefreshNetwork, msg, 0)
	case strings.Contains(lower, "unsupported protocol scheme") || strings.HasPrefix(lower, "parse ") || strings.Contains(lower, "invalid url"):
		return newSubscriptionRefreshError(SubRefreshInvalidURL, msg, 0)
	default:
		return newSubscriptionRefreshError(SubRefreshUnknown, msg, 0)
	}
}

func classifyHTTPStatus(status int) *SubscriptionRefreshError {
	switch status {
	case 401:
		return newSubscriptionRefreshError(SubRefreshUnauthorized, fmt.Sprintf("subscription HTTP %d", status), status)
	case 403:
		return newSubscriptionRefreshError(SubRefreshForbidden, fmt.Sprintf("subscription HTTP %d", status), status)
	default:
		return newSubscriptionRefreshError(SubRefreshHTTP, fmt.Sprintf("subscription HTTP %d", status), status)
	}
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isNetworkError(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET)
}
