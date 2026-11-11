package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/model"
)

func (a *application) ruleSetsView(c *ui.Context) {
	a.ensureRuleSets()
	a.pageColumn(c, func() {
		pageHeading(c, "Rule sets", "Inspect kernel rule-set sources, update cached files and configure local auto-update.")
		a.ruleSetAutoUpdateCard(c)
		card(c, func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				sectionHeading(c, "Rule-set sources", fmt.Sprintf("%d configured entries", len(a.ruleSets.items)))
				ui.Spacer(c)
				if ui.Button(c, "Refresh").Key("ruleset-refresh").Disabled(a.ruleSets.loading || a.model.busy).Clicked() {
					a.refreshRuleSets()
				}
				if ui.PrimaryButton(c, "Update all").Key("ruleset-update-all").Disabled(a.ruleSets.loading || a.model.busy).Clicked() {
					a.updateRuleSets(nil)
				}
			})
			if a.ruleSets.loading {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c)
					ui.Text(c, "Loading rule sets…").TextColor(c.Theme().TextMuted)
				})
				return
			}
			if len(a.ruleSets.items) == 0 {
				emptyState(c, "No rule sets configured", "Install route defaults or add rule-set entries in the Route section.")
				return
			}
			a.ruleSetTable(c)
		})
	})
}

func (a *application) ruleSetAutoUpdateCard(c *ui.Context) {
	card(c, func() {
		sectionHeading(c, "Local auto-update", "Only boxd-managed local rule-set files are refreshed by this schedule")
		if !a.ruleSets.autoLoaded {
			ui.Text(c, "Auto-update settings are not loaded yet.").FontSize(12).TextColor(c.Theme().TextMuted)
			return
		}
		enabled := a.ruleSets.draftAuto.Enabled
		if ui.Switch(c.Key("ruleset-auto-enabled"), &a.ruleSets.draftAuto.Enabled).Changed() {
			a.setRuleSetAutoEnabled(a.ruleSets.draftAuto.Enabled)
		}
		ui.Text(c, "Enable scheduled local rule-set refresh").FontSize(12)
		ui.Row(c).Gap(10).AlignItems(ui.End).Children(func() {
			ui.Field(c, "Interval", func() {
				ui.TextInput(c.Key("ruleset-auto-interval"), &a.ruleSets.draftAuto.Interval).
					Placeholder("24h").Disabled(a.model.busy)
			}).Width(220)
			if ui.PrimaryButton(c, "Save interval").Key("ruleset-auto-save").Disabled(a.model.busy).Clicked() {
				a.saveRuleSetAutoInterval(a.ruleSets.draftAuto.Interval)
			}
			if a.ruleSets.savedAuto.Enabled != enabled {
				ui.Badge(c, "Saving").Background(c.Theme().Warning.Alpha(0.15)).TextColor(c.Theme().Warning)
			}
		})
		ui.Text(c, "Remote rule sets remain managed by the kernel's own update_interval.").
			FontSize(11).TextColor(c.Theme().TextMuted)
	})
}

func (a *application) ruleSetTable(c *ui.Context) {
	columns := []ui.TableColumn{
		{Title: "Tag", MinWidth: 140},
		{Title: "Type", Width: 90},
		{Title: "Format", Width: 90},
		{Title: "Updated", Width: 150},
		{Title: "Size", Width: 90, Align: ui.End},
		{Title: "Status", MinWidth: 150},
		{Title: "Actions", Width: 120},
	}
	ui.Table(c, &a.ruleSetsList, columns, len(a.ruleSets.items), func(row, col int) {
		if row < 0 || row >= len(a.ruleSets.items) {
			return
		}
		item := a.ruleSets.items[row]
		switch col {
		case 0:
			ui.Text(c, item.Tag).NoWrap().Bold()
		case 1:
			ui.Text(c, firstText(item.Type, "—")).NoWrap()
		case 2:
			ui.Text(c, firstText(item.Format, "—")).NoWrap()
		case 3:
			ui.Text(c, ruleSetUpdated(item)).NoWrap()
		case 4:
			ui.Textf(c, "%d B", item.FileSize).NoWrap()
		case 5:
			ui.Text(c, ruleSetStatus(item)).NoWrap().TextColor(ruleSetStatusColor(c, item))
		case 6:
			if ui.Button(c, "Update").Key("ruleset-update-" + item.Tag).Disabled(!item.Updatable || a.model.busy).Clicked() {
				a.updateRuleSets([]string{item.Tag})
			}
		}
	}).FillWidth().Height(380).Label("Rule sets")
}

func ruleSetUpdated(item model.RuleSetStatusItem) string {
	if item.LastUpdated == nil || item.LastUpdated.IsZero() {
		return "Never"
	}
	return item.LastUpdated.Local().Format("2006-01-02 15:04")
}

func ruleSetStatus(item model.RuleSetStatusItem) string {
	switch {
	case item.Note != "":
		return item.Note
	case item.Updatable:
		return "Updatable"
	case item.KernelManaged:
		return "Kernel managed"
	case item.Builtin:
		return "Built in"
	default:
		return "Static"
	}
}

func ruleSetStatusColor(c *ui.Context, item model.RuleSetStatusItem) ui.Color {
	if strings.Contains(strings.ToLower(item.Note), "error") || strings.Contains(strings.ToLower(item.Note), "invalid") {
		return c.Theme().Danger
	}
	if item.Updatable {
		return c.Theme().Accent
	}
	return c.Theme().TextMuted
}
