package main

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// setupTray 配置系统托盘：显示状态、打开窗口、启停内核、退出。
func setupTray(app *application.App, rt *desktopRuntime) {
	if app.SystemTray == nil {
		return
	}
	tray := app.SystemTray.New()
	tray.SetLabel("boxd")
	tray.SetTooltip("boxd — sing-box control plane")
	tray.SetIcon(trayIcon)

	menu := application.NewMenu()
	menu.Add("Show").OnClick(func(_ *application.Context) {
		if window, ok := app.Window.GetByName("main"); ok {
			window.Show()
			window.Focus()
		}
	})
	menu.AddSeparator()

	startItem := menu.Add("Start Kernel")
	startItem.OnClick(func(_ *application.Context) {
		if err := startKernel(rt); err != nil {
			slog.Error("kernel start failed", "err", err)
		}
	})
	stopItem := menu.Add("Stop Kernel")
	stopItem.OnClick(func(_ *application.Context) {
		if err := stopKernel(rt); err != nil {
			slog.Error("kernel stop failed", "err", err)
		}
	})
	restartItem := menu.Add("Restart Kernel")
	restartItem.OnClick(func(_ *application.Context) {
		if err := restartKernel(rt); err != nil {
			slog.Error("kernel restart failed", "err", err)
		}
	})
	_ = startItem
	_ = stopItem
	_ = restartItem

	menu.AddSeparator()

	// 渲染进程卡死后手动重建 UI 的兜底入口。
	menu.Add("Reload UI").OnClick(func(_ *application.Context) {
		reloadMainWindow(app)
	})

	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		app.Quit()
	})

	tray.SetMenu(menu)
	tray.Run()
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
