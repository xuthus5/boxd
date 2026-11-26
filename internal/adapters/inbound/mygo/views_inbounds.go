package mygo

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
)

func (c *Controller) inboundsSettingsCard(ctx *ui.Context) {
	p := c.colors()
	c.card(ctx, "settings-inbounds", func() {
		c.sectionHeader(ctx, "入口管理", "本地监听、端口与 TUN 接入", nil)
		enabled := 0
		mixed := control.InboundProfile{}
		tun := control.InboundProfile{}
		for _, inbound := range c.snapshot.Inbounds {
			if inbound.Enabled {
				enabled++
			}
			switch inbound.Kind {
			case control.InboundKindMixed:
				mixed = inbound
			case control.InboundKindTun:
				tun = inbound
			}
		}
		ui.Row(ctx).FillWidth().Gap(10).AlignItems(ui.Stretch).Children(func() {
			c.statTile(ctx, "inbound-enabled", "已启用入口", strconv.Itoa(enabled), "共 "+strconv.Itoa(len(c.snapshot.Inbounds))+" 个入口", p.Success)
			c.statTile(ctx, "inbound-mixed", "Mixed 监听", listenAddressLabel(mixed.Listen), inboundStateHint(mixed), p.Indigo)
			c.statTile(ctx, "inbound-tun", "TUN 接入", boolLabel(tun.Enabled, "已启用", "未启用"), tunInterfaceSummary(tun), p.Sky)
		})

		inbounds := append([]control.InboundProfile(nil), c.snapshot.Inbounds...)
		sort.SliceStable(inbounds, func(i, j int) bool {
			if inbounds[i].Kind == inbounds[j].Kind {
				return strings.ToLower(inbounds[i].Name) < strings.ToLower(inbounds[j].Name)
			}
			return inbounds[i].Kind == control.InboundKindMixed
		})
		cols := []ui.TableColumn{
			{Title: "名称", Width: 170},
			{Title: "类型", Width: 90},
			{Title: "监听 / 接口", MinWidth: 250},
			{Title: "UDP", Width: 70},
			{Title: "状态", Width: 82},
			{Title: "操作", Width: 58, Fixed: true},
		}
		state := c.tableState("inbounds")
		ui.Table(ctx, state, cols, len(inbounds), func(row, col int) {
			inbound := inbounds[row]
			switch col {
			case 0:
				ui.Text(ctx, inbound.Name).FontSize(12.5).FontWeight(600).SingleLine().Tooltip(inbound.Name)
			case 1:
				c.smallPill(ctx, inboundKindLabel(inbound.Kind), p.Sky, p.SurfaceAlt)
			case 2:
				label := listenAddressLabel(inbound.Listen)
				if inbound.Kind == control.InboundKindTun {
					label = tunInterfaceSummary(inbound)
				}
				ui.Text(ctx, label).FontSize(11.5).Font("monospace").TextColor(p.TextMuted).SingleLine().Tooltip(label)
			case 3:
				label := "—"
				if inbound.Kind == control.InboundKindMixed {
					label = boolLabel(inbound.UDP, "开启", "关闭")
				}
				ui.Text(ctx, label).FontSize(11.5).TextColor(p.TextMuted).TextAlign(ui.End)
			case 4:
				tone, fill := p.TextMuted, p.SurfaceAlt
				if inbound.Enabled {
					tone, fill = p.Success, p.SuccessSoft
				}
				c.smallPill(ctx, boolLabel(inbound.Enabled, "已启用", "已停用"), tone, fill)
			case 5:
				if c.iconAction(ctx, iconEdit, "编辑入口 "+inbound.Name, false).Clicked() {
					c.openEditor(editorInbound, inbound.ID)
				}
			}
		}).Grow(1).Height(c.tableHeightForRows(ctx, len(inbounds)))

		ui.Row(ctx).FillWidth().Padding(10, 12).Radius(10).Background(p.SurfaceAlt).
			AlignItems(ui.Start).Gap(8).Children(func() {
			ui.Icon(ctx, iconAlert).FontSize(14).TextColor(p.TextFaint)
			ui.Text(ctx, "修改端口后，内核会重新编译并应用配置；若内核正在运行，现有连接会被重新建立。Windows 上启用 TUN 需要管理员权限。").
				FontSize(11).TextColor(p.TextMuted).Grow(1).Wrap().LineHeight(1.5)
		})
	})
}

func inboundKindLabel(kind control.InboundKind) string {
	switch kind {
	case control.InboundKindMixed:
		return "Mixed"
	case control.InboundKindTun:
		return "TUN"
	default:
		return string(kind)
	}
}

func inboundKindHint(kind control.InboundKind) string {
	switch kind {
	case control.InboundKindMixed:
		return "HTTP / SOCKS 混合代理入口"
	case control.InboundKindTun:
		return "系统级透明网络入口"
	default:
		return "本地网络入口"
	}
}

func listenAddressLabel(address control.ListenAddress) string {
	if address.Host == "" && address.Port == 0 {
		return "—"
	}
	return net.JoinHostPort(address.Host, strconv.Itoa(int(address.Port)))
}

func inboundStateHint(inbound control.InboundProfile) string {
	if inbound.ID == "" {
		return "未配置"
	}
	if !inbound.Enabled {
		return "入口已停用"
	}
	if inbound.Kind == control.InboundKindTun {
		return "TUN 已启用"
	}
	return "监听 TCP / UDP"
}

func tunInterfaceSummary(inbound control.InboundProfile) string {
	if inbound.ID == "" {
		return "未配置"
	}
	name := strings.TrimSpace(inbound.Tun.InterfaceName)
	if name == "" {
		name = "系统默认接口"
	}
	if len(inbound.Tun.Addresses) == 0 {
		return name
	}
	return fmt.Sprintf("%s · %s", name, strings.Join(inbound.Tun.Addresses, ", "))
}
