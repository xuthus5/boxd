package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *application) connectionsView(c *ui.Context) {
	a.pageColumn(c, func() {
		pageHeading(c, "Connections", "Live connection snapshots from the same traffic tracker used by the Wails app.")
		ui.Row(c).Gap(12).Children(func() {
			metricCard(c, "Active", fmt.Sprint(len(a.model.connections)), "Open routed connections", c.Theme().Accent)
			metricCard(c, "Uploaded", formatBytes(totalConnectionUpload(a.model.connections)), "Across active connections", c.Theme().Warning)
			metricCard(c, "Downloaded", formatBytes(totalConnectionDownload(a.model.connections)), "Across active connections", c.Theme().Success)
		})
		if len(a.model.connections) == 0 {
			card(c, func() {
				emptyState(c, "No active connections", "Start the kernel and route traffic through boxd to see connections here.")
			})
			return
		}
		a.connectionsTable(c)
	})
}

func (a *application) connectionsTable(c *ui.Context) {
	columns := []ui.TableColumn{
		{Title: "Target", MinWidth: 190},
		{Title: "Outbound", Width: 130},
		{Title: "Rule", Width: 130},
		{Title: "Upload", Width: 100, Align: ui.End},
		{Title: "Download", Width: 110, Align: ui.End},
		{Title: "Duration", Width: 100},
		{Title: "Action", Width: 90},
	}
	ui.Table(c, &a.connsList, columns, len(a.model.connections), func(row, col int) {
		if row < 0 || row >= len(a.model.connections) {
			return
		}
		conn := a.model.connections[row]
		switch col {
		case 0:
			ui.Text(c, conn.Target).NoWrap()
		case 1:
			ui.Text(c, firstText(conn.Outbound, "—")).NoWrap()
		case 2:
			ui.Text(c, firstText(conn.Rule, "—")).NoWrap().TextColor(c.Theme().TextMuted)
		case 3:
			ui.Text(c, formatBytes(conn.Upload)).NoWrap()
		case 4:
			ui.Text(c, formatBytes(conn.Download)).NoWrap()
		case 5:
			ui.Text(c, conn.Duration).NoWrap().TextColor(c.Theme().TextMuted)
		case 6:
			if ui.Button(c, "Close").Key(fmt.Sprintf("close-conn-%d", conn.ID)).
				Disabled(a.model.busy).Clicked() {
				a.closeConnection(conn.ID)
			}
		}
	}).FillWidth().Height(500).Label("Connection list")
}

func totalConnectionUpload(connections []connectionView) int64 {
	var total int64
	for _, conn := range connections {
		total += conn.Upload
	}
	return total
}

func totalConnectionDownload(connections []connectionView) int64 {
	var total int64
	for _, conn := range connections {
		total += conn.Download
	}
	return total
}
