package main

import (
	"log"

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
			log.Printf("kernel start failed: %v", err)
		}
	})
	stopItem := menu.Add("Stop Kernel")
	stopItem.OnClick(func(_ *application.Context) {
		if err := stopKernel(rt); err != nil {
			log.Printf("kernel stop failed: %v", err)
		}
	})
	restartItem := menu.Add("Restart Kernel")
	restartItem.OnClick(func(_ *application.Context) {
		if err := restartKernel(rt); err != nil {
			log.Printf("kernel restart failed: %v", err)
		}
	})
	_ = startItem
	_ = stopItem
	_ = restartItem

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
