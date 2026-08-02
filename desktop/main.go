package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

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
		OnShutdown: func() {
			if err := stopKernel(rt); err != nil {
				log.Printf("kernel stop on shutdown failed: %v", err)
			}
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
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

	// 前端 runtime 就绪后启动事件流推送（替代 SSE）。
	var streamer *EventStreamer
	if rt.svc != nil {
		window.OnWindowEvent(events.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
			streamer = NewEventStreamer(app, rt)
			streamer.Start()
		})
	}

	if err := app.Run(); err != nil {
		log.Fatalf("app run failed: %v", err)
	}
	if streamer != nil {
		streamer.Stop()
	}
}

// registerServices 将 service.ServiceSet 的各子服务注册为 Wails 可绑定服务。
func registerServices(rt *desktopRuntime) []application.Service {
	if rt.svc == nil {
		return nil
	}
	return []application.Service{
		application.NewService(newBoxdConfigService(rt)),
		application.NewService(newBoxdServiceControlService(rt)),
		application.NewService(newBoxdSettingsService(rt)),
		application.NewService(newBoxdAuthService(rt)),
		application.NewService(newBoxdStatsService(rt)),
		application.NewService(newBoxdBridgeService(rt)),
	}
}
