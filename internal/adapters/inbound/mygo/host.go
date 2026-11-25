package mygo

import (
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// HostOptions are presentation and build metadata owned by the desktop
// composition root. The loader itself lives in cmd/boxd-desktop, where storage
// and kernel drivers are wired.
type HostOptions struct {
	Version       string
	Commit        string
	BuiltAt       string
	Dark          bool
	OnThemeChange func(bool) error
}

// Host owns the single native window and optional tray icon. It never opens an
// HTTP listener and never imports the Web authentication package.
type Host struct {
	controller *Controller
	options    HostOptions

	window *mygo.Window
	tray   *mygo.Tray
	close  sync.Once
}

func NewHost(loader Loader, options HostOptions) *Host {
	return &Host{controller: NewController(loader, ControllerOptions{
		Dark:          options.Dark,
		OnThemeChange: options.OnThemeChange,
	}), options: options}
}

func (h *Host) Controller() *Controller { return h.controller }

// Open creates the main window. It must be called after MyGo is ready.
func (h *Host) Open() {
	if h.window != nil && !h.window.IsDestroyed() {
		h.Show()
		return
	}
	h.window = mygo.NewWindow(mygo.WindowOptions{
		Title:           "BOXD",
		Width:           1320,
		Height:          860,
		MinWidth:        960,
		MinHeight:       640,
		TitleBarStyle:   mygo.TitleBarHidden,
		TitleBarHeight:  38,
		BackgroundColor: "#f6f7fb",
		StateKey:        "boxd-main",
		UseContentSize:  true,
		Content:         ui.View(h.controller.View),
	})
	h.controller.Attach(h.window)
	if icon, err := iconAssets.ReadFile("assets/icon-256.png"); err == nil {
		_ = h.window.SetIcon(icon)
	}
	h.setUpMenu()
	h.setUpTray()
	h.controller.Boot()
}

func (h *Host) Show() {
	if h.window == nil || h.window.IsDestroyed() {
		h.Open()
		return
	}
	h.window.Show()
	h.window.Focus()
}

func (h *Host) Hide() {
	if h.window != nil && !h.window.IsDestroyed() {
		h.window.Hide()
	}
}

func (h *Host) setUpMenu() {
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Role: mygo.RoleFileMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	}))
}

func (h *Host) setUpTray() {
	icon, err := iconAssets.ReadFile("assets/icon-64.png")
	if err != nil {
		return
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "打开 BOXD", Click: func(*mygo.MenuItem, *mygo.Window) { h.Show() }},
		{Label: "隐藏窗口", Click: func(*mygo.MenuItem, *mygo.Window) { h.Hide() }},
		mygo.Separator(),
		{Label: "刷新控制面", Click: func(*mygo.MenuItem, *mygo.Window) { h.controller.refresh(true) }},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	})
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:      "boxd-main",
		Icon:    icon,
		ToolTip: "BOXD · 网络控制平面",
		Menu:    menu,
	})
	if err != nil {
		return
	}
	h.tray = tray
}

// Close releases the control plane and tray. It is safe during application
// shutdown and from repeated callbacks.
func (h *Host) Close() {
	h.close.Do(func() {
		if h.tray != nil {
			h.tray.Destroy()
			h.tray = nil
		}
		h.controller.Close()
	})
}
