package core

import (
	"errors"
	"net/netip"
	"os"
	"strings"
)

// 内核启动 / 配置应用稳定错误码，供前端展示可操作提示。
const (
	KernelErrorConfigInvalid   = "config_invalid"
	KernelErrorConfigMissing   = "config_missing"
	KernelErrorRestartFailed   = "restart_failed"
	KernelErrorStartFailed     = "start_failed"
	KernelErrorPermission      = "permission"
	KernelErrorIPv6Unavailable = "ipv6_unavailable"
	KernelErrorUnknown         = "unknown"
)

// ClassifyKernelError 将内核/配置应用失败映射为稳定错误码。
func ClassifyKernelError(msg string, err error) string {
	if isIPv6AddressSetupError(msg) {
		return KernelErrorIPv6Unavailable
	}
	if err != nil && isIPv6AddressSetupError(err.Error()) {
		return KernelErrorIPv6Unavailable
	}
	if code := classifyKernelErrorValue(err); code != "" {
		return code
	}
	return classifyKernelErrorMessage(msg)
}

func isIPv6AddressSetupError(msg string) bool {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "set ipv6 address") {
		return true
	}
	for _, detail := range strings.Split(lower, "add address ")[1:] {
		fields := strings.Fields(detail)
		if len(fields) == 0 {
			continue
		}
		prefix, err := netip.ParsePrefix(strings.TrimSuffix(fields[0], ":"))
		if err == nil && prefix.Addr().Is6() {
			return true
		}
	}
	return false
}

func classifyKernelErrorValue(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, os.ErrNotExist) {
		return KernelErrorConfigMissing
	}
	if errors.Is(err, os.ErrPermission) {
		return KernelErrorPermission
	}
	return ""
}

func classifyKernelErrorMessage(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return KernelErrorUnknown
	}
	switch {
	case strings.Contains(lower, "no such file"), strings.Contains(lower, "not exist"), strings.Contains(lower, "cannot find"):
		return KernelErrorConfigMissing
	case strings.Contains(lower, "permission denied"), strings.Contains(lower, "operation not permitted"):
		return KernelErrorPermission
	case strings.Contains(lower, "restart failed"):
		return KernelErrorRestartFailed
	case strings.Contains(lower, "decode"),
		strings.Contains(lower, "invalid"),
		strings.Contains(lower, "unmarshal"),
		strings.Contains(lower, "unknown field"),
		strings.Contains(lower, "missing required"),
		strings.Contains(lower, "legacy"),
		strings.Contains(lower, "missing "),
		strings.Contains(lower, "required"):
		return KernelErrorConfigInvalid
	case strings.Contains(lower, "start failed"),
		strings.Contains(lower, "listen"),
		strings.Contains(lower, "bind"),
		strings.Contains(lower, "address already in use"),
		strings.Contains(lower, "factory failed"):
		return KernelErrorStartFailed
	default:
		return KernelErrorUnknown
	}
}
