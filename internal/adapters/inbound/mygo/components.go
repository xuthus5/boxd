package mygo

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func (c *Controller) card(ctx *ui.Context, key string, fn func()) ui.Element {
	p := c.colors()
	return ui.Column(ctx).Key(key).Background(p.Surface).Border(1, p.Border).Radius(16).
		Padding(18).Gap(12).Shadow(0, 4, 18, 0, p.Shadow).Children(fn)
}

func (c *Controller) softCard(ctx *ui.Context, key string, fn func()) ui.Element {
	p := c.colors()
	return ui.Column(ctx).Key(key).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(14).
		Padding(16).Gap(10).Children(fn)
}

func (c *Controller) sectionHeader(ctx *ui.Context, title, subtitle string, actions func()) {
	ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Column(ctx).Grow(1).Gap(3).Children(func() {
			ui.Text(ctx, title).FontSize(15).FontWeight(650).SingleLine()
			if subtitle != "" {
				ui.Text(ctx, subtitle).FontSize(12).TextColor(c.colors().TextMuted).SingleLine()
			}
		})
		if actions != nil {
			ui.Row(ctx).Gap(8).AlignItems(ui.Center).Children(actions)
		}
	})
}

func (c *Controller) iconAction(ctx *ui.Context, icon *ui.SVG, label string, danger bool) ui.Element {
	p := c.colors()
	button := ui.ButtonBase(ctx).Key("action-"+label).Label(label).Size(32, 32).Radius(9).Padding(0)
	if danger {
		button.Tooltip(label).TextColor(p.Danger).Background(p.DangerSoft)
	} else {
		button.Tooltip(label).TextColor(p.TextMuted).Background(p.SurfaceAlt)
	}
	return button.Children(func() {
		ui.Icon(ctx, icon).FontSize(16).TextColor(func() ui.Color {
			if danger {
				return p.Danger
			}
			return p.TextMuted
		}())
	})
}

func (c *Controller) labeledAction(ctx *ui.Context, icon *ui.SVG, label string, primary bool) ui.Element {
	p := c.colors()
	button := ui.ButtonBase(ctx).Key("action-"+label).Label(label).Height(34).Padding(0, 13).Radius(9).
		Gap(7).AlignItems(ui.Center)
	if primary {
		c.brandGradient(button).TextColor(p.AccentText)
	} else {
		button.Background(p.SurfaceAlt).Border(1, p.Border).TextColor(p.Text)
	}
	return button.Children(func() {
		if icon != nil {
			ui.Icon(ctx, icon).FontSize(15).TextColor(func() ui.Color {
				if primary {
					return p.AccentText
				}
				return p.TextMuted
			}())
		}
		ui.Text(ctx, label).FontSize(12.5).FontWeight(600).SingleLine()
	})
}

func (c *Controller) dangerAction(ctx *ui.Context, label string) ui.Element {
	p := c.colors()
	return ui.ButtonBase(ctx).Key("danger-"+label).Label(label).Height(34).Padding(0, 13).Radius(9).
		Background(p.DangerSoft).TextColor(p.Danger).Children(func() {
		ui.Text(ctx, label).FontSize(12.5).FontWeight(600).SingleLine().TextColor(p.Danger)
	})
}

func (c *Controller) statTile(ctx *ui.Context, key, label, value, hint string, tone ui.Color) {
	p := c.colors()
	ui.Column(ctx).Key(key).Grow(1).Basis(0).MinWidth(130).Background(p.SurfaceAlt).Border(1, p.BorderSoft).
		Radius(13).Padding(14).Gap(6).Children(func() {
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Box(ctx).Size(7, 7).Radius(7).Background(tone)
			ui.Text(ctx, label).FontSize(11.5).FontWeight(600).TextColor(p.TextMuted).SingleLine()
		})
		ui.Text(ctx, value).FontSize(21).FontWeight(720).SingleLine()
		if hint != "" {
			ui.Text(ctx, hint).FontSize(11).TextColor(p.TextFaint).SingleLine()
		}
	})
}

func (c *Controller) statusPill(ctx *ui.Context, state string) {
	p := c.colors()
	color, fill, label := p.TextMuted, p.SurfaceAlt, state
	switch state {
	case "running":
		color, fill, label = p.Success, p.SuccessSoft, "运行中"
	case "starting":
		color, fill, label = p.Warning, p.WarningSoft, "启动中"
	case "stopping":
		color, fill, label = p.Warning, p.WarningSoft, "停止中"
	case "failed":
		color, fill, label = p.Danger, p.DangerSoft, "异常"
	case "stopped":
		color, fill, label = p.TextMuted, p.SurfaceAlt, "未启动"
	}
	ui.Row(ctx).Height(28).Padding(0, 10).Radius(99).Background(fill).Gap(7).AlignItems(ui.Center).Children(func() {
		ui.Box(ctx).Size(6, 6).Radius(6).Background(color)
		ui.Text(ctx, label).FontSize(11.5).FontWeight(650).TextColor(color).SingleLine()
	})
}

func (c *Controller) smallPill(ctx *ui.Context, text string, tone ui.Color, fill ui.Color) {
	ui.Row(ctx).Height(24).Padding(0, 8).Radius(7).Background(fill).AlignItems(ui.Center).Children(func() {
		ui.Text(ctx, text).FontSize(10.5).FontWeight(650).TextColor(tone).SingleLine()
	})
}

func (c *Controller) subtleDivider(ctx *ui.Context) {
	ui.Divider(ctx).Width(1).Height(1).Background(c.colors().BorderSoft)
}

func (c *Controller) emptyState(ctx *ui.Context, title, hint string, action func()) {
	p := c.colors()
	ui.Column(ctx).FillWidth().Padding(34, 18).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Row(ctx).Size(40, 40).Radius(12).Background(p.SurfaceAlt).Border(1, p.BorderSoft).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(ctx, navIcon(c.page)).FontSize(18).TextColor(p.TextFaint)
		})
		ui.Text(ctx, title).FontSize(13).FontWeight(650).TextColor(p.Text)
		ui.Text(ctx, hint).FontSize(12).TextColor(p.TextMuted).TextAlign(ui.Center).Wrap()
		if action != nil {
			action()
		}
	})
}

func (c *Controller) tableState(key string) *ui.ListState {
	state := c.lists[key]
	if state == nil {
		state = &ui.ListState{}
		c.lists[key] = state
	}
	state.RowHeight = 42
	return state
}

func tableCell(ctx *ui.Context, text string, color ui.Color) {
	ui.Text(ctx, text).FontSize(12.5).SingleLine().TextColor(color).Tooltip(text)
}

func tableMonoCell(ctx *ui.Context, text string, color ui.Color) {
	ui.Text(ctx, text).FontSize(11.5).Font("monospace").SingleLine().TextColor(color).Tooltip(text)
}

func (c *Controller) tableHeight(ctx *ui.Context) float32 {
	_, h := ctx.Size()
	return max(300, h-300)
}

func (c *Controller) tableHeightForRows(ctx *ui.Context, rows int) float32 {
	const rowHeight = 42
	contentHeight := float32(max(rows, 1)*rowHeight + 54)
	return min(c.tableHeight(ctx), max(170, contentHeight))
}

func humanBytes(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	number := float64(value)
	unit := "B"
	for _, next := range units {
		number /= 1024
		unit = next
		if number < 1024 {
			break
		}
	}
	if number >= 100 {
		return fmt.Sprintf("%.0f %s", number, unit)
	}
	return fmt.Sprintf("%.1f %s", number, unit)
}

func humanRate(value int64) string {
	if value <= 0 {
		return "—"
	}
	return humanBytes(value) + "/s"
}

func percent(value, total int) string {
	if total <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%d%%", int(math.Round(float64(value)*100/float64(total))))
}

func shortTime(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Local().Format("01-02 15:04")
}

func fullTime(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func relativeTime(value time.Time) string {
	if value.IsZero() {
		return "从未"
	}
	delta := time.Since(value)
	if delta < 0 {
		delta = 0
	}
	switch {
	case delta < time.Minute:
		return "刚刚"
	case delta < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(delta.Hours()))
	default:
		return fmt.Sprintf("%d 天前", int(delta.Hours()/24))
	}
}

func duration(value time.Duration) string {
	if value <= 0 {
		return "—"
	}
	if value < time.Millisecond {
		return fmt.Sprintf("%.0f µs", float64(value.Microseconds()))
	}
	if value < time.Second {
		return fmt.Sprintf("%.0f ms", float64(value.Milliseconds()))
	}
	return fmt.Sprintf("%.2f s", value.Seconds())
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func logLevelColor(p palette, level string) ui.Color {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return p.TextFaint
	case "debug":
		return p.Violet
	case "info":
		return p.Sky
	case "warn", "warning":
		return p.Warning
	case "error", "fatal", "panic":
		return p.Danger
	default:
		return p.TextMuted
	}
}

func protocolLabel(protocol string) string {
	switch protocol {
	case "shadowsocks", "ss":
		return "Shadowsocks"
	case "vmess":
		return "VMess"
	case "vless":
		return "VLESS"
	case "trojan":
		return "Trojan"
	case "hysteria2", "hy2":
		return "Hysteria2"
	case "tuic":
		return "TUIC"
	case "wireguard":
		return "WireGuard"
	default:
		return protocol
	}
}

func groupKindLabel(kind string) string {
	switch kind {
	case "urltest":
		return "自动测速"
	case "selector":
		return "手动选择"
	default:
		return kind
	}
}

func routeModeLabel(mode string) string {
	switch mode {
	case "global":
		return "全局代理"
	case "direct":
		return "全局直连"
	case "rules":
		return "规则分流"
	default:
		return mode
	}
}

func routeActionLabel(action string) string {
	switch action {
	case "direct":
		return "直连"
	case "reject":
		return "拒绝"
	case "egress":
		return "指定出口"
	default:
		return action
	}
}
