package main

import (
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	trayMu             sync.Mutex
	activeTray         *application.SystemTray
	activeTrayLanguage string
)

// setupTray 配置系统托盘：显示窗口、启停内核、重载界面、退出，文案跟随界面语言。
func setupTray(app *application.App, rt *desktopRuntime) {
	if app.SystemTray == nil {
		return
	}
	tray := app.SystemTray.New()
	tray.SetLabel("boxd")
	tray.SetIcon(trayIcon)
	language := trayLanguage(rt)
	applyTrayLabels(app, rt, tray, language)
	tray.Run()
	setActiveTray(tray, language)
}

// applyTrayLabels 按语言更新托盘 tooltip 与菜单，语言切换后可重复调用。
func applyTrayLabels(app *application.App, rt *desktopRuntime, tray *application.SystemTray, language string) {
	if tray == nil {
		return
	}
	labels := trayLabelsFor(language)
	tray.SetTooltip(labels.Tooltip)
	tray.SetMenu(trayMenu(app, rt, labels))
}

// trayMenu 构建托盘菜单。
func trayMenu(app *application.App, rt *desktopRuntime, labels trayLabels) *application.Menu {
	menu := application.NewMenu()
	menu.Add(labels.Show).OnClick(func(_ *application.Context) {
		if window, ok := app.Window.GetByName("main"); ok {
			window.Show()
			window.Focus()
		}
	})
	menu.AddSeparator()
	menu.Add(labels.StartKernel).OnClick(func(_ *application.Context) {
		if err := startKernel(rt); err != nil {
			slog.Error("kernel start failed", "err", err)
		}
	})
	menu.Add(labels.StopKernel).OnClick(func(_ *application.Context) {
		if err := stopKernel(rt); err != nil {
			slog.Error("kernel stop failed", "err", err)
		}
	})
	menu.Add(labels.RestartKernel).OnClick(func(_ *application.Context) {
		if err := restartKernel(rt); err != nil {
			slog.Error("kernel restart failed", "err", err)
		}
	})
	menu.AddSeparator()
	// 渲染进程卡死后手动重建 UI 的兜底入口。
	menu.Add(labels.ReloadUI).OnClick(func(_ *application.Context) {
		reloadMainWindow(app)
	})
	menu.AddSeparator()
	menu.Add(labels.Quit).OnClick(func(_ *application.Context) {
		app.Quit()
	})
	return menu
}

// setActiveTray 记录当前托盘与已生效语言，供语言切换时原地刷新。
func setActiveTray(tray *application.SystemTray, language string) {
	trayMu.Lock()
	defer trayMu.Unlock()
	activeTray = tray
	activeTrayLanguage = language
}

// refreshTrayLanguage 界面语言变化后重建托盘菜单；托盘未创建或语言未变时直接返回。
func refreshTrayLanguage(rt *desktopRuntime) {
	trayMu.Lock()
	tray, current := activeTray, activeTrayLanguage
	trayMu.Unlock()
	if tray == nil {
		return
	}
	language := trayLanguage(rt)
	if language == current {
		return
	}
	applyTrayLabels(globalApp, rt, tray, language)
	setActiveTray(tray, language)
}

// startKernel 启动 sing-box 内核。
func startKernel(rt *desktopRuntime) error {
	if rt == nil || rt.instance == nil {
		return nil
	}
	return rt.instance.Start()
}

// stopKernel 停止 sing-box 内核。
func stopKernel(rt *desktopRuntime) error {
	if rt == nil || rt.instance == nil {
		return nil
	}
	return rt.instance.Stop()
}

// restartKernel 重启 sing-box 内核。
func restartKernel(rt *desktopRuntime) error {
	if rt == nil || rt.instance == nil {
		return nil
	}
	return rt.instance.Restart()
}
