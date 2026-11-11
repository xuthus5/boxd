package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/core"
)

var (
	themeOptions    = []string{themeSystem, themeLight, themeDark}
	languageOptions = []string{languageChinese, languageEnglish}
)

func (a *application) settingsView(c *ui.Context) {
	a.pageColumn(c, func() {
		pageHeading(c, "Settings", "Appearance, startup behavior, test URL and runtime paths.")
		ui.Row(c).FillWidth().Gap(12).Children(func() {
			a.appearanceSettings(c)
			a.startupSettings(c)
		})
		ui.Row(c).FillWidth().Gap(12).Children(func() {
			a.networkSettings(c)
			a.runtimeInfo(c)
		})
	})
}

func (a *application) appearanceSettings(c *ui.Context) {
	card(c, func() {
		sectionHeading(c, "Appearance", "Stored in the shared UI preferences")
		theme := a.model.prefs.Theme
		ui.Text(c, "Theme").FontSize(12).TextColor(c.Theme().TextMuted)
		if ui.Select(c.Key("theme-select"), &theme, themeOptions).Changed() {
			a.updateUIPreferences(prefsWithTheme(a.model.prefs, theme))
		}
		language := a.model.prefs.Language
		ui.Text(c, "Language").FontSize(12).TextColor(c.Theme().TextMuted)
		if ui.Select(c.Key("language-select"), &language, languageOptions).Changed() {
			next := a.model.prefs
			next.Language = language
			a.updateUIPreferences(next)
		}
		ui.Text(c, "The preference is persisted for future localized native copy; this prototype keeps English labels.").
			FontSize(11).TextColor(c.Theme().TextMuted)
	}).Grow(1)
}

func (a *application) startupSettings(c *ui.Context) {
	card(c, func() {
		sectionHeading(c, "Startup", "Kernel and desktop login behavior")
		kernel := a.model.kernelAutostart
		if ui.Switch(c.Key("kernel-autostart"), &kernel).Changed() {
			a.setKernelAutostart(kernel)
		}
		ui.Text(c, "Start kernel when boxd launches").FontSize(12)
		appLogin := a.model.appAutostart
		if ui.Switch(c.Key("app-autostart"), &appLogin).Changed() {
			a.setAppAutostart(appLogin)
		}
		ui.Text(c, "Open boxd when the user logs in").FontSize(12)
		ui.Text(c, "Login startup is provided by MyGo and remains available even if the tray is unavailable.").
			FontSize(11).TextColor(c.Theme().TextMuted)
	}).Grow(1)
}

func (a *application) networkSettings(c *ui.Context) {
	card(c, func() {
		sectionHeading(c, "Network test", "URL used by node latency checks")
		ui.TextInput(c.Key("test-url"), &a.model.testURLDraft).Label("Test URL")
		if ui.PrimaryButton(c, "Save test URL").Key("save-test-url").Disabled(a.model.busy).Clicked() {
			a.setTestURL(a.model.testURLDraft)
		}
		ui.Text(c, "The backend validates HTTP and HTTPS URLs before saving.").
			FontSize(11).TextColor(c.Theme().TextMuted)
	}).Grow(1)
}

func (a *application) runtimeInfo(c *ui.Context) {
	card(c, func() {
		sectionHeading(c, "Runtime", "Read-only paths and versions")
		keyValue(c, "App version", core.Version)
		keyValue(c, "Kernel version", firstText(a.model.status.Version, "Unavailable"))
		keyValue(c, "Data directory", a.cfg.DataDir)
		keyValue(c, "Config", a.cfg.ConfigPath)
		keyValue(c, "Remote mode", remoteModeText(a.cfg.Mode))
		keyValue(c, "Warnings", fmt.Sprint(len(a.model.warnings)))
	}).Grow(1)
}

func remoteModeText(mode string) string {
	if mode == "embedded" || mode == "" {
		return "Embedded (shared ~/.boxd data)"
	}
	return mode + " (not implemented in this prototype)"
}
