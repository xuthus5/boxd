package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// NativeCapabilities 提供桌面原生能力：自启、单实例、对话框、通知、数据目录等。
// 封装为 Wails 可绑定服务，供前端与托盘调用。
type NativeCapabilities struct {
	rt *desktopRuntime
}

// NewNativeCapabilities 构造原生能力服务。
func NewNativeCapabilities(rt *desktopRuntime) *NativeCapabilities {
	return &NativeCapabilities{rt: rt}
}

// IsAutostartEnabled 返回开机自启是否已启用。
func (n *NativeCapabilities) IsAutostartEnabled(_ context.Context) (bool, error) {
	if n.rt == nil || n.rt.autostart == nil {
		return false, nil
	}
	enabled, err := n.rt.autostart.IsEnabled()
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// SetAutostart 启用/禁用开机自启。
func (n *NativeCapabilities) SetAutostart(_ context.Context, enabled bool) error {
	if n.rt == nil || n.rt.autostart == nil {
		return errors.New("autostart is not available")
	}
	if enabled {
		return n.rt.autostart.Enable()
	}
	return n.rt.autostart.Disable()
}

// autostartEnabled 判断自启状态，供托盘使用（错误时不阻塞）。
func (n *NativeCapabilities) autostartEnabled() bool {
	enabled, _ := n.IsAutostartEnabled(context.Background())
	return enabled
}

// Runtime 返回桌面运行时信息。
func (n *NativeCapabilities) Runtime(_ context.Context) map[string]any {
	mode := "embedded"
	remoteURL := ""
	if n.rt != nil {
		if n.rt.cfg.Mode != "" {
			mode = n.rt.cfg.Mode
		}
		remoteURL = n.rt.cfg.RemoteURL
	}
	return map[string]any{
		"mode":       mode,
		"remote_url": remoteURL,
		"platform":   runtime.GOOS,
	}
}

// Notify 发送系统通知（Linux 通过 freedesktop 通知规范，不依赖 notify-send）。
func (n *NativeCapabilities) Notify(_ context.Context, title, message string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	return notifyLinux(title, message)
}

// notifyLinux 通过 Wails notifications 服务发送通知；服务不可用（如无 DBus 会话）时静默降级。
func notifyLinux(title, message string) error {
	return sendNotification(title, message)
}

// sendNotification 可注入的通知发送钩子，便于单测替换。
var sendNotification = func(title, message string) error {
	svc := getWailsNotificationService()
	if svc == nil {
		// DBus 会话不可用或服务未初始化时静默降级，不阻塞调用方。
		return nil
	}
	return notifyViaService(svc, title, message)
}

// notifyViaService 通过 notifications 服务发送通知，防御服务未完全初始化导致的 panic。
func notifyViaService(svc *notifications.NotificationService, title, message string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("notification service panic: %v", r)
		}
	}()
	return svc.SendNotification(notifications.NotificationOptions{
		ID:    fmt.Sprintf("boxd-%d", time.Now().UnixNano()),
		Title: title,
		Body:  message,
	})
}

// registeredNotificationService 保存注册的 notifications 服务实例，供发送时获取。
var registeredNotificationService *notifications.NotificationService

// getWailsNotificationService 返回已注册的通知服务（未注册时为 nil）。
func getWailsNotificationService() *notifications.NotificationService {
	return registeredNotificationService
}

// DataDir 返回内嵌模式数据目录。
func (n *NativeCapabilities) DataDir(_ context.Context) (string, error) {
	if n.rt == nil {
		return "", errors.New("runtime is not ready")
	}
	return n.rt.cfg.DataDir, nil
}

// ConfigPath 返回内嵌模式配置文件路径。
func (n *NativeCapabilities) ConfigPath(_ context.Context) (string, error) {
	if n.rt == nil {
		return "", errors.New("runtime is not ready")
	}
	return n.rt.cfg.ConfigPath, nil
}

// ResolvePath 解析相对路径为绝对路径（供文件对话框使用）。
func (n *NativeCapabilities) ResolvePath(_ context.Context, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return abs, nil
}
