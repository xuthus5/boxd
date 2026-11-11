package main

import (
	"log/slog"

	"github.com/egoist/mygo"
)

const appShortcut = "CmdOrCtrl+Shift+B"

func (a *application) setupGlobalShortcut() {
	if err := mygo.GlobalShortcut.Register(appShortcut, a.toggleWindow); err != nil {
		slog.Warn("global shortcut unavailable", "shortcut", appShortcut, "err", err)
		return
	}
	a.shortcutReady = true
}

func (a *application) clearGlobalShortcut() {
	if !a.shortcutReady {
		return
	}
	mygo.GlobalShortcut.Unregister(appShortcut)
	a.shortcutReady = false
}

func (a *application) notifyAction(message string) {
	if message == "" || a.win == nil || a.win.IsFocused() || !mygo.NotificationsSupported() {
		return
	}
	go func() {
		notification := mygo.NewNotification(mygo.NotificationOptions{
			Title: "boxd", Body: message,
		})
		notification.OnClick(a.showWindow)
		if err := notification.Show(); err != nil {
			slog.Warn("desktop notification failed", "err", err)
		}
	}()
}
