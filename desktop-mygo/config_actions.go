package main

import (
	"context"
)

func (a *application) loadConfig() {
	if a.backend == nil || a.config.loading {
		return
	}
	a.configEditorState()
	a.config.loading = true
	a.config.attempted = true
	a.model.actionError = ""
	a.invalidate()
	if a.win == nil {
		a.loadConfigSync()
		return
	}
	go func() {
		text, err := a.fetchConfig()
		a.win.Update(func() { a.applyLoadedConfig(text, err) })
	}()
}

func (a *application) loadConfigSync() {
	a.config.attempted = true
	text, err := a.fetchConfig()
	a.applyLoadedConfig(text, err)
}

func (a *application) fetchConfig() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	return a.backend.LoadConfig(ctx)
}

func (a *application) applyLoadedConfig(text string, err error) {
	a.config.loading = false
	if err != nil {
		a.model.setActionError(err)
		return
	}
	a.configEditorState().Buffer.Reset(text)
	a.configEditorState().Buffer.MarkSaved()
	a.config.baseline = text
	a.config.loaded = true
	a.model.actionError = ""
}

func (a *application) formatConfig() {
	body, err := formatConfigJSON(a.configEditorState().Buffer.String())
	if err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	a.configEditorState().Buffer.Reset(body)
	a.model.actionError = ""
	a.model.notice = "Configuration formatted"
	a.invalidate()
}

func (a *application) resetConfigDraft() {
	if !a.configCanReset() {
		return
	}
	a.configEditorState().Buffer.Reset(a.config.baseline)
	a.configEditorState().Buffer.MarkSaved()
	a.model.actionError = ""
	a.model.notice = "Configuration reset to the loaded version"
	a.invalidate()
}

func (a *application) validateConfig() {
	body := a.configEditorState().Buffer.String()
	a.runAction("Validating configuration", func(ctx context.Context) error {
		return a.backend.ValidateConfig(ctx, body)
	}, "Configuration is valid")
}

func (a *application) applyConfig() {
	body := a.configEditorState().Buffer.String()
	a.runAction("Applying configuration", func(ctx context.Context) error {
		if err := a.backend.ApplyConfig(ctx, body); err != nil {
			return err
		}
		a.configEditorState().Buffer.MarkSaved()
		a.config.baseline = body
		return nil
	}, "Configuration applied")
}
