package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

func (a *application) dashboardView(c *ui.Context) {
	a.pageColumn(c, func() {
		pageHeading(c, "Dashboard", "Kernel health, live traffic and the controls that matter most.")
		if !a.model.loaded {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, noDataPlaceholder).TextColor(c.Theme().TextMuted)
			})
		}
		a.dashboardMetrics(c)
		a.dashboardTraffic(c)
		a.dashboardKernel(c)
	})
}

func (a *application) dashboardMetrics(c *ui.Context) {
	t := c.Theme()
	running := a.model.hasStatus && a.model.status.Running
	statusTone := t.Danger
	if running {
		statusTone = t.Success
	}
	ui.Row(c).FillWidth().Gap(12).Children(func() {
		metricCard(c, "Kernel", kernelStatusLabel(running, a.model.busy),
			firstText(a.model.status.Uptime, "Not running"), statusTone)
		metricCard(c, "Download", formatRate(lastDownRate(a.model.traffic)),
			formatBytes(a.model.totalDown)+" total", t.Accent)
		metricCard(c, "Upload", formatRate(lastUpRate(a.model.traffic)),
			formatBytes(a.model.totalUp)+" total", t.Warning)
		metricCard(c, "Connections", fmt.Sprint(len(a.model.connections)),
			"Active right now", t.Success)
	})
}

func (a *application) dashboardTraffic(c *ui.Context) {
	t := c.Theme()
	card(c, func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			sectionHeading(c, "Traffic", "Rolling two-minute sample window")
			ui.Spacer(c)
			ui.Row(c).Gap(14).Children(func() {
				legend(c, "Download", t.Accent)
				legend(c, "Upload", t.Warning)
			})
		})
		if len(a.model.traffic) < 2 {
			emptyState(c, "Collecting traffic samples", "The native chart appears after the second refresh.")
			return
		}
		up, down := a.model.trafficSeries()
		xs := make([]float32, len(up))
		for i := range xs {
			xs[i] = float32(i) / float32(len(xs)-1)
		}
		peak := float64(1)
		for _, sample := range a.model.traffic {
			peak = max(peak, float64(sample.Up), float64(sample.Down))
		}
		y := chart.NewLinear(0, peak).Nice(4)
		ticks := y.Ticks(4)
		chart.Plot(c, chart.Axes{Left: ticks}, func(area ui.Element) {
			area.Key("traffic-chart").Label("Traffic rates").Description("Upload and download bytes per second")
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tick := range ticks {
					p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, t.Border)
				}
				downY := scaledSeries(down, y)
				upY := scaledSeries(up, y)
				p.FillPath(chart.Area(r, xs, nil, downY, chart.CurveMonotone), t.Accent.Alpha(0.12))
				p.StrokePath(chart.Line(r, xs, downY, chart.CurveMonotone), 2, t.Accent)
				p.StrokePath(chart.Line(r, xs, upY, chart.CurveMonotone), 2, t.Warning)
			})
		}).FillWidth().Height(220).Gap(8)
	})
}

func (a *application) dashboardKernel(c *ui.Context) {
	t := c.Theme()
	card(c, func() {
		sectionHeading(c, "Kernel controls", "Operations are serialized to avoid duplicate starts or stops.")
		ui.Row(c).Gap(8).Children(func() {
			running := a.model.hasStatus && a.model.status.Running
			if ui.PrimaryButton(c, "Start").Key("kernel-start").Disabled(a.model.busy || running).Clicked() {
				a.startKernelAction("Starting kernel")
			}
			if ui.Button(c, "Stop").Key("kernel-stop").Disabled(a.model.busy || !running).Clicked() {
				a.stopKernelAction()
			}
			if ui.Button(c, "Restart").Key("kernel-restart").Disabled(a.model.busy || !running).Clicked() {
				a.restartKernelAction()
			}
		})
		ui.Divider(c)
		keyValue(c, "Version", firstText(a.model.status.Version, "Unavailable"))
		keyValue(c, "Uptime", firstText(a.model.status.Uptime, "Not running"))
		keyValue(c, "Memory", memoryText(a.model.status.Memory))
		keyValue(c, "Config", firstText(a.model.status.ConfigPath, a.cfg.ConfigPath))
		if a.model.status.LastError != "" {
			ui.Text(c, "Last error: "+shortErrorText(a.model.status.LastError)).FontSize(12).
				TextColor(t.Danger).MaxWidth(860)
		}
	})
}

func legend(c *ui.Context, label string, color ui.Color) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(8, 8).Radius(4).Background(color)
		ui.Text(c, label).FontSize(11).TextColor(c.Theme().TextMuted)
	})
}

func scaledSeries(values []float32, scale chart.Linear) []float32 {
	out := make([]float32, len(values))
	for i, value := range values {
		out[i] = scale.At(float64(value))
	}
	return out
}

func lastUpRate(samples []trafficSample) float32 {
	if len(samples) == 0 {
		return 0
	}
	return samples[len(samples)-1].Up
}

func lastDownRate(samples []trafficSample) float32 {
	if len(samples) == 0 {
		return 0
	}
	return samples[len(samples)-1].Down
}

func firstText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func memoryText(value int64) string {
	if value <= 0 {
		return "Unavailable"
	}
	return formatBytes(value)
}

func shortErrorText(value string) string {
	return truncateText(value, maxErrorLength)
}
