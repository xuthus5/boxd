//go:build !linux

package core

// 桌面平台默认使用本机代理，TUN 由用户显式启用并授予平台权限。
func bootstrapTUNAvailable() bool { return false }
