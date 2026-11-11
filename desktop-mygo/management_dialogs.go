package main

import "github.com/egoist/mygo/ui"

func (a *application) managementDialogs(c *ui.Context) {
	a.nodeFormDialog(c)
	a.subscriptionFormDialog(c)
	a.selectorDialog(c)
	a.deleteDialog(c)
}

func dialogPanel(c *ui.Context, title, subtitle string, body func()) {
	t := c.Theme()
	ui.Column(c).Width(620).Padding(20).Gap(14).Radius(14).Background(t.Surface).
		Border(1, t.Border).Children(func() {
		ui.Text(c, title).FontSize(20).Bold().Label(title)
		ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted).MaxWidth(560)
		body()
	})
}
