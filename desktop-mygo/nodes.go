package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *application) nodesView(c *ui.Context) {
	a.pageColumn(c, func() {
		pageHeading(c, "Nodes", "Manage imported outbounds, subscription nodes and latency probes.")
		ui.Row(c).Gap(8).Children(func() {
			if ui.PrimaryButton(c, "Import node").Key("node-import").Disabled(a.model.busy).Clicked() {
				a.openAddNode()
			}
			if ui.Button(c, "Test all").Key("node-test-all").Disabled(a.model.busy || len(a.model.nodes) == 0).Clicked() {
				a.testAllNodes()
			}
		})
		ui.Row(c).Gap(12).Children(func() {
			metricCard(c, "Nodes", fmt.Sprint(len(a.model.nodes)), "Imported and subscription outbounds", c.Theme().Accent)
			metricCard(c, "Imported", fmt.Sprint(countImportedNodes(a.model.nodes)), "Managed in the shared database", c.Theme().Warning)
			metricCard(c, "Tested", fmt.Sprint(countTested(a.model.nodes)), "Nodes with at least one result", c.Theme().Success)
			metricCard(c, "Failing", fmt.Sprint(countFailed(a.model.nodes)), "Latest result failed", c.Theme().Danger)
		})
		if len(a.model.nodes) == 0 {
			card(c, func() {
				emptyState(c, "No nodes imported yet", "Import a proxy link or create a subscription to populate this list.")
				if ui.PrimaryButton(c, "Import node").Key("node-empty-import").Clicked() {
					a.openAddNode()
				}
			})
			return
		}
		a.nodesTable(c)
	})
}

func (a *application) nodesTable(c *ui.Context) {
	columns := []ui.TableColumn{
		{Title: "Tag", MinWidth: 160},
		{Title: "Source", Width: 150},
		{Title: "Type", Width: 110},
		{Title: "Server", MinWidth: 150},
		{Title: "Port", Width: 70, Align: ui.End},
		{Title: "Latest test", Width: 110},
		{Title: "Actions", Width: 220},
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
			ui.Text(c, nodeSourceLabel(node)).NoWrap().TextColor(c.Theme().TextMuted)
		case 2:
			ui.Text(c, firstText(node.Type, "unknown")).NoWrap()
		case 3:
			ui.Text(c, firstText(node.Server, "—")).NoWrap()
		case 4:
			ui.Textf(c, "%d", node.Port).NoWrap()
		case 5:
			ui.Text(c, node.Test).NoWrap().TextColor(nodeTestColor(c, node.Test))
		case 6:
			a.nodeRowActions(c, node)
		}
	}).FillWidth().Height(500).Label("Node list")
}

func (a *application) nodeRowActions(c *ui.Context, node nodeView) {
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, "Test").Key("test-node-" + node.Tag).Disabled(a.model.busy).Clicked() {
			a.testNode(node)
		}
		imported := node.Source == "import"
		if ui.Button(c, "Edit").Key("edit-node-" + node.Tag).Disabled(a.model.busy || !imported).Clicked() {
			a.openEditNode(node)
		}
		if ui.Button(c, "Delete").Key("delete-node-" + node.Tag).Disabled(a.model.busy || !imported).Clicked() {
			a.askDeleteNode(node)
		}
	})
}

func nodeSourceLabel(node nodeView) string {
	if node.Source == "subscription" {
		return firstText(node.SourceName, "Subscription")
	}
	return "Imported"
}

func nodeTestColor(c *ui.Context, test string) ui.Color {
	if test == "Failed" {
		return c.Theme().Danger
	}
	if strings.HasSuffix(test, "ms") {
		return c.Theme().Success
	}
	return c.Theme().TextMuted
}

func countImportedNodes(nodes []nodeView) int {
	count := 0
	for _, node := range nodes {
		if node.Source == "import" {
			count++
		}
	}
	return count
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
