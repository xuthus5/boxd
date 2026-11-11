package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *application) connectionsView(c *ui.Context) {
	connections := a.filteredConnections()
	a.pageColumn(c, func() {
		pageHeading(c, "Connections", "Inspect active routed connections, filter traffic and close connections safely.")
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.SearchField(c, &a.connFilter).Placeholder("Filter target, outbound or rule").Label("Filter connections").Width(320)
			ui.Textf(c, "%d matched", len(connections)).FontSize(12).TextColor(c.Theme().TextMuted)
			ui.Spacer(c)
			if ui.Button(c, "Close filtered").Key("connections-close-filtered").
				Disabled(a.model.busy || a.connFilter == "" || len(connections) == 0).Clicked() {
				a.askCloseConnections(connections, true)
			}
			if ui.Button(c, "Close all").Key("connections-close-all").
				Disabled(a.model.busy || len(a.model.connections) == 0).Clicked() {
				a.askCloseConnections(nil, false)
			}
		})
		ui.Row(c).Gap(12).Children(func() {
			metricCard(c, "Active", fmt.Sprint(len(a.model.connections)), "Open routed connections", c.Theme().Accent)
			metricCard(c, "Uploaded", formatBytes(totalConnectionUpload(connections)), "Across matched connections", c.Theme().Warning)
			metricCard(c, "Downloaded", formatBytes(totalConnectionDownload(connections)), "Across matched connections", c.Theme().Success)
		})
		if len(a.model.connections) == 0 {
			card(c, func() {
				emptyState(c, "No active connections", "Start the kernel and route traffic through boxd to see connections here.")
			})
			return
		}
		if len(connections) == 0 {
			card(c, func() {
				emptyState(c, "No matching connections", "Clear or change the filter to see active connections.")
			})
			return
		}
		a.connectionsTable(c, connections)
	})
}

func (a *application) filteredConnections() []connectionView {
	query := strings.ToLower(strings.TrimSpace(a.connFilter))
	if query == "" {
		return a.model.connections
	}
	out := make([]connectionView, 0, len(a.model.connections))
	for _, conn := range a.model.connections {
		haystack := strings.ToLower(strings.Join([]string{conn.Target, conn.Outbound, conn.Rule}, " "))
		if strings.Contains(haystack, query) {
			out = append(out, conn)
		}
	}
	return out
}

func (a *application) connectionsTable(c *ui.Context, connections []connectionView) {
	a.connsList.Key = func(i int) any {
		if i < 0 || i >= len(connections) {
			return i
		}
		return connections[i].ID
	}
	columns := []ui.TableColumn{
		{Title: "Target", MinWidth: 190},
		{Title: "Outbound", Width: 130},
		{Title: "Rule", Width: 130},
		{Title: "Upload", Width: 100, Align: ui.End},
		{Title: "Download", Width: 110, Align: ui.End},
		{Title: "Duration", Width: 100},
		{Title: "Action", Width: 90},
	}
	ui.Table(c, &a.connsList, columns, len(connections), func(row, col int) {
		if row < 0 || row >= len(connections) {
			return
		}
		conn := connections[row]
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
				a.askCloseConnection(conn)
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
