package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/core"
)

var logFilters = []string{"All", "Info", "Warn", "Error"}

func (a *application) logsView(c *ui.Context) {
	a.pageColumn(c, func() {
		pageHeading(c, "Logs", "Kernel and application log entries share one bounded native buffer.")
		card(c, func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				sectionHeading(c, "Recent activity", fmt.Sprintf("%d entries kept in memory", len(a.model.logs)))
				ui.Spacer(c)
				if ui.Button(c, "Copy").Key("logs-copy").Disabled(len(a.model.visibleLogs()) == 0).Clicked() {
					a.copyLogs()
				}
				if ui.Button(c, "Export").Key("logs-export").Disabled(len(a.model.visibleLogs()) == 0).Clicked() {
					a.exportLogs()
				}
				ui.Segmented(c.Key("log-filter"), &a.model.logFilter, logFilters...)
			})
			logs := a.model.visibleLogs()
			if len(logs) == 0 {
				emptyState(c, "No matching log entries", "Change the filter or wait for kernel activity.")
				return
			}
			a.logsList.Key = func(i int) any {
				if i < 0 || i >= len(logs) {
					return i
				}
				return logs[i].Timestamp.String() + logs[i].Message
			}
			ui.List(c, &a.logsList, len(logs), func(i int) {
				logRow(c, logs[i])
			}).FillWidth().Height(520).Label("Log entries")
		})
	})
}

func logRow(c *ui.Context, entry core.LogEntry) {
	t := c.Theme()
	color := levelColor(c, entry.Level)
	ui.Row(c).FillWidth().Padding(5, 8).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border.Alpha(0.55)).Children(func() {
		ui.Text(c, formatShortTime(entry.Timestamp)).Width(66).FontSize(11).TextColor(t.TextMuted).NoWrap()
		ui.Text(c, strings.ToUpper(firstText(entry.Level, "info"))).Width(52).FontSize(10).
			Bold().TextColor(color).NoWrap()
		ui.Text(c, entry.Message).Grow(1).FontSize(12).NoWrap()
	})
}

func levelColor(c *ui.Context, level string) ui.Color {
	switch strings.ToLower(level) {
	case "error", "fatal", "panic":
		return c.Theme().Danger
	case "warn", "warning":
		return c.Theme().Warning
	case "debug", "trace":
		return c.Theme().TextMuted
	default:
		return c.Theme().Accent
	}
}
