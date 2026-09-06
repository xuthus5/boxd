//go:build !linux || !cgo

package main

import "log/slog"

// hookWebProcessTerminated 非 Linux/无 cgo 平台的空实现：
// 渲染进程自动恢复仅对 GTK/WebKitGTK 后端有意义。
func hookWebProcessTerminated(reloadFn func()) {
	_ = reloadFn
	slog.Info("web process recovery not supported on this platform")
}
