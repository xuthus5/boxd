package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xuthus5/boxd/internal/model"
)

const historyPreviewHeight = 440

func (a *application) configHistoryView(c *ui.Context) {
	a.ensureConfigHistory()
	a.pageColumn(c, func() {
		pageHeading(c, "Config history", "Inspect validated, applied and failed configuration changes, then restore a snapshot.")
		card(c, func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				sectionHeading(c, "Apply timeline", fmt.Sprintf("%d recorded events", len(a.configHistory.events)))
				ui.Spacer(c)
				if ui.Button(c, "Export").Key("history-export").Disabled(len(a.configHistory.events) == 0).Clicked() {
					a.exportConfigHistory()
				}
				if ui.Button(c, "Refresh").Key("history-refresh").Disabled(a.configHistory.loading || a.model.busy).Clicked() {
					a.refreshConfigHistory()
				}
			})
			if a.configHistory.loading {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c)
					ui.Text(c, "Loading configuration history…").TextColor(c.Theme().TextMuted)
				})
				return
			}
			if len(a.configHistory.events) == 0 {
				emptyState(c, "No configuration history", "Apply or validate a configuration to create the first snapshot.")
				return
			}
			a.configHistoryTable(c)
		})
	})
}

func (a *application) configHistoryTable(c *ui.Context) {
	columns := []ui.TableColumn{
		{Title: "Applied", Width: 165},
		{Title: "Source", MinWidth: 120},
		{Title: "Status", Width: 125},
		{Title: "Size", Width: 80, Align: ui.End},
		{Title: "Detail", MinWidth: 160},
		{Title: "Actions", Width: 190},
	}
	ui.Table(c, &a.historyList, columns, len(a.configHistory.events), func(row, col int) {
		if row < 0 || row >= len(a.configHistory.events) {
			return
		}
		event := a.configHistory.events[row]
		switch col {
		case 0:
			ui.Text(c, event.AppliedAt.Format("2006-01-02 15:04:05")).NoWrap()
		case 1:
			ui.Text(c, firstText(event.Source, "unknown")).NoWrap()
		case 2:
			ui.Text(c, historyStatusLabel(event)).NoWrap().TextColor(historyStatusColor(c, event.Status))
		case 3:
			ui.Textf(c, "%d B", event.Size).NoWrap()
		case 4:
			ui.Text(c, firstText(event.Error, shortHash(event.Hash))).NoWrap().TextColor(c.Theme().TextMuted)
		case 5:
			a.configHistoryActions(c, event)
		}
	}).FillWidth().Height(480).Label("Configuration history")
}

func (a *application) configHistoryActions(c *ui.Context, event model.ConfigApplyEvent) {
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, "View").Key("history-view-" + event.ID).Disabled(a.configHistory.previewLoad).Clicked() {
			a.openConfigSnapshot(event)
		}
		if ui.Button(c, "Restore").Key("history-restore-" + event.ID).Disabled(!event.Restorable || a.model.busy).Clicked() {
			a.askRestoreConfigSnapshot(event.ID)
		}
	})
}

func (a *application) configHistoryDialog(c *ui.Context) {
	if !a.configHistory.previewOpen {
		return
	}
	preview := a.configHistory.previewState()
	preview.ReadOnly = true
	ui.Modal(c, &a.configHistory.previewOpen, func() {
		dialogPanel(c, "Configuration snapshot", "Source: "+firstText(a.configHistory.previewSource, "unknown"), func() {
			editor.View(c, preview).Key("history-preview").FillWidth().Height(historyPreviewHeight)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "Close").Key("history-preview-close").Clicked() {
					a.configHistory.previewOpen = false
				}
				if ui.PrimaryButton(c, "Restore").Key("history-preview-restore").Disabled(a.model.busy).Clicked() {
					a.askRestoreConfigSnapshot(a.configHistory.previewID)
				}
			})
		})
	})
}

func historyStatusLabel(event model.ConfigApplyEvent) string {
	if event.Current {
		return firstText(event.Status, "current") + " · current"
	}
	return firstText(event.Status, "unknown")
}

func historyStatusColor(c *ui.Context, status string) ui.Color {
	switch {
	case strings.Contains(status, "failed"), strings.Contains(status, "error"):
		return c.Theme().Danger
	case status == "applied", status == "validated":
		return c.Theme().Success
	case status == "rolled_back":
		return c.Theme().Warning
	default:
		return c.Theme().TextMuted
	}
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
