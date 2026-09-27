package main

import "log/slog"

// defaultTrayLanguage 托盘默认语言，与前端偏好默认值保持一致。
const defaultTrayLanguage = "zh"

// trayLabels 托盘 tooltip 与菜单文案。
type trayLabels struct {
	Tooltip       string
	Show          string
	StartKernel   string
	StopKernel    string
	RestartKernel string
	ReloadUI      string
	Quit          string
}

// trayLabelsByLanguage 托盘文案表，键与界面偏好的 language 取值一致。
var trayLabelsByLanguage = map[string]trayLabels{
	defaultTrayLanguage: {
		Tooltip:       "boxd — sing-box 控制面板",
		Show:          "显示主窗口",
		StartKernel:   "启动内核",
		StopKernel:    "停止内核",
		RestartKernel: "重启内核",
		ReloadUI:      "重载界面",
		Quit:          "退出",
	},
	"en": {
		Tooltip:       "boxd — sing-box control plane",
		Show:          "Show",
		StartKernel:   "Start Kernel",
		StopKernel:    "Stop Kernel",
		RestartKernel: "Restart Kernel",
		ReloadUI:      "Reload UI",
		Quit:          "Quit",
	},
}

// trayLabelsFor 返回指定语言的托盘文案，未知语言回退到默认语言。
func trayLabelsFor(language string) trayLabels {
	if labels, ok := trayLabelsByLanguage[language]; ok {
		return labels
	}
	return trayLabelsByLanguage[defaultTrayLanguage]
}

// trayLanguage 读取界面偏好中的语言；读取失败时回退到默认语言。
func trayLanguage(rt *desktopRuntime) string {
	if rt == nil || rt.svc == nil {
		return defaultTrayLanguage
	}
	prefs, err := rt.svc.Settings().GetUIPreferences(ctx())
	if err != nil {
		slog.Warn("tray language lookup failed", "err", err)
		return defaultTrayLanguage
	}
	return prefs.Language
}
