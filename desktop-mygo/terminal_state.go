package main

import (
	"context"

	"github.com/xuthus5/boxd/internal/core"
)

type terminalPageState struct {
	status    core.SetupStatus
	loaded    bool
	loading   bool
	attempted bool
}

func (a *application) ensureTerminal() {
	if a.terminal.attempted {
		return
	}
	a.loadTerminal()
}

func (a *application) loadTerminal() {
	if a.backend == nil || a.terminal.loading {
		return
	}
	a.terminal.loading = true
	a.terminal.attempted = true
	a.invalidate()
	if a.win == nil {
		status, err := a.fetchSetupStatus()
		a.applyTerminalStatus(status, err)
		return
	}
	go func() {
		status, err := a.fetchSetupStatus()
		a.win.Update(func() { a.applyTerminalStatus(status, err) })
	}()
}

func (a *application) fetchSetupStatus() (core.SetupStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.SetupStatus(ctx)
}

func (a *application) applyTerminalStatus(status core.SetupStatus, err error) {
	a.terminal.loading = false
	if err != nil {
		a.model.setActionError(err)
		return
	}
	a.terminal.status = status
	a.terminal.loaded = true
	a.model.actionError = ""
}

func (a *application) copyTerminalCommand(label, command string) {
	if err := mygoClipboardWrite(command); err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	a.model.actionError = ""
	a.model.notice = label + " command copied"
	a.invalidate()
}
