package main

import (
	"context"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func (a *application) ensureConfigSection(route string) {
	state, ok := a.configSectionForRoute(route)
	if !ok || state.attempted {
		return
	}
	a.loadConfigSection(state)
}

func (a *application) loadConfigSection(state *configSectionState) {
	if a.backend == nil || state == nil || state.loading {
		return
	}
	state.loading = true
	state.attempted = true
	a.invalidate()
	if a.win == nil {
		text, err := a.fetchConfigSection(state.spec.Key)
		a.applyLoadedConfigSection(state, text, err)
		return
	}
	go func() {
		text, err := a.fetchConfigSection(state.spec.Key)
		a.win.Update(func() { a.applyLoadedConfigSection(state, text, err) })
	}()
}

func (a *application) fetchConfigSection(section string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.LoadConfigSection(ctx, section)
}

func (a *application) applyLoadedConfigSection(state *configSectionState, text string, err error) {
	state.loading = false
	if err != nil {
		a.model.setActionError(err)
		return
	}
	state.editorState().Buffer.Reset(text)
	state.editorState().Buffer.MarkSaved()
	state.loaded = true
	a.model.actionError = ""
}

func (a *application) formatConfigSection(state *configSectionState) {
	body, err := formatConfigJSON(state.editorState().Buffer.String())
	if err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	state.editorState().Buffer.Reset(body)
	a.model.actionError = ""
	a.model.notice = state.spec.Title + " formatted"
	a.invalidate()
}

func (a *application) validateConfigSection(state *configSectionState) {
	body := state.editorState().Buffer.String()
	a.runAction("Validating "+strings.ToLower(state.spec.Title), func(ctx context.Context) error {
		return a.backend.ValidateConfigSection(ctx, state.spec.Key, body)
	}, state.spec.Title+" is valid")
}

func (a *application) applyConfigSection(state *configSectionState) {
	body := state.editorState().Buffer.String()
	a.runActionWithCallback("Applying "+strings.ToLower(state.spec.Title), func(ctx context.Context) error {
		return a.backend.ApplyConfigSection(ctx, state.spec.Key, body)
	}, state.spec.Title+" applied", func() {
		state.editorState().Buffer.MarkSaved()
		state.loaded = true
		a.requestRefresh()
	})
}

func (a *application) installConfigSectionDefaults(state *configSectionState) {
	if state == nil || !state.spec.CanDefaults {
		return
	}
	a.runActionWithCallback("Installing default "+strings.ToLower(state.spec.Title), func(ctx context.Context) error {
		return a.backend.InstallDefaultConfigSection(ctx, state.spec.Key)
	}, state.spec.Title+" defaults installed", func() {
		state.loaded = false
		state.attempted = true
		a.loadConfigSection(state)
	})
}

func (a *application) loadConfigDiagnostics() {
	if a.backend == nil || a.config.diagnosticsLoading {
		return
	}
	a.config.diagnosticsTried = true
	a.config.diagnosticsLoading = true
	if a.win == nil {
		diagnostics, err := a.fetchConfigDiagnostics()
		a.applyConfigDiagnostics(diagnostics, err)
		return
	}
	go func() {
		diagnostics, err := a.fetchConfigDiagnostics()
		a.win.Update(func() { a.applyConfigDiagnostics(diagnostics, err) })
	}()
}

func (a *application) fetchConfigDiagnostics() (model.ConfigDiagnostics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.ConfigDiagnostics(ctx)
}

func (a *application) applyConfigDiagnostics(diagnostics model.ConfigDiagnostics, err error) {
	a.config.diagnosticsLoading = false
	if err != nil {
		a.model.setActionError(err)
		return
	}
	a.config.diagnostics = diagnostics
	a.config.diagnosticsLoaded = true
}
