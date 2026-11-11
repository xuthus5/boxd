package main

import (
	"github.com/egoist/mygo/ui"
)

func card(c *ui.Context, fn func()) ui.Element {
	t := c.Theme()
	return ui.Column(c).Padding(cardPadding).Gap(compactGap).Radius(cardRadius).
		Background(t.Surface).Border(1, t.Border).Children(fn)
}

func metricCard(c *ui.Context, label, value, detail string, tone ui.Color) ui.Element {
	t := c.Theme()
	return card(c, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(8, 8).Radius(4).Background(tone)
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
		})
		ui.Text(c, value).FontSize(24).Bold().NoWrap()
		ui.Text(c, detail).FontSize(12).TextColor(t.TextMuted).NoWrap()
	}).Grow(1)
}

func pageHeading(c *ui.Context, title, subtitle string) {
	t := c.Theme()
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, title).FontSize(26).Bold().Label(title + " page title")
		ui.Text(c, subtitle).FontSize(13).TextColor(t.TextMuted)
	})
}

func sectionHeading(c *ui.Context, title, subtitle string) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
		ui.Text(c, title).FontSize(16).Bold()
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted)
		}
	})
}

func statusPill(c *ui.Context, label string, color ui.Color) ui.Element {
	return ui.Badge(c, label).Background(color).TextColor(ui.RGB(255, 255, 255))
}

func errorBanner(c *ui.Context, message string) {
	if message == "" {
		return
	}
	t := c.Theme()
	ui.Row(c).FillWidth().Padding(10, 14).Gap(10).Radius(10).
		Background(t.Danger.Alpha(0.1)).Border(1, t.Danger.Alpha(0.35)).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Error").FontSize(12).Bold().TextColor(t.Danger)
		ui.Text(c, message).FontSize(12).Grow(1).TextColor(t.Text)
	})
}

func warningBanner(c *ui.Context, message string) {
	if message == "" {
		return
	}
	t := c.Theme()
	ui.Row(c).FillWidth().Padding(10, 14).Gap(10).Radius(10).
		Background(t.Warning.Alpha(0.1)).Border(1, t.Warning.Alpha(0.35)).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Notice").FontSize(12).Bold().TextColor(t.Warning)
		ui.Text(c, message).FontSize(12).Grow(1).TextColor(t.Text)
	})
}

func successBanner(c *ui.Context, message string) {
	successBannerLabel(c, "Saved", message)
}

func successBannerLabel(c *ui.Context, label, message string) {
	if message == "" {
		return
	}
	t := c.Theme()
	ui.Row(c).FillWidth().Padding(10, 14).Gap(10).Radius(10).
		Background(t.Success.Alpha(0.1)).Border(1, t.Success.Alpha(0.35)).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).Bold().TextColor(t.Success)
		ui.Text(c, message).FontSize(12).Grow(1).TextColor(t.Text)
	})
}

func emptyState(c *ui.Context, title, detail string) {
	t := c.Theme()
	ui.Column(c).FillWidth().Padding(32).Gap(8).AlignItems(ui.Center).
		Justify(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(16).Bold().TextColor(t.Text)
		ui.Text(c, detail).FontSize(12).TextColor(t.TextMuted).MaxWidth(520)
	})
}

func keyValue(c *ui.Context, key, value string) {
	t := c.Theme()
	ui.Row(c).Gap(12).Children(func() {
		ui.Text(c, key).Width(120).FontSize(12).TextColor(t.TextMuted)
		ui.Text(c, value).FontSize(12).Grow(1).NoWrap()
	})
}
