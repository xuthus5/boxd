package main

import "github.com/egoist/mygo/ui"

const windowControlSize = float32(30)

func (a *application) windowControls(c *ui.Context) {
	ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
		a.windowControl(c, "minimize-window", "Minimize window", iconMinimize, false, a.minimizeWindow)
		maximize := a.windowControl(c, "maximize-window", "Maximize or restore window", iconMaximize, false, a.toggleMaximizeWindow)
		maximize.MaximizeButton()
		a.windowControl(c, "close-window", "Close window", iconClose, true, a.closeWindow)
	})
}

func (a *application) windowControl(c *ui.Context, key, label string, icon *ui.SVG, danger bool, action func()) ui.Element {
	t := c.Theme()
	button := ui.ButtonBase(c).Key(key).Size(windowControlSize, windowControlSize).Radius(7).
		Label(label).Tooltip(label).Cursor(ui.CursorPointer).Background(ui.Transparent).TextColor(t.TextMuted)
	if button.Hovered() {
		button.Background(t.SurfaceHover).TextColor(t.Text)
	}
	if danger && button.Hovered() {
		button.Background(t.Danger.Alpha(0.12)).TextColor(t.Danger)
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(14, 14)
	})
	if button.Clicked() {
		action()
	}
	return button
}

func (a *application) minimizeWindow() {
	if a.win != nil {
		a.win.Minimize()
	}
}

func (a *application) toggleMaximizeWindow() {
	if a.win == nil {
		return
	}
	a.win.ToggleMaximize()
	a.invalidate()
}

func (a *application) closeWindow() {
	if a.win != nil {
		a.win.Close()
	}
}
