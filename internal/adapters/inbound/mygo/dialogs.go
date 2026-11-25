package mygo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
)

func (c *Controller) dialogs(ctx *ui.Context) {
	c.editorDialog(ctx)
	if c.deleteOpen {
		switch ui.AlertDialog(ctx, &c.deleteOpen, "删除“"+c.deleteName+"”？", "此操作会立即写入控制面，并且无法撤销。", "取消", "删除") {
		case 1:
			c.confirmDelete()
		}
	}
}

func (c *Controller) editorDialog(ctx *ui.Context) {
	if !c.editorOpen {
		return
	}
	p := c.colors()
	ui.Modal(ctx, &c.editorOpen, func() {
		ui.Column(ctx).Width(editorWidth(c.editor.kind)).Gap(14).Padding(4).Children(func() {
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Column(ctx).Grow(1).Gap(3).Children(func() {
					ui.Text(ctx, editorTitle(c.editor)).FontSize(17).FontWeight(720).SingleLine()
					ui.Text(ctx, editorHint(c.editor.kind)).FontSize(11).TextColor(p.TextMuted).SingleLine()
				})
				closeButton := c.iconAction(ctx, iconClose, "关闭编辑器", false)
				if closeButton.Clicked() {
					c.editorOpen = false
				}
			})
			if c.editor.kind == editorGroup || c.editor.kind == editorChain {
				_, height := ctx.Size()
				limit := float32(520)
				if c.editor.kind == editorChain {
					limit = 420
				}
				bodyHeight := min(limit, max(float32(300), height-300))
				ui.Scroll(ctx).Key("editor-body").FillWidth().Height(bodyHeight).Gap(14).Padding(0, 2, 8, 2).Children(func() {
					c.editorFields(ctx)
				})
			} else {
				c.editorFields(ctx)
			}
			if c.editorErr != "" {
				c.editorErrorBanner(ctx, c.editorErr)
			}
			ui.Row(ctx).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				if c.labeledAction(ctx, nil, "取消", false).Clicked() {
					c.editorOpen = false
				}
				if c.labeledAction(ctx, iconCheck, "保存", true).Clicked() {
					c.submitEditor()
				}
			})
		})
	})
}

func editorWidth(kind editorKind) float32 {
	switch kind {
	case editorRouteRule:
		return 700
	case editorImportLinks:
		return 600
	case editorInbound:
		return 620
	case editorGroup:
		return 820
	case editorChain:
		return 780
	default:
		return 540
	}
}

func editorTitle(e editorState) string {
	action := "编辑"
	if e.id == "" {
		action = "添加"
	}
	switch e.kind {
	case editorInbound:
		return action + "入口"
	case editorSubscription:
		return action + "订阅"
	case editorNode:
		return action + "节点"
	case editorImportLinks:
		return "导入节点链接"
	case editorGroup:
		return action + "策略组"
	case editorChain:
		return action + "代理链路"
	case editorRouteRule:
		return action + "路由规则"
	case editorDNS:
		return "编辑 " + dnsSlotLabel(e.dnsSlot)
	case editorRuleSet:
		if e.builtin {
			return "编辑内置规则集"
		}
		return action + "规则集"
	default:
		return "编辑"
	}
}

func editorHint(kind editorKind) string {
	switch kind {
	case editorInbound:
		return "入口修改会立即写入控制面；内核运行时会自动应用新配置"
	case editorSubscription:
		return "订阅地址只保存在本地控制面数据库中"
	case editorNode:
		return "协议参数使用 JSON 对象；具体字段由协议类型决定"
	case editorImportLinks:
		return "支持每行一个 VMess、VLESS、Trojan、Shadowsocks 等分享链接"
	case editorGroup:
		return "搜索并添加节点，拖拽或使用上下按钮调整成员顺序"
	case editorChain:
		return "按本地到目标的顺序选择跳点；支持拖拽与上下按钮排序"
	case editorRouteRule:
		return "规则按列表顺序求值，越靠前优先级越高"
	case editorDNS:
		return "地址、协议与出口组合会直接编译为内核 DNS 配置"
	case editorRuleSet:
		return "内置规则集只能修改启用状态，不能删除或替换来源"
	default:
		return ""
	}
}

func (c *Controller) editorFields(ctx *ui.Context) {
	e := &c.editor
	switch e.kind {
	case editorInbound:
		ui.Field(ctx, "名称", func() { ui.TextInput(ctx, &e.name).AutoFocus().Label("名称") })
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			c.smallPill(ctx, inboundKindLabel(e.inboundKind), c.colors().Sky, c.colors().SurfaceAlt)
			ui.Column(ctx).Grow(1).Children(func() {
				ui.Text(ctx, inboundKindHint(e.inboundKind)).FontSize(11).TextColor(c.colors().TextMuted).SingleLine()
			})
			ui.Text(ctx, "启用入口").FontSize(11.5).TextColor(c.colors().TextMuted)
			ui.Switch(ctx, &e.enabled).Label("启用入口")
		})
		if e.inboundKind == control.InboundKindMixed {
			ui.Row(ctx).FillWidth().Gap(12).Children(func() {
				ui.Column(ctx).Grow(1).Basis(0).Children(func() {
					ui.Field(ctx, "监听地址", func() {
						ui.TextInput(ctx, &e.listenHost).Placeholder("::").Label("监听地址")
					})
				})
				ui.Column(ctx).Width(140).Children(func() {
					ui.Field(ctx, "监听端口", func() {
						ui.TextInput(ctx, &e.listenPort).Placeholder("1090").Label("监听端口")
					})
				})
			})
			ui.Text(ctx, ":: 表示同时监听 IPv4 和 IPv6；127.0.0.1 仅允许本机访问，0.0.0.0 监听全部 IPv4 地址。").
				FontSize(10.5).TextColor(c.colors().TextFaint).Wrap().LineHeight(1.5)
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Column(ctx).Grow(1).Gap(2).Children(func() {
					ui.Text(ctx, "UDP 转发").FontSize(12.5).FontWeight(600).SingleLine()
					ui.Text(ctx, "允许 SOCKS UDP Associate 通过入口转发").FontSize(10.5).TextColor(c.colors().TextFaint).SingleLine()
				})
				ui.Switch(ctx, &e.udp).Label("UDP 转发")
			})
		} else {
			ui.Row(ctx).FillWidth().Gap(12).Children(func() {
				ui.Column(ctx).Grow(1).Basis(0).Children(func() {
					ui.Field(ctx, "接口名称", func() {
						ui.TextInput(ctx, &e.tunInterface).Placeholder("留空使用系统默认").Label("接口名称")
					})
				})
				ui.Column(ctx).Width(130).Children(func() {
					ui.Field(ctx, "MTU", func() { ui.TextInput(ctx, &e.tunMTU).Label("MTU") })
				})
				ui.Column(ctx).Width(150).Children(func() {
					ui.Field(ctx, "协议栈", func() {
						labels := []string{"混合", "系统", "GVisor"}
						values := []string{"mixed", "system", "gvisor"}
						current := optionLabel(labels, values, e.tunStack)
						selector := ui.Select(ctx, &current, labels).Label("协议栈")
						if selector.Changed() {
							e.tunStack = optionValueByLabel(labels, values, current)
						}
					})
				})
			})
			ui.Field(ctx, "接口地址（每行一个 CIDR）", func() {
				ui.TextArea(ctx, &e.tunAddresses).Height(76).Font("monospace").Label("接口地址")
			})
			ui.Row(ctx).FillWidth().Gap(28).Children(func() {
				ui.Row(ctx).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Text(ctx, "自动路由").FontSize(11.5).TextColor(c.colors().TextMuted)
					ui.Switch(ctx, &e.tunAutoRoute).Label("自动路由")
				})
				ui.Row(ctx).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Text(ctx, "严格路由").FontSize(11.5).TextColor(c.colors().TextMuted)
					ui.Switch(ctx, &e.tunStrictRoute).Label("严格路由")
				})
			})
			ui.Text(ctx, "Windows 上启用 TUN 需要管理员权限；如果当前账户没有网络适配器权限，保存会自动回滚并给出错误。").
				FontSize(10.5).TextColor(c.colors().TextFaint).Wrap().LineHeight(1.5)
		}
	case editorSubscription:
		ui.Field(ctx, "名称", func() { ui.TextInput(ctx, &e.name).AutoFocus().Label("名称") })
		ui.Field(ctx, "订阅地址", func() {
			ui.TextInput(ctx, &e.url).Placeholder("https://example.com/subscription").Label("订阅地址")
		})
		ui.Field(ctx, "同步间隔（秒）", func() { ui.TextInput(ctx, &e.interval).Label("同步间隔（秒）") })
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(ctx, "启用此订阅").FontSize(11.5).TextColor(c.colors().TextMuted).Grow(1)
			ui.Switch(ctx, &e.enabled).Label("启用此订阅")
		})
	case editorNode:
		ui.Field(ctx, "名称", func() { ui.TextInput(ctx, &e.name).AutoFocus().Label("名称") })
		ui.Row(ctx).FillWidth().Gap(12).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "协议", func() { protocolSelector(ctx, &e.protocol) })
			})
			ui.Column(ctx).Width(110).Children(func() {
				ui.Field(ctx, "端口", func() { ui.TextInput(ctx, &e.port).Label("端口") })
			})
		})
		ui.Field(ctx, "服务器地址", func() {
			ui.TextInput(ctx, &e.server).Placeholder("example.com 或 203.0.113.1").Label("服务器地址")
		})
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(ctx, "允许 UDP").FontSize(11.5).TextColor(c.colors().TextMuted).Grow(1)
			ui.Switch(ctx, &e.udp).Label("允许 UDP")
		})
		ui.Field(ctx, "协议参数（JSON）", func() {
			ui.TextArea(ctx, &e.options).Height(190).Font("monospace").Label("协议参数（JSON）")
		})
	case editorImportLinks:
		ui.Row(ctx).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Gap(6).Children(func() {
				ui.Text(ctx, "导入到策略组").FontSize(11).FontWeight(600).TextColor(c.colors().TextMuted)
				options, values := importGroupOptions(c.snapshot, e.groupID)
				selected := optionLabel(options, values, e.groupID)
				selector := ui.Select(ctx, &selected, options).Label("导入到策略组")
				if selector.Changed() {
					e.groupID = optionValueByLabel(options, values, selected)
				}
			})
			ui.Text(ctx, "留空时仅保存为未分组节点").FontSize(10.5).TextColor(c.colors().TextFaint).Padding(0, 0, 9, 0)
		})
		ui.Field(ctx, "节点分享链接", func() {
			ui.TextArea(ctx, &e.links).Height(250).Font("monospace").Placeholder("vless://...\nss://...").AutoFocus().Label("节点分享链接")
		})
	case editorGroup:
		c.groupEditorFields(ctx, e)
	case editorChain:
		c.chainEditorFields(ctx, e)
	case editorRouteRule:
		ui.Row(ctx).FillWidth().Gap(12).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "规则名称", func() { ui.TextInput(ctx, &e.name).AutoFocus().Label("规则名称") })
			})
			ui.Column(ctx).Width(150).Children(func() {
				ui.Field(ctx, "动作", func() {
					labels := []string{"指定出口", "直连", "拒绝"}
					values := []string{string(control.RouteActionEgress), string(control.RouteActionDirect), string(control.RouteActionReject)}
					current := routeActionLabel(e.action)
					selector := ui.Select(ctx, &current, labels).Label("动作")
					if selector.Changed() {
						e.action = optionValueByLabel(labels, values, current)
					}
				})
			})
		})
		ui.Row(ctx).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "目标出口", func() {
					options := egressOptions(c.snapshot)
					current := egressOptionLabel(control.EgressRef{Kind: control.EgressKind(e.egressKind), ID: e.egressID}, c.snapshot)
					selector := ui.Select(ctx, &current, options).Label("目标出口")
					if selector.Changed() {
						kind, id := parseEgressOption(optionValue(current))
						e.egressKind, e.egressID = kind, id
					}
				})
			})
			ui.Column(ctx).Gap(7).Padding(0, 0, 3, 0).Children(func() {
				ui.Text(ctx, "停用规则").FontSize(10.5).TextColor(c.colors().TextMuted)
				disabled := e.rule.Disabled
				if ui.Switch(ctx, &disabled).Label("停用规则").Changed() {
					e.rule.Disabled = disabled
				}
			})
		})
		ui.Row(ctx).FillWidth().Gap(12).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "域名（每行一个）", func() { ui.TextArea(ctx, &e.matchDomains).Height(86).Font("monospace").Label("域名") })
			})
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "域名后缀（每行一个）", func() { ui.TextArea(ctx, &e.matchSuffixes).Height(86).Font("monospace").Label("域名后缀") })
			})
		})
		ui.Row(ctx).FillWidth().Gap(12).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "规则集（每行一个）", func() { ui.TextArea(ctx, &e.matchRuleSets).Height(72).Font("monospace").Label("规则集") })
			})
			ui.Column(ctx).Width(160).Children(func() {
				ui.Field(ctx, "端口（逗号分隔）", func() { ui.TextInput(ctx, &e.matchPorts).Label("端口") })
			})
		})
	case editorDNS:
		ui.Field(ctx, "服务器地址", func() { ui.TextInput(ctx, &e.dnsServer.Address).AutoFocus().Label("服务器地址") })
		ui.Row(ctx).FillWidth().Gap(12).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "协议", func() {
					labels := []string{"UDP", "DoH", "DoT"}
					values := []string{string(control.DNSProtocolUDP), string(control.DNSProtocolDoH), string(control.DNSProtocolDoT)}
					current := strings.ToUpper(string(e.dnsServer.Protocol))
					selector := ui.Select(ctx, &current, labels).Label("协议")
					if selector.Changed() {
						e.dnsServer.Protocol = control.DNSProtocol(optionValueByLabel(labels, values, current))
					}
				})
			})
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "出口", func() {
					labels := []string{"直连", "跟随路由", "指定出口"}
					values := []string{string(control.DNSDetourDirect), string(control.DNSDetourRouteDefault), string(control.DNSDetourEgress)}
					current := dnsDetourLabel(e.dnsServer.Detour)
					selector := ui.Select(ctx, &current, labels).Label("出口")
					if selector.Changed() {
						e.dnsServer.Detour = control.DNSDetour(optionValueByLabel(labels, values, current))
					}
				})
			})
			ui.Column(ctx).Width(120).Children(func() {
				ui.Field(ctx, "超时（秒）", func() {
					timeout := strconv.Itoa(e.dnsServer.TimeoutSeconds)
					ui.TextInput(ctx, &timeout).Label("超时（秒）")
					if value, err := strconv.Atoi(timeout); err == nil {
						e.dnsServer.TimeoutSeconds = value
					}
				})
			})
		})
		ui.Row(ctx).FillWidth().Gap(24).Children(func() {
			ui.Row(ctx).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Text(ctx, "启用槽位").FontSize(11.5).TextColor(c.colors().TextMuted)
				ui.Switch(ctx, &e.dnsServer.Enabled).Label("启用槽位")
			})
			if e.dnsSlot == "default" {
				ui.Row(ctx).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Text(ctx, "沿用国外 DNS").FontSize(11.5).TextColor(c.colors().TextMuted)
					ui.Switch(ctx, &e.dnsServer.FollowForeign).Label("沿用国外 DNS")
				})
			}
		})
	case editorRuleSet:
		ui.Field(ctx, "名称", func() {
			input := ui.TextInput(ctx, &e.name).Label("名称")
			if e.builtin {
				input.Disabled(true)
			}
		})
		ui.Field(ctx, "来源 URL", func() {
			input := ui.TextInput(ctx, &e.url).Label("来源 URL")
			if e.builtin {
				input.Disabled(true)
			}
		})
		ui.Row(ctx).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).Children(func() {
				ui.Field(ctx, "格式", func() {
					labels := []string{"二进制", "源码"}
					values := []string{string(control.RuleSetFormatBinary), string(control.RuleSetFormatSource)}
					current := string(e.format)
					if current == "source" {
						current = "源码"
					} else {
						current = "二进制"
					}
					selector := ui.Select(ctx, &current, labels).Label("格式")
					selector.Disabled(e.builtin)
					if selector.Changed() {
						e.format = optionValueByLabel(labels, values, current)
					}
				})
			})
			ui.Column(ctx).Gap(7).Padding(0, 0, 3, 0).Children(func() {
				ui.Text(ctx, "启用规则集").FontSize(10.5).TextColor(c.colors().TextMuted)
				ui.Switch(ctx, &e.enabled).Label("启用规则集")
			})
		})
	}
}

func protocolSelector(ctx *ui.Context, value *string) {
	labels := []string{"Shadowsocks", "VMess", "VLESS", "Trojan", "Hysteria2", "TUIC", "WireGuard"}
	values := []string{"shadowsocks", "vmess", "vless", "trojan", "hysteria2", "tuic", "wireguard"}
	current := protocolLabel(*value)
	selector := ui.Select(ctx, &current, labels).Label("协议")
	if selector.Changed() {
		*value = optionValueByLabel(labels, values, current)
	}
}

func importGroupOptions(s snapshot, selected string) ([]string, []string) {
	options := []string{"不加入策略组"}
	values := []string{""}
	for _, group := range s.Catalog.Groups {
		options = append(options, group.Name)
		values = append(values, group.ID)
	}
	return options, values
}

func optionLabel(labels, values []string, current string) string {
	for i, value := range values {
		if value == current {
			return labels[i]
		}
	}
	if len(labels) > 0 {
		return labels[0]
	}
	return ""
}

func optionValueByLabel(labels, values []string, label string) string {
	for i, candidate := range labels {
		if candidate == label && i < len(values) {
			return values[i]
		}
	}
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

var _ = fmt.Sprintf
