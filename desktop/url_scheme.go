package main

import (
	"context"
	"net/url"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// URLHandler 处理 boxd:// URL Scheme 深链。
// 支持：boxd://import?link=<编码后的订阅/节点链接>
type URLHandler struct {
	app *application.App
	rt  *desktopRuntime
}

// NewURLHandler 构造 URL Scheme 处理器。
func NewURLHandler(app *application.App, rt *desktopRuntime) *URLHandler {
	return &URLHandler{app: app, rt: rt}
}

// Register 注册 URL 深链监听。
func (h *URLHandler) Register() {
	if h.app == nil {
		return
	}
	h.app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(e *application.ApplicationEvent) {
		raw := e.Context().URL()
		if raw == "" {
			return
		}
		h.handle(raw)
	})
}

// ParseBoxdURL 解析 boxd:// URL，返回 action 与参数。
func ParseBoxdURL(raw string) (action string, params url.Values, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(trimmed), "boxd://") {
		return "", nil, false
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", nil, false
	}
	action = u.Host
	// boxd://import?link=... 形式：host 是 "import"。
	// 兼容 boxd:import?link=... 形式：host 可能是 "boxd:import"。
	action = strings.TrimPrefix(action, "boxd:")
	if action == "" {
		action = strings.TrimPrefix(u.Path, "/")
	}
	return action, u.Query(), true
}

// handle 处理深链动作。
func (h *URLHandler) handle(raw string) {
	action, params, ok := ParseBoxdURL(raw)
	if !ok {
		return
	}
	switch action {
	case "import":
		link := params.Get("link")
		if link != "" {
			h.importLink(link)
		}
	case "show":
		h.focusWindow()
	default:
		h.focusWindow()
	}
}

// importLink 导入节点链接。
func (h *URLHandler) importLink(link string) {
	if h.rt == nil || h.rt.svc == nil {
		return
	}
	h.focusWindow()
	if _, err := h.rt.svc.Import().ParseLink(context.Background(), link); err != nil {
		return
	}
	// 通知前端有新的导入链接。
	h.app.Event.Emit("boxd:deep-link-import", map[string]string{"link": link})
}

// focusWindow 聚焦主窗口。
func (h *URLHandler) focusWindow() {
	if h.app == nil {
		return
	}
	if window, ok := h.app.Window.GetByName("main"); ok {
		window.Show()
		window.Focus()
	}
}
