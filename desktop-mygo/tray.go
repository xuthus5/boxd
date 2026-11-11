package main

import (
	_ "embed"
	"log/slog"
	"runtime"

	"github.com/egoist/mygo"
)

//go:embed resources/icon.png
var trayIconPNG []byte

func (a *application) setupTray() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon: trayIconPNG, ToolTip: "boxd — native control plane",
	})
	if err != nil {
		slog.Warn("tray unavailable; keeping the main window available", "err", err)
		return
	}
	a.tray = tray
	menu := trayMenu(a)
	if runtime.GOOS == "linux" {
		tray.SetMenu(menu)
	} else {
		tray.OnClick(a.toggleWindow)
		tray.OnRightClick(func() {
			tray.SetMenu(menu)
			tray.PopUpMenu()
			tray.SetMenu(nil)
		})
	}
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		tray.Destroy()
	})
}

func trayMenu(a *application) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Show boxd", Click: func(*mygo.MenuItem, *mygo.Window) { a.showWindow() }},
		{Label: "Hide boxd", Click: func(*mygo.MenuItem, *mygo.Window) {
			if a.win != nil {
				a.win.Hide()
			}
		}},
		mygo.Separator(),
		{Label: "Start kernel", Click: func(*mygo.MenuItem, *mygo.Window) { a.startKernelAction("Starting kernel") }},
		{Label: "Stop kernel", Click: func(*mygo.MenuItem, *mygo.Window) { a.stopKernelAction() }},
		{Label: "Restart kernel", Click: func(*mygo.MenuItem, *mygo.Window) { a.restartKernelAction() }},
		mygo.Separator(),
		{Role: mygo.RoleQuit, Label: "Quit boxd"},
	})
}
