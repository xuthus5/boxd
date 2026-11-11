package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *application) nodesView(c *ui.Context) {
	a.pageColumn(c, func() {
		pageHeading(c, "Nodes", "Read-only view of imported outbounds and their latest latency result.")
		ui.Row(c).Gap(12).Children(func() {
			metricCard(c, "Imported nodes", fmt.Sprint(len(a.model.nodes)), "Stored in the shared bbolt database", c.Theme().Accent)
			metricCard(c, "Tested", fmt.Sprint(countTested(a.model.nodes)), "Nodes with at least one result", c.Theme().Success)
			metricCard(c, "Failing", fmt.Sprint(countFailed(a.model.nodes)), "Latest result failed", c.Theme().Danger)
		})
		if len(a.model.nodes) == 0 {
			card(c, func() {
				emptyState(c, "No nodes imported yet", "Import a proxy link or refresh a subscription in the existing Wails app; this prototype reads the same data.")
			})
			return
		}
		a.nodesTable(c)
		ui.Text(c, "Node editing is intentionally out of scope for this prototype; navigation and rendering use the shared data model.").
			FontSize(11).TextColor(c.Theme().TextMuted)
	})
}

func (a *application) nodesTable(c *ui.Context) {
	columns := []ui.TableColumn{
		{Title: "Tag", Width: 220},
		{Title: "Type", Width: 110},
		{Title: "Server", MinWidth: 150},
		{Title: "Port", Width: 80, Align: ui.End},
		{Title: "Latest test", Width: 130},
	}
	ui.Table(c, &a.nodesList, columns, len(a.model.nodes), func(row, col int) {
		if row < 0 || row >= len(a.model.nodes) {
			return
		}
		node := a.model.nodes[row]
		switch col {
		case 0:
			ui.Text(c, node.Tag).NoWrap().Bold()
		case 1:
			ui.Text(c, firstText(node.Type, "unknown")).NoWrap()
		case 2:
			ui.Text(c, firstText(node.Server, "—")).NoWrap()
		case 3:
			ui.Textf(c, "%d", node.Port).NoWrap()
		case 4:
			color := c.Theme().TextMuted
			if node.Test == "Failed" {
				color = c.Theme().Danger
			} else if strings.HasSuffix(node.Test, "ms") {
				color = c.Theme().Success
			}
			ui.Text(c, node.Test).NoWrap().TextColor(color)
		}
	}).FillWidth().Height(500).Label("Node list")
}

func countTested(nodes []nodeView) int {
	count := 0
	for _, node := range nodes {
		if node.Test != "Not tested" {
			count++
		}
	}
	return count
}

func countFailed(nodes []nodeView) int {
	count := 0
	for _, node := range nodes {
		if node.Test == "Failed" {
			count++
		}
	}
	return count
}
