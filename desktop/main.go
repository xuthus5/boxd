package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// globalApp 保存应用实例，供单实例二次启动回调引用。
var globalApp *application.App

func main() {
	cfg := parseDesktopConfig()
	rt, err := initRuntime(cfg)
	if err != nil {
		log.Fatalf("runtime init failed: %v", err)
	}
	defer func() {
		if err := rt.close(); err != nil {
			log.Printf("runtime close failed: %v", err)
		}
	}()

	app := application.New(application.Options{
		Name:        "boxd",
		Description: "sing-box control plane desktop app",
		Services:    registerServices(rt),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Linux: application.LinuxOptions{
			ProgramName: "boxd-desktop",
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.boxd.desktop",
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				// 二次启动时聚焦已有窗口。
				focusMainWindow(globalApp)
			},
		},
		OnShutdown: func() {
			if err := stopKernel(rt); err != nil {
				log.Printf("kernel stop on shutdown failed: %v", err)
			}
		},
	})
	globalApp = app

	// 注入原生能力依赖（AutostartManager）。
	rt.autostart = app.Autostart

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "boxd",
		Width:            1280,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})
	_ = window

	setupTray(app, rt)

	// 隐私/极简模式：关闭窗口时隐藏到托盘而非退出。
	setupPrivacyMode(window)

	// 全局快捷键：显示/隐藏窗口。
	setupGlobalShortcuts(app)

	// 前端 runtime 就绪后启动事件流推送（替代 SSE）。
	var streamer *EventStreamer
	if rt.svc != nil {
		window.OnWindowEvent(events.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
			streamer = NewEventStreamer(app, rt)
			streamer.Start()
		})
	}

	// URL Scheme 深链（boxd://import?link=...）。
	NewURLHandler(app, rt).Register()

	if err := app.Run(); err != nil {
		log.Fatalf("app run failed: %v", err)
	}
	if streamer != nil {
		streamer.Stop()
	}
}

// focusMainWindow 聚焦主窗口（二次启动时使用）。
func focusMainWindow(app *application.App) {
	if app == nil {
		return
	}
	if window, ok := app.Window.GetByName("main"); ok {
		window.Show()
		window.Focus()
	}
}

// setupPrivacyMode 关闭窗口时隐藏到托盘而非退出。
func setupPrivacyMode(window *application.WebviewWindow) {
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		// 阻止关闭，隐藏窗口到托盘。
		e.Cancel()
		window.Hide()
	})
}

// setupGlobalShortcuts 注册全局快捷键：Ctrl+Shift+B 显示/隐藏窗口。
func setupGlobalShortcuts(app *application.App) {
	if app.GlobalShortcut == nil {
		return
	}
	_ = app.GlobalShortcut.Register("Ctrl+Shift+B", func() {
		if window, ok := app.Window.GetByName("main"); ok {
			if window.IsMinimised() || !window.IsVisible() {
				window.Show()
				window.Focus()
			} else {
				window.Hide()
			}
		}
	})
}

// registerServices 将 service.ServiceSet 的各子服务注册为 Wails 可绑定服务。
func registerServices(rt *desktopRuntime) []application.Service {
	if rt.svc == nil {
		return nil
	}
	notificationService := notifications.New()
	registeredNotificationService = notificationService
	return []application.Service{
		application.NewService(newBoxdConfigService(rt)),
		application.NewService(newBoxdServiceControlService(rt)),
		application.NewService(newBoxdSettingsService(rt)),
		application.NewService(newBoxdAuthService(rt)),
		application.NewService(newBoxdStatsService(rt)),
		application.NewService(newBoxdBridgeService(rt)),
		application.NewService(NewNativeCapabilities(rt)),
		application.NewService(NewDialogService()),
		application.NewService(notificationService),
	}
}
