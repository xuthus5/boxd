package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

const (
	refreshInterval = time.Second
	actionTimeout   = 60 * time.Second
)

type application struct {
	backend         backend
	cfg             runtime.Config
	startupErr      error
	model           *appModel
	router          *ui.Router
	win             *mygo.Window
	tray            *mygo.Tray
	systemDark      bool
	quitting        bool
	autostartKernel bool
	shortcutReady   bool

	nodesList ui.ListState
	subsList  ui.ListState
	connsList ui.ListState
	logsList  ui.ListState

	refreshCancel context.CancelFunc
	refreshDone   chan struct{}
	refreshNow    chan struct{}
	refreshOnce   sync.Once
	config        configPageState
}

func newApplication(source backend, cfg runtime.Config, startupErr error, autostart bool) *application {
	app := &application{
		backend: source, cfg: cfg, startupErr: startupErr,
		model: newAppModel(), router: ui.NewRouter("/dashboard"),
		autostartKernel: autostart, refreshNow: make(chan struct{}, 1),
	}
	app.model.startupErr = startupErr
	app.nodesList.Key = func(i int) any {
		if i < 0 || i >= len(app.model.nodes) {
			return i
		}
		return app.model.nodes[i].Tag
	}
	app.subsList.Key = func(i int) any {
		if i < 0 || i >= len(app.model.subs) {
			return i
		}
		return app.model.subs[i].ID
	}
	app.connsList.Key = func(i int) any {
		if i < 0 || i >= len(app.model.connections) {
			return i
		}
		return app.model.connections[i].ID
	}
	app.logsList.FollowEnd = true
	return app
}

func (a *application) whenReady() {
	a.systemDark = mygo.Theme.IsDark()
	startHidden := mygo.App.WasOpenedAtLogin() || os.Getenv("BOXD_START_HIDDEN") == "1"
	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title: "boxd", Width: 1280, Height: 820, MinWidth: 960, MinHeight: 640,
		StateKey: "mygo-main", Hidden: startHidden, BackgroundColor: "#fafafa",
		Content: ui.View(a.view),
	})
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if a.quitting {
			return
		}
		if !a.shouldHideOnClose() {
			return
		}
		e.PreventDefault()
		a.win.Hide()
	})
	a.setupTray()
	a.setupGlobalShortcut()
	if !startHidden || a.tray == nil {
		a.win.Show()
		a.win.Focus()
	}
	a.model.appAutostart = mygo.App.OpenAtLogin()
	a.model.appAutostartSet = true
	if a.startupErr != nil {
		return
	}
	a.startRefresh()
	a.loadConfig()
	if a.autostartKernel {
		a.startKernelAction("Autostart")
	}
}

func (a *application) startRefresh() {
	if a.backend == nil {
		return
	}
	a.refreshOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		a.refreshCancel = cancel
		a.refreshDone = make(chan struct{})
		go a.refreshLoop(ctx)
	})
}

func (a *application) refreshLoop(ctx context.Context) {
	defer close(a.refreshDone)
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	a.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refresh(ctx)
		case <-a.refreshNow:
			a.refresh(ctx)
		}
	}
}

func (a *application) refresh(parent context.Context) {
	if a.backend == nil || parent.Err() != nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, actionTimeout)
	snap, err := a.backend.Snapshot(ctx)
	cancel()
	if parent.Err() != nil {
		return
	}
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		a.applyRefresh(snap, err, time.Now())
	})
}

func (a *application) applyRefresh(snap snapshot, err error, now time.Time) {
	if err != nil {
		a.model.stale = a.model.loaded
		a.model.warnings = []string{shortError(err)}
		return
	}
	a.model.applySnapshot(snap, now)
}

func (a *application) requestRefresh() {
	select {
	case a.refreshNow <- struct{}{}:
	default:
	}
}

func (a *application) stopRefresh() {
	if a.refreshCancel != nil {
		a.refreshCancel()
	}
	if a.refreshDone != nil {
		<-a.refreshDone
	}
}

func (a *application) runAction(label string, action func(context.Context) error, success string) {
	a.runActionWithRollback(label, action, success, nil)
}

func (a *application) runActionWithRollback(label string, action func(context.Context) error, success string, rollback func()) {
	if !a.beginAction(label) {
		return
	}
	if a.win == nil {
		a.executeAction(action, success, rollback)
		return
	}
	go func() {
		a.executeAction(action, success, rollback)
	}()
}

func (a *application) executeAction(action func(context.Context) error, success string, rollback func()) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	err := action(ctx)
	cancel()
	a.finishAction(err, success, rollback)
	a.requestRefresh()
}

func (a *application) beginAction(label string) bool {
	if a.backend == nil || a.model.busy {
		return false
	}
	a.model.busy = true
	a.model.busyLabel = label
	a.model.actionError = ""
	a.model.notice = ""
	a.invalidate()
	return true
}

func (a *application) finishAction(err error, success string, rollback func()) {
	apply := func() {
		a.model.busy = false
		a.model.busyLabel = ""
		a.model.setActionError(err)
		if err != nil {
			if rollback != nil {
				rollback()
			}
			return
		}
		a.model.notice = success
		a.notifyAction(success)
	}
	if a.win == nil {
		apply()
		return
	}
	a.win.Update(apply)
}

func (a *application) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

func (a *application) showWindow() {
	if a.win == nil {
		return
	}
	a.win.Show()
	a.win.Focus()
}

func (a *application) toggleWindow() {
	if a.win == nil {
		return
	}
	if a.win.IsVisible() {
		a.win.Hide()
		return
	}
	a.showWindow()
}

func (a *application) focusExisting() {
	a.showWindow()
}

func (a *application) shouldHideOnClose() bool {
	return a.tray != nil
}

func (a *application) handleURL(raw string) {
	if a.backend == nil {
		return
	}
	a.showWindow()
	link, err := parseDeepLink(raw)
	if err != nil {
		if a.win == nil {
			a.model.setActionError(err)
			return
		}
		a.win.Update(func() { a.model.setActionError(err) })
		return
	}
	if link == "" {
		return
	}
	a.runAction("Importing link", func(ctx context.Context) error {
		return a.backend.ImportLink(ctx, link)
	}, "Import link accepted")
}
