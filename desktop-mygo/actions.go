package main

import (
	"context"

	"github.com/egoist/mygo"

	"github.com/xuthus5/boxd/internal/model"
)

func (a *application) startKernelAction(label string) {
	if label == "" {
		label = "Starting kernel"
	}
	a.runAction(label, func(ctx context.Context) error {
		return a.backend.StartKernel(ctx)
	}, "Kernel started")
}

func (a *application) stopKernelAction() {
	a.runAction("Stopping kernel", func(ctx context.Context) error {
		return a.backend.StopKernel(ctx)
	}, "Kernel stopped")
}

func (a *application) restartKernelAction() {
	a.runAction("Restarting kernel", func(ctx context.Context) error {
		return a.backend.RestartKernel(ctx)
	}, "Kernel restarted")
}

func (a *application) refreshSubscription(id string) {
	a.runAction("Refreshing subscription", func(ctx context.Context) error {
		return a.backend.RefreshSubscription(ctx, id)
	}, "Subscription refreshed")
}

func (a *application) closeConnection(id int64) {
	a.runAction("Closing connection", func(ctx context.Context) error {
		return a.backend.CloseConnection(ctx, id)
	}, "Connection closed")
}

func (a *application) setKernelAutostart(enabled bool) {
	if a.model.busy {
		return
	}
	previous := a.model.kernelAutostart
	if previous == enabled {
		return
	}
	a.model.kernelAutostart = enabled
	a.runActionWithRollback("Saving startup setting", func(ctx context.Context) error {
		return a.backend.SetKernelAutostart(ctx, enabled)
	}, "Kernel autostart updated", func() {
		a.model.kernelAutostart = previous
	})
}

func (a *application) setUIPreferences(previous, prefs model.UIPreferences) {
	a.runActionWithRollback("Saving preferences", func(ctx context.Context) error {
		return a.backend.SetUIPreferences(ctx, prefs)
	}, "Preferences saved", func() {
		a.model.prefs = previous
	})
}

func (a *application) updateUIPreferences(prefs model.UIPreferences) {
	if a.model.busy {
		return
	}
	previous := a.model.prefs
	if previous == prefs {
		return
	}
	a.model.prefs = prefs
	a.setUIPreferences(previous, prefs)
}

func (a *application) setTestURL(url string) {
	if a.model.busy {
		return
	}
	previous := a.model.testURL
	if previous == url {
		return
	}
	a.model.testURL = url
	a.runActionWithRollback("Saving test URL", func(ctx context.Context) error {
		return a.backend.SetTestURL(ctx, url)
	}, "Test URL saved", func() {
		a.model.testURL = previous
	})
}

func (a *application) setAppAutostart(enabled bool) {
	previous := a.model.appAutostart
	if previous == enabled {
		return
	}
	a.model.appAutostart = enabled
	if !a.beginAction("Saving login item") {
		a.model.appAutostart = previous
		return
	}
	go func() {
		err := mygo.App.SetOpenAtLogin(enabled)
		a.finishAction(err, "Login item updated", func() {
			a.model.appAutostart = previous
		})
	}()
}
