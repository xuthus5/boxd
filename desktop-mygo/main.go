package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/egoist/mygo"

	"github.com/xuthus5/boxd/internal/core"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "boxd:", err)
		os.Exit(1)
	}
}

func run() error {
	mygo.App.SetName("boxd")
	mygo.App.SetVersion(core.Version)
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	cfg, err := runtime.LoadConfig()
	if err != nil {
		return fmt.Errorf("load desktop config: %w", err)
	}
	rt, openErr := runtime.Open(cfg)
	autostart := rt != nil && rt.KernelAutostart() && !cfg.SkipAutostart
	var source backend
	if rt != nil {
		source = newRuntimeBackend(rt)
	}
	app := newApplication(source, cfg, openErr, autostart)
	mygo.App.OnSecondInstance(func([]string, string) { app.focusExisting() })
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		app.quitting = true
		app.clearGlobalShortcut()
	})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			app.showWindow()
		}
	})
	mygo.App.OnOpenURL(app.handleURL)
	if err := mygo.App.RegisterURLScheme("boxd"); err != nil {
		slog.Warn("register boxd URL scheme failed", "err", err)
	}
	mygo.App.WhenReady(app.whenReady)
	runErr := mygo.App.Run()
	closeErr := app.closeRuntime(rt)
	return errors.Join(runErr, closeErr)
}

func (a *application) closeRuntime(rt *runtime.Runtime) error {
	a.stopRefresh()
	if rt == nil {
		return nil
	}
	if err := rt.Close(); err != nil {
		slog.Error("runtime close failed", "err", err)
		return err
	}
	return nil
}
