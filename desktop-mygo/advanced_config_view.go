package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xuthus5/boxd/internal/model"
)

const sectionEditorHeight = 500

func (a *application) configSectionView(c *ui.Context, route string) {
	state, ok := a.configSectionForRoute(route)
	if !ok {
		return
	}
	a.ensureConfigSection(route)
	a.pageColumn(c, func() {
		pageHeading(c, state.spec.Title, state.spec.Subtitle)
		if state.loading {
			card(c, func() {
				loadingRow(c, "Loading "+strings.ToLower(state.spec.Title)+"…")
			})
			return
		}
		a.configSectionVisualForm(c, state)
		renderConfigDiffCard(c, state.spec.Title+" diff", state.baseline,
			state.editorState().Buffer.String(), state.loaded)
		card(c, func() {
			a.configSectionToolbar(c, state)
			ui.Divider(c)
			editor.View(c, state.editorState()).Key("section-editor-" + state.spec.Key).
				FillWidth().Height(sectionEditorHeight)
		})
	})
}

func (a *application) configSectionToolbar(c *ui.Context, state *configSectionState) {
	t := c.Theme()
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
		sectionHeading(c, state.spec.Title+" JSON", "Edited as a native JSON section and merged into the shared config")
		ui.Spacer(c)
		if state.editorState().Buffer.Modified() {
			ui.Badge(c, "Unsaved").Background(t.Warning.Alpha(0.2)).TextColor(t.Warning)
		}
		if ui.Button(c, "Reload").Key("section-reload-" + state.spec.Key).Disabled(state.loading || a.model.busy).Clicked() {
			a.loadConfigSection(state)
		}
		if ui.Button(c, "Reset").Key("section-reset-" + state.spec.Key).Disabled(!configSectionCanReset(state)).Clicked() {
			a.resetConfigSection(state)
		}
		if ui.Button(c, "Format").Key("section-format-" + state.spec.Key).Disabled(state.loading).Clicked() {
			a.formatConfigSection(state)
		}
		if ui.Button(c, "Validate").Key("section-validate-" + state.spec.Key).Disabled(state.loading || a.model.busy).Clicked() {
			a.validateConfigSection(state)
		}
		if state.spec.CanDefaults && ui.Button(c, "Install defaults").Key("section-defaults-"+state.spec.Key).
			Disabled(state.loading || a.model.busy).Clicked() {
			a.installConfigSectionDefaults(state)
		}
		if ui.PrimaryButton(c, "Apply").Key("section-apply-" + state.spec.Key).Disabled(state.loading || a.model.busy).Clicked() {
			a.applyConfigSection(state)
		}
	})
}

func (a *application) configDiagnosticsCard(c *ui.Context) {
	if !a.config.diagnosticsTried && !a.config.diagnosticsLoading {
		a.loadConfigDiagnostics()
	}
	card(c, func() {
		sectionHeading(c, "Configuration diagnostics", "Read-only analysis of the persisted sing-box configuration")
		if a.config.diagnosticsLoading {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "Analyzing configuration…").TextColor(c.Theme().TextMuted)
			})
			return
		}
		if !a.config.diagnosticsLoaded {
			ui.Text(c, "Diagnostics are not available.").TextColor(c.Theme().TextMuted)
			return
		}
		a.configDiagnosticsSummary(c, a.config.diagnostics)
	})
}

func (a *application) configDiagnosticsSummary(c *ui.Context, diagnostics model.ConfigDiagnostics) {
	statusColor := c.Theme().Success
	if diagnostics.Status == model.ConfigDiagnosticsWarning {
		statusColor = c.Theme().Warning
	}
	if diagnostics.Status == model.ConfigDiagnosticsError {
		statusColor = c.Theme().Danger
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		statusPill(c, firstText(diagnostics.Status, "unknown"), statusColor)
		ui.Textf(c, "%d errors, %d warnings", diagnostics.Summary.Errors, diagnostics.Summary.Warnings).
			FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Spacer(c)
		if ui.Button(c, "Reanalyze").Key("diagnostics-refresh").Disabled(a.config.diagnosticsLoading).Clicked() {
			a.loadConfigDiagnostics()
		}
	})
	ui.Row(c).Gap(16).Children(func() {
		keyValue(c, "Inbounds", fmt.Sprint(diagnostics.Counts.Inbounds))
		keyValue(c, "Outbounds", fmt.Sprint(diagnostics.Counts.Outbounds))
		keyValue(c, "Route rules", fmt.Sprint(diagnostics.Counts.RouteRules))
		keyValue(c, "Rule sets", fmt.Sprint(diagnostics.Counts.RuleSets))
		keyValue(c, "DNS servers", fmt.Sprint(diagnostics.Counts.DNSServers))
	})
	if len(diagnostics.Issues) == 0 {
		ui.Text(c, "No configuration issues detected.").FontSize(12).TextColor(c.Theme().Success)
		return
	}
	for i, issue := range diagnostics.Issues {
		if i >= 6 {
			ui.Textf(c, "…and %d more issues", len(diagnostics.Issues)-i).FontSize(11).TextColor(c.Theme().TextMuted)
			break
		}
		diagnosticRow(c, issue)
	}
}

func diagnosticRow(c *ui.Context, issue model.ConfigDiagnostic) {
	color := c.Theme().Warning
	if issue.Severity == model.ConfigDiagnosticSeverityError {
		color = c.Theme().Danger
	}
	detail := issue.Detail
	if detail == "" {
		detail = issue.Code
	}
	if issue.Path != "" {
		detail = issue.Path + ": " + detail
	}
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Badge(c, strings.ToUpper(firstText(issue.Severity, "warning"))).Background(color.Alpha(0.15)).TextColor(color)
		ui.Text(c, detail).FontSize(12).Grow(1)
	})
}
