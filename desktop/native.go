package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// 可注入命令执行钩子，便于单测覆盖而不依赖真实系统命令。
// 使用 context 超时避免 dbus 等外部命令挂起阻塞调用方。
var (
	execLookPath = exec.LookPath
	runCommand   = func(name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), nativeCommandTimeout)
		defer cancel()
		return exec.CommandContext(ctx, name, args...).CombinedOutput()
	}
	runOutputCommand = func(name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), nativeCommandTimeout)
		defer cancel()
		return exec.CommandContext(ctx, name, args...).Output()
	}
)

// nativeCommandTimeout 外部命令执行超时，防止 dbus 不可用时阻塞。
const nativeCommandTimeout = 5 * time.Second

// NativeCapabilities 提供桌面原生能力：自启、单实例、对话框、通知、代理切换等。
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

// SetSystemProxy 切换 GNOME 系统代理（仅内嵌模式且桌面会话）。
// enabled=true 时使用 mixed 入站 127.0.0.1:1080。
func (n *NativeCapabilities) SetSystemProxy(_ context.Context, enabled bool) error {
	if runtime.GOOS != "linux" {
		return errors.New("system proxy is only supported on linux")
	}
	return setGNOMEProxy(enabled)
}

// setGNOMEProxy 通过 gsettings 设置 org.gnome.system.proxy。
func setGNOMEProxy(enabled bool) error {
	if _, err := execLookPath("gsettings"); err != nil {
		return errors.New("gsettings is not available")
	}
	mode := "none"
	if enabled {
		mode = "manual"
	}
	if err := gsettingsSet("org.gnome.system.proxy", "mode", mode); err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	host := "127.0.0.1"
	port := "1080"
	if err := gsettingsSet("org.gnome.system.proxy.http", "host", host); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.http", "port", port); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "host", host); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "port", port); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.socks", "host", host); err != nil {
		return err
	}
	return gsettingsSet("org.gnome.system.proxy.socks", "port", port)
}

// gsettingsSet 设置 gsettings 键值。
func gsettingsSet(schema, key, value string) error {
	output, err := runCommand("gsettings", "set", schema, key, value)
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			return err
		}
		return errors.New(detail)
	}
	return nil
}

// SystemProxyStatus 返回当前系统代理状态。
func (n *NativeCapabilities) SystemProxyStatus(_ context.Context) (map[string]any, error) {
	if runtime.GOOS != "linux" {
		return map[string]any{"enabled": false, "available": false}, nil
	}
	if _, err := execLookPath("gsettings"); err != nil {
		return map[string]any{"enabled": false, "available": false}, nil
	}
	output, err := runOutputCommand("gsettings", "get", "org.gnome.system.proxy", "mode")
	if err != nil {
		return map[string]any{"enabled": false, "available": true}, nil
	}
	mode := strings.TrimSpace(strings.Trim(strings.TrimSpace(string(output)), "'"))
	return map[string]any{"enabled": mode == "manual", "available": true}, nil
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
