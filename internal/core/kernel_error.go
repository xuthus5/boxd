package core

import (
	"errors"
	"os"
	"strings"
)

// 内核启动 / 配置应用稳定错误码，供前端展示可操作提示。
const (
	KernelErrorConfigInvalid = "config_invalid"
	KernelErrorConfigMissing = "config_missing"
	KernelErrorRestartFailed = "restart_failed"
	KernelErrorStartFailed   = "start_failed"
	KernelErrorPermission    = "permission"
	KernelErrorUnknown       = "unknown"
)

// ClassifyKernelError 将内核/配置应用失败映射为稳定错误码。
func ClassifyKernelError(msg string, err error) string {
	if code := classifyKernelErrorValue(err); code != "" {
		return code
	}
	return classifyKernelErrorMessage(msg)
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
