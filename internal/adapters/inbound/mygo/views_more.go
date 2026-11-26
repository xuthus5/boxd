package mygo

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

func (c *Controller) egressPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "egress-groups", func() {
			c.sectionHeader(ctx, "策略组", "按名称、类型和当前出口快速检查运行状态", nil)
			if len(c.snapshot.Catalog.Groups) == 0 {
				c.emptyState(ctx, "尚未创建策略组", "策略组可以把多个节点组合为手动选择或自动测速出口。", nil)
				return
			}
			groups := append([]control.Group(nil), c.snapshot.Catalog.Groups...)
			sort.SliceStable(groups, func(i, j int) bool { return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name) })
			cols := []ui.TableColumn{
				{Title: "名称", Width: 180},
				{Title: "类型", Width: 100},
				{Title: "当前出口", MinWidth: 180},
				{Title: "成员", Width: 64, Align: ui.End},
				{Title: "测速间隔", Width: 92, Align: ui.End},
				{Title: "操作", Width: 106, Fixed: true},
			}
			state := c.tableState("egress-groups")
			ui.Table(ctx, state, cols, len(groups), func(row, col int) {
				group := groups[row]
				switch col {
				case 0:
					ui.Text(ctx, group.Name).FontSize(12.5).FontWeight(600).SingleLine().Tooltip(group.Name)
				case 1:
					c.smallPill(ctx, groupKindLabel(string(group.Kind)), p.Violet, p.SurfaceAlt)
				case 2:
					names := make([]string, 0, len(group.Members))
					for _, id := range group.Members {
						if node, ok := c.snapshot.NodesByID[id]; ok {
							names = append(names, node.Name)
						} else {
							names = append(names, id)
						}
					}
					if group.Kind == control.GroupKindSelector && len(names) > 0 {
						selected := nodeName(c.snapshot, group.Selected)
						selector := ui.Select(ctx, &selected, names).Label("当前出口")
						if selector.Changed() {
							for i, name := range names {
								if name == selected {
									c.selectGroupMember(group.ID, group.Members[i])
									break
								}
							}
						}
					} else {
						ui.Text(ctx, fmt.Sprintf("自动选择 · %s", nodeName(c.snapshot, group.Selected))).FontSize(11.5).
							TextColor(p.TextMuted).SingleLine()
					}
				case 3:
					ui.Text(ctx, strconv.Itoa(len(group.Members))).FontSize(12).TextAlign(ui.End)
				case 4:
					ui.Text(ctx, humanInterval(group.IntervalSeconds)).FontSize(11.5).TextColor(p.TextMuted).TextAlign(ui.End)
				case 5:
					ui.Row(ctx).Gap(4).Children(func() {
						if c.iconAction(ctx, iconEdit, "编辑策略组 "+group.Name, false).Clicked() {
							c.openEditor(editorGroup, group.ID)
						}
						if c.iconAction(ctx, iconTrash, "删除策略组 "+group.Name, true).Clicked() {
							c.askDelete("group", group.ID, group.Name)
						}
					})
				}
			}).Grow(1).Height(c.tableHeightForRows(ctx, len(groups)))
		})
		c.card(ctx, "egress-chains", func() {
			c.sectionHeader(ctx, "代理链路", "从上到下依次经过各个跳点；过长内容支持查看提示", nil)
			if len(c.snapshot.Catalog.Chains) == 0 {
				c.emptyState(ctx, "还没有代理链路", "链路用于串联多个出口，例如本地节点 → 中转节点 → 目标网络。", nil)
				return
			}
			chains := append([]control.ProxyChain(nil), c.snapshot.Catalog.Chains...)
			sort.SliceStable(chains, func(i, j int) bool { return strings.ToLower(chains[i].Name) < strings.ToLower(chains[j].Name) })
			cols := []ui.TableColumn{
				{Title: "名称", Width: 190},
				{Title: "链路顺序", MinWidth: 300},
				{Title: "跳点", Width: 64, Align: ui.End},
				{Title: "操作", Width: 106, Fixed: true},
			}
			state := c.tableState("egress-chains")
			ui.Table(ctx, state, cols, len(chains), func(row, col int) {
				chain := chains[row]
				switch col {
				case 0:
					ui.Text(ctx, chain.Name).FontSize(12.5).FontWeight(600).SingleLine().Tooltip(chain.Name)
				case 1:
					parts := make([]string, 0, len(chain.Hops))
					for _, hop := range chain.Hops {
						parts = append(parts, labelForEgress(control.EgressRef{Kind: hop.Kind, ID: hop.ID}, c.snapshot))
					}
					text := strings.Join(parts, "  →  ")
					ui.Text(ctx, text).FontSize(11.5).TextColor(p.TextMuted).SingleLine().Tooltip(text)
				case 2:
					ui.Text(ctx, strconv.Itoa(len(chain.Hops))).FontSize(12).TextAlign(ui.End)
				case 3:
					ui.Row(ctx).Gap(4).Children(func() {
						if c.iconAction(ctx, iconEdit, "编辑链路 "+chain.Name, false).Clicked() {
							c.openEditor(editorChain, chain.ID)
						}
						if c.iconAction(ctx, iconTrash, "删除链路 "+chain.Name, true).Clicked() {
							c.askDelete("chain", chain.ID, chain.Name)
						}
					})
				}
			}).Grow(1).Height(c.tableHeightForRows(ctx, len(chains)))
		})
	})
}

func (c *Controller) routesPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "routes-profile", func() {
			c.sectionHeader(ctx, "运行模式与默认出口", "修改后立即写入控制面并重新编译内核配置", nil)
			ui.Row(ctx).FillWidth().Gap(16).AlignItems(ui.End).Children(func() {
				ui.Column(ctx).Width(190).Gap(6).Children(func() {
					ui.Text(ctx, "运行模式").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
					labels := []string{"规则分流", "全局代理", "全局直连"}
					values := []control.RouteMode{control.RouteModeRules, control.RouteModeGlobal, control.RouteModeDirect}
					current := routeModeLabel(string(c.snapshot.Routes.Mode))
					selector := ui.Select(ctx, &current, labels).Label("运行模式")
					if selector.Changed() {
						for i, label := range labels {
							if label == current {
								c.setRouteMode(values[i])
								break
							}
						}
					}
				})
				ui.Column(ctx).Grow(1).MinWidth(220).Gap(6).Children(func() {
					ui.Text(ctx, "默认出口").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
					options := egressOptions(c.snapshot)
					current := egressOptionLabel(c.snapshot.Routes.DefaultEgress, c.snapshot)
					selector := ui.Select(ctx, &current, options).Label("默认出口")
					if selector.Changed() {
						c.setDefaultEgress(egressRefFromLabel(current, c.snapshot))
					}
				})
				ui.Row(ctx).Gap(18).Padding(0, 0, 3, 0).Children(func() {
					c.presetSwitch(ctx, "绕过局域网", c.snapshot.Routes.Presets.BypassLAN, func(value bool) {
						presets := c.snapshot.Routes.Presets
						presets.BypassLAN = value
						c.setPresets(presets)
					})
					c.presetSwitch(ctx, "国内直连", c.snapshot.Routes.Presets.ChinaDirect, func(value bool) {
						presets := c.snapshot.Routes.Presets
						presets.ChinaDirect = value
						c.setPresets(presets)
					})
					c.presetSwitch(ctx, "广告拦截", c.snapshot.Routes.Presets.AdBlock, func(value bool) {
						presets := c.snapshot.Routes.Presets
						presets.AdBlock = value
						c.setPresets(presets)
					})
				})
			})
		})
		c.card(ctx, "routes-table", func() {
			c.sectionHeader(ctx, "自定义规则", "规则按照列出的顺序自上而下匹配", nil)
			if len(c.snapshot.Routes.Rules) == 0 {
				c.emptyState(ctx, "还没有自定义规则", "预设策略会继续生效；添加规则可实现更精细的分流。", nil)
				return
			}
			rules := c.snapshot.Routes.Rules
			cols := []ui.TableColumn{
				{Title: "顺序", Width: 72, Fixed: true},
				{Title: "名称", Width: 180},
				{Title: "匹配条件", MinWidth: 280},
				{Title: "动作", Width: 92},
				{Title: "目标出口", Width: 160},
				{Title: "状态", Width: 72},
				{Title: "操作", Width: 76, Fixed: true},
			}
			state := c.tableState("routes")
			ui.Table(ctx, state, cols, len(rules), func(row, col int) {
				rule := rules[row]
				switch col {
				case 0:
					ui.Row(ctx).Gap(2).Children(func() {
						up := ui.ButtonBase(ctx).Label("上移规则").Size(24, 24).Radius(6).Padding(0).Background(p.SurfaceAlt)
						up.Children(func() { ui.Icon(ctx, iconChevronUp).FontSize(12).TextColor(p.TextMuted) })
						if up.Clicked() {
							c.moveRule(rule.ID, -1)
						}
						down := ui.ButtonBase(ctx).Label("下移规则").Size(24, 24).Radius(6).Padding(0).Background(p.SurfaceAlt)
						down.Children(func() { ui.Icon(ctx, iconChevronDown).FontSize(12).TextColor(p.TextMuted) })
						if down.Clicked() {
							c.moveRule(rule.ID, 1)
						}
					})
				case 1:
					ui.Text(ctx, rule.Name).FontSize(12.5).FontWeight(600).SingleLine().Tooltip(rule.Name)
				case 2:
					text := routeMatchSummary(rule.Match)
					ui.Text(ctx, text).FontSize(11.5).TextColor(p.TextMuted).SingleLine().Tooltip(text)
				case 3:
					c.smallPill(ctx, routeActionLabel(string(rule.Action)), p.Sky, p.SurfaceAlt)
				case 4:
					ui.Text(ctx, labelForEgress(rule.Egress, c.snapshot)).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
				case 5:
					c.smallPill(ctx, boolLabel(!rule.Disabled, "启用", "停用"), func() ui.Color {
						if rule.Disabled {
							return p.TextMuted
						}
						return p.Success
					}(), func() ui.Color {
						if rule.Disabled {
							return p.SurfaceAlt
						}
						return p.SuccessSoft
					}())
				case 6:
					ui.Row(ctx).Gap(4).Children(func() {
						if c.iconAction(ctx, iconEdit, "编辑规则 "+rule.Name, false).Clicked() {
							c.openEditor(editorRouteRule, rule.ID)
						}
						if c.iconAction(ctx, iconTrash, "删除规则 "+rule.Name, true).Clicked() {
							c.askDelete("rule", rule.ID, rule.Name)
						}
					})
				}
			}).Grow(1).Height(c.tableHeightForRows(ctx, len(rules)))
		})
	})
}

func (c *Controller) presetSwitch(ctx *ui.Context, label string, value bool, apply func(bool)) {
	p := c.colors()
	ui.Column(ctx).Gap(5).AlignItems(ui.Center).Children(func() {
		ui.Text(ctx, label).FontSize(10.5).TextColor(p.TextMuted).SingleLine()
		current := value
		if ui.Switch(ctx, &current).Label(label).Changed() {
			apply(current)
		}
	})
}

func (c *Controller) dnsPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "dns-profile", func() {
			c.sectionHeader(ctx, "全局设置", "策略与缓存属于控制面意图，不依赖当前内核状态", nil)
			ui.Row(ctx).FillWidth().Gap(18).AlignItems(ui.End).Children(func() {
				ui.Column(ctx).Width(180).Gap(6).Children(func() {
					ui.Text(ctx, "解析策略").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
					labels, values := dnsStrategyChoices()
					current := optionLabel(labels, values, c.snapshot.DNS.Strategy)
					selector := ui.Select(ctx, &current, labels).Label("解析策略")
					if selector.Changed() {
						for i, label := range labels {
							if label == current {
								c.setDNSStrategy(values[i])
								break
							}
						}
					}
				})
				ui.Column(ctx).Gap(6).Children(func() {
					ui.Text(ctx, "本地缓存").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
					cache := c.snapshot.DNS.Cache
					if ui.Switch(ctx, &cache).Label("本地缓存").Changed() {
						c.setDNSCache(cache)
					}
				})
				ui.Text(ctx, fmt.Sprintf("%d / 3 个槽位启用", dnsEnabledCount(c.snapshot.DNS))).FontSize(11.5).
					TextColor(p.TextMuted).Padding(0, 0, 8, 0)
			})
		})
		ui.Row(ctx).FillWidth().Gap(12).AlignItems(ui.Stretch).Children(func() {
			ui.Column(ctx).Grow(1).Basis(0).MinWidth(0).Children(func() {
				c.dnsSlotCard(ctx, "domestic", "国内 DNS", "处理国内域名与直连场景", c.snapshot.DNS.Domestic)
			})
			ui.Column(ctx).Grow(1).Basis(0).MinWidth(0).Children(func() {
				c.dnsSlotCard(ctx, "foreign", "国外 DNS", "处理代理场景与海外域名", c.snapshot.DNS.Foreign)
			})
			ui.Column(ctx).Grow(1).Basis(0).MinWidth(0).Children(func() {
				c.dnsSlotCard(ctx, "default", "默认 DNS", "前两个槽位未命中时使用", c.snapshot.DNS.Default)
			})
		})
		c.card(ctx, "dns-results", func() {
			c.sectionHeader(ctx, "最近探测", "点击“测试解析”后显示真实请求结果", nil)
			if len(c.dnsResults) == 0 {
				ui.Row(ctx).FillWidth().Padding(10, 0).AlignItems(ui.Center).Children(func() {
					ui.Text(ctx, "—").FontSize(15).TextColor(p.TextFaint).Width(32)
					ui.Text(ctx, "尚未执行解析探测").FontSize(11.5).TextColor(p.TextMuted).SingleLine()
				})
				return
			}
			for _, result := range c.dnsResults {
				status := duration(result.Latency)
				tone := p.Success
				if result.Skipped {
					status = result.Reason
					tone = p.TextMuted
				} else if result.Error != "" {
					status = oneLine(result.Error)
					tone = p.Danger
				}
				ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
					ui.Text(ctx, dnsSlotLabel(result.Key)).FontSize(11.5).FontWeight(600).Width(110).SingleLine()
					ui.Text(ctx, status).FontSize(11.5).TextColor(tone).Grow(1).SingleLine().Tooltip(status)
					ui.Text(ctx, shortTime(result.TestedAt)).FontSize(10.5).TextColor(p.TextFaint).SingleLine()
				})
			}
		})
	})
}

func (c *Controller) dnsSlotCard(ctx *ui.Context, slot, title, hint string, server control.DNSServerConfig) {
	p := c.colors()
	c.card(ctx, "dns-"+slot, func() {
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Column(ctx).Grow(1).Gap(3).Children(func() {
				ui.Text(ctx, title).FontSize(14).FontWeight(680).SingleLine()
				ui.Text(ctx, hint).FontSize(10.5).TextColor(p.TextFaint).SingleLine()
			})
			c.smallPill(ctx, boolLabel(server.Enabled, "启用", "停用"), func() ui.Color {
				if server.Enabled {
					return p.Success
				}
				return p.TextMuted
			}(), func() ui.Color {
				if server.Enabled {
					return p.SuccessSoft
				}
				return p.SurfaceAlt
			}())
		})
		ui.Text(ctx, valueOrDash(server.Address)).FontSize(13).Font("monospace").SingleLine().Tooltip(server.Address)
		ui.Row(ctx).Gap(6).Children(func() {
			c.smallPill(ctx, strings.ToUpper(string(server.Protocol)), p.Sky, p.SurfaceAlt)
			c.smallPill(ctx, dnsDetourLabel(server.Detour), p.TextMuted, p.SurfaceAlt)
			if server.FollowForeign {
				c.smallPill(ctx, "沿用国外", p.Violet, p.SurfaceAlt)
			}
		})
		ui.Row(ctx).FillWidth().Justify(ui.End).Children(func() {
			if c.labeledAction(ctx, iconEdit, "编辑", false).Clicked() {
				c.openEditor(editorDNS, slot)
			}
		})
	})
}

func (c *Controller) ruleSetsPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "rulesets-table", func() {
			if len(c.snapshot.RuleSets) == 0 {
				c.emptyState(ctx, "还没有规则集", "添加 source 或 binary 规则集后即可在路由规则中引用。", nil)
				return
			}
			ruleSets := append([]control.RuleSet(nil), c.snapshot.RuleSets...)
			sort.SliceStable(ruleSets, func(i, j int) bool { return strings.ToLower(ruleSets[i].Name) < strings.ToLower(ruleSets[j].Name) })
			cols := []ui.TableColumn{
				{Title: "名称", Width: 190},
				{Title: "格式", Width: 90},
				{Title: "大小", Width: 90, Align: ui.End},
				{Title: "更新时间", Width: 130},
				{Title: "状态", Width: 100},
				{Title: "操作", Width: 140, Fixed: true},
			}
			state := c.tableState("rulesets")
			ui.Table(ctx, state, cols, len(ruleSets), func(row, col int) {
				item := ruleSets[row]
				switch col {
				case 0:
					ui.Row(ctx).Gap(7).AlignItems(ui.Center).Children(func() {
						ui.Text(ctx, item.Name).FontSize(12.5).FontWeight(600).SingleLine().Tooltip(item.Name)
						if item.Builtin {
							c.smallPill(ctx, "内置", p.Violet, p.SurfaceAlt)
						}
					})
				case 1:
					ui.Text(ctx, string(item.Format)).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
				case 2:
					ui.Text(ctx, humanBytes(item.Status.SizeBytes)).FontSize(11.5).TextColor(p.TextMuted).TextAlign(ui.End)
				case 3:
					ui.Text(ctx, relativeTime(item.Status.UpdatedAt)).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
				case 4:
					status := boolLabel(item.Enabled, "启用", "停用")
					tone := p.Success
					fill := p.SuccessSoft
					if !item.Enabled {
						tone, fill = p.TextMuted, p.SurfaceAlt
					}
					if item.Status.Error != "" {
						status, tone, fill = "刷新失败", p.Danger, p.DangerSoft
					}
					c.smallPill(ctx, status, tone, fill)
				case 5:
					ui.Row(ctx).Gap(4).Children(func() {
						if c.iconAction(ctx, iconEdit, "编辑规则集 "+item.Name, false).Clicked() {
							c.openEditor(editorRuleSet, item.Name)
						}
						if c.iconAction(ctx, iconRefresh, "刷新规则集 "+item.Name, false).Clicked() {
							c.refreshRuleSet(item.Name)
						}
						if !item.Builtin {
							if c.iconAction(ctx, iconTrash, "删除规则集 "+item.Name, true).Clicked() {
								c.askDelete("ruleset", item.Name, item.Name)
							}
						}
					})
				}
			}).Grow(1).Height(c.tableHeightForRows(ctx, len(ruleSets)))
		})
	})
}

func (c *Controller) logsPage(ctx *ui.Context) {
	p := c.colors()
	ui.Column(ctx).Grow(1).FillWidth().MinWidth(0).Padding(22, 24, 18).Gap(12).Children(func() {
		logs := c.snapshot.mergedLogs(c.logFilter == 1, c.logFilter == 2)
		for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
			logs[i], logs[j] = logs[j], logs[i]
		}
		panel := ui.Column(ctx).Grow(1).MinWidth(0).Background(p.Surface).Border(1, p.Border).Radius(14).Clip()
		panel.Children(func() {
			ui.Row(ctx).FillWidth().Height(34).Padding(0, 12).Background(p.SurfaceAlt).
				BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSoft).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Text(ctx, fmt.Sprintf("%d 条记录 · 点击任意行复制", len(logs))).FontSize(10.5).
					TextColor(p.TextMuted).Grow(1).SingleLine()
				c.smallPill(ctx, []string{"全部日志", "应用日志", "内核日志"}[c.logFilter], p.TextMuted, p.Surface)
			})
			if len(logs) == 0 {
				c.emptyState(ctx, "暂无日志", "应用或内核产生日志后会自动出现在这里。", nil)
				return
			}
			ui.ScrollBoth(ctx).Key("logs-scroll").Grow(1).FillWidth().MinWidth(0).Padding(0).Children(func() {
				ui.Column(ctx).Key("logs-content").Label("日志内容").Width(c.logsContentWidth(ctx)).
					Gap(1).Padding(5, 6, 24, 6).Children(func() {
					for index, entry := range logs {
						levelColor := logLevelColor(p, entry.Level)
						background := p.Surface
						if index%2 == 1 {
							background = p.SurfaceAlt
						}
						line := entry.Time.Local().Format("2006-01-02 15:04:05.000") + "  " +
							strings.ToUpper(entry.Level) + "  " + oneLine(entry.Message)
						row := ui.ButtonBase(ctx).Key("log-"+entry.ID).Label(line).FillWidth().Height(27).
							Padding(0, 8).Radius(5).Gap(0).AlignItems(ui.Center).Background(background).
							Tooltip("点击复制此日志")
						row.Children(func() {
							ui.Text(ctx, entry.Time.Local().Format("01-02 15:04:05.000")).FontSize(10.5).
								Font("monospace").TextColor(p.TextFaint).Width(148).SingleLine()
							ui.Text(ctx, strings.ToUpper(entry.Level)).FontSize(10).FontWeight(700).
								TextColor(levelColor).Width(72).SingleLine()
							ui.Text(ctx, oneLine(entry.Message)).FontSize(10.5).Font("monospace").
								TextColor(p.Text).Grow(1).MinWidth(0).SingleLine().Tooltip(entry.Message)
						})
						if row.Clicked() {
							c.copyLog(entry)
						}
					}
				})
			})
		})
	})
}

func (c *Controller) logsContentWidth(ctx *ui.Context) float32 {
	const minimum = 1560
	width, _ := ctx.Size()
	// The page scroll and log panel each contribute 24px, and the panel has
	// a 1px border on both sides. Below the minimum, ScrollBoth keeps the
	// full log line available through horizontal scrolling.
	return max(minimum, width-c.sidebarWidth()-50)
}

func (c *Controller) connectionsPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "connections-table", func() {
			if c.snapshot.Kernel.State != kernel.StateRunning {
				ui.Row(ctx).FillWidth().Padding(12, 0).AlignItems(ui.Center).Children(func() {
					ui.Text(ctx, "—").FontSize(16).TextColor(p.TextFaint).Width(34)
					ui.Text(ctx, "内核未启动，未采集数据").FontSize(12).TextColor(p.TextMuted).SingleLine()
				})
				return
			}
			if len(c.snapshot.Connections) == 0 {
				c.emptyState(ctx, "当前没有活动连接", "新的代理连接会自动出现在这里。", nil)
				return
			}
			conns := append([]kernel.Connection(nil), c.snapshot.Connections...)
			sort.Slice(conns, func(i, j int) bool { return conns[i].StartedAt.After(conns[j].StartedAt) })
			cols := []ui.TableColumn{
				{Title: "网络", Width: 62},
				{Title: "来源", Width: 180},
				{Title: "目标", MinWidth: 240},
				{Title: "出口", Width: 130},
				{Title: "规则", Width: 130},
				{Title: "进程", Width: 140},
				{Title: "流量", Width: 90, Align: ui.End},
				{Title: "操作", Width: 58, Fixed: true},
			}
			state := c.tableState("connections")
			ui.Table(ctx, state, cols, len(conns), func(row, col int) {
				conn := conns[row]
				switch col {
				case 0:
					c.smallPill(ctx, strings.ToUpper(conn.Network), p.Sky, p.SurfaceAlt)
				case 1:
					tableMonoCell(ctx, conn.Source, p.TextMuted)
				case 2:
					tableMonoCell(ctx, conn.Target, p.Text)
				case 3:
					ui.Text(ctx, conn.Outbound).FontSize(11.5).TextColor(p.TextMuted).SingleLine().Tooltip(conn.Outbound)
				case 4:
					ui.Text(ctx, valueOrDash(conn.Rule)).FontSize(11.5).TextColor(p.TextMuted).SingleLine().Tooltip(conn.Rule)
				case 5:
					ui.Text(ctx, valueOrDash(conn.Process)).FontSize(11.5).TextColor(p.TextMuted).SingleLine().Tooltip(conn.Process)
				case 6:
					ui.Text(ctx, humanBytes(conn.Upload+conn.Download)).FontSize(11.5).TextColor(p.TextMuted).TextAlign(ui.End)
				case 7:
					if c.iconAction(ctx, iconClose, "关闭连接 "+conn.Target, true).Clicked() {
						c.closeConnection(conn.ID)
					}
				}
			}).Grow(1).Height(c.tableHeightForRows(ctx, len(conns)))
		})
	})
}

func (c *Controller) settingsPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		ui.Row(ctx).FillWidth().Gap(14).AlignItems(ui.Stretch).Children(func() {
			ui.Column(ctx).Grow(3).Basis(0).MinWidth(0).Children(func() {
				c.card(ctx, "settings-sync", func() {
					c.sectionHeader(ctx, "订阅同步", "这些设置会影响所有订阅的统一同步行为", func() {
						c.smallPill(ctx, boolLabel(c.snapshot.Scheduler.Running, "调度器运行中", "调度器已停止"), func() ui.Color {
							if c.snapshot.Scheduler.Running {
								return p.Success
							}
							return p.TextMuted
						}(), func() ui.Color {
							if c.snapshot.Scheduler.Running {
								return p.SuccessSoft
							}
							return p.SurfaceAlt
						}())
						if c.labeledAction(ctx, iconPlay, "立即同步", false).Clicked() {
							c.runSchedulerOnce()
						}
					})
					autoRefresh := settingBool(c.settingsDraft, control.SettingAutoRefresh, true)
					ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
						ui.Column(ctx).Grow(1).Gap(2).Children(func() {
							ui.Text(ctx, "自动同步").FontSize(12.5).FontWeight(600).SingleLine()
							ui.Text(ctx, "按订阅各自的间隔后台刷新节点").FontSize(10.5).TextColor(p.TextFaint).SingleLine()
						})
						if ui.Switch(ctx, &autoRefresh).Label("自动同步").Changed() {
							c.settingsDraft[control.SettingAutoRefresh] = strconv.FormatBool(autoRefresh)
						}
					})
					ui.Row(ctx).FillWidth().Gap(16).AlignItems(ui.End).Children(func() {
						ui.Column(ctx).Width(160).Gap(6).Children(func() {
							ui.Text(ctx, "失败重试次数").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
							value := c.settingsDraft[control.SettingSubscriptionMaxAttempts]
							ui.TextInput(ctx, &value).Placeholder("3").Label("失败重试次数")
							c.settingsDraft[control.SettingSubscriptionMaxAttempts] = value
						})
						ui.Column(ctx).Grow(1).MinWidth(240).Gap(6).Children(func() {
							ui.Text(ctx, "节点冲突策略").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
							labels := []string{"保留本地修改", "以远端为准"}
							values := []string{control.SubscriptionConflictKeepLocal, control.SubscriptionConflictFollowRemote}
							current := conflictPolicyLabel(c.settingsDraft[control.SettingSubscriptionConflictPolicy])
							selector := ui.Select(ctx, &current, labels).Label("节点冲突策略")
							if selector.Changed() {
								for i, label := range labels {
									if label == current {
										c.settingsDraft[control.SettingSubscriptionConflictPolicy] = values[i]
										break
									}
								}
							}
						})
					})
					ui.Row(ctx).FillWidth().Justify(ui.End).Children(func() {
						if c.labeledAction(ctx, iconCheck, "保存同步设置", true).Clicked() {
							c.saveSettings()
						}
					})
				})
			})
			ui.Column(ctx).Grow(2).Basis(0).MinWidth(0).Children(func() {
				c.card(ctx, "settings-appearance", func() {
					c.sectionHeader(ctx, "界面", "原生 UI 跟随这里的偏好即时切换", nil)
					ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
						ui.Column(ctx).Grow(1).Gap(2).Children(func() {
							ui.Text(ctx, "深色模式").FontSize(12.5).FontWeight(600).SingleLine()
							ui.Text(ctx, "默认使用白底与克制的品牌渐变").FontSize(10.5).TextColor(p.TextFaint).SingleLine()
						})
						dark := c.dark
						if ui.Switch(ctx, &dark).Label("深色模式").Changed() {
							c.toggleTheme()
						}
					})
				})
			})
		})
		c.card(ctx, "settings-runtime", func() {
			c.sectionHeader(ctx, "本地环境", "桌面进程直接持有控制面，不经过 HTTP 管理端口", nil)
			c.pathRow(ctx, "数据目录", c.runtime.DataDir)
			c.pathRow(ctx, "数据库", c.runtime.DatabasePath)
			c.pathRow(ctx, "控制面架构", "单进程 · SQLite · sing-box")
		})
	})
}

func (c *Controller) pathRow(ctx *ui.Context, label, value string) {
	p := c.colors()
	ui.Row(ctx).FillWidth().Padding(8, 0).AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Text(ctx, label).FontSize(11.5).TextColor(p.TextMuted).Width(100).SingleLine()
		ui.Text(ctx, valueOrDash(value)).FontSize(11.5).Font("monospace").Grow(1).SingleLine().Tooltip(value)
	})
}

func (c *Controller) aboutPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		ui.Row(ctx).FillWidth().Gap(14).AlignItems(ui.Stretch).Children(func() {
			ui.Column(ctx).Grow(2).Basis(0).MinWidth(0).Children(func() {
				c.card(ctx, "about-product", func() {
					ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(14).Children(func() {
						ui.Image(ctx, logoBitmap).Size(52, 52)
						ui.Column(ctx).Grow(1).Gap(3).Children(func() {
							ui.Text(ctx, "BOXD").FontSize(22).FontWeight(760).SingleLine()
							ui.Text(ctx, "网络控制平面 · 原生桌面客户端").FontSize(12).TextColor(p.TextMuted).SingleLine()
						})
						c.smallPill(ctx, "Apache-2.0", p.TextMuted, p.SurfaceAlt)
					})
					ui.Text(ctx, "BOXD 将订阅、节点、出口编排、路由和 DNS 收敛为可审计的本地控制面。桌面端直接调用同一控制面接口，不依赖 Web 登录或 HTTP 管理端口。").FontSize(12).TextColor(p.TextMuted).Wrap().LineHeight(1.55)
					ui.Row(ctx).Gap(8).Children(func() {
						c.smallPill(ctx, c.runtime.Version, p.Indigo, p.AccentSoft)
						c.smallPill(ctx, "commit "+shortVersion(c.runtime.Commit), p.TextMuted, p.SurfaceAlt)
						c.smallPill(ctx, "built "+shortVersion(c.runtime.BuiltAt), p.TextMuted, p.SurfaceAlt)
					})
				})
				c.card(ctx, "about-capabilities", func() {
					c.sectionHeader(ctx, "内核能力", "来自当前引擎的能力描述", nil)
					capabilities := []struct {
						label string
						on    bool
					}{
						{"配置热重载", c.snapshot.Capabilities.Capabilities.HotReload},
						{"Clash 模式", c.snapshot.Capabilities.Capabilities.ClashMode},
						{"DNS 刷新", c.snapshot.Capabilities.Capabilities.DNSFlush},
						{"连接控制", c.snapshot.Capabilities.Capabilities.ConnectionControl},
						{"流量观测", c.snapshot.Capabilities.Capabilities.TrafficObservation},
						{"配置诊断", c.snapshot.Capabilities.Capabilities.ConfigDiagnostics},
					}
					for start := 0; start < len(capabilities); start += 3 {
						end := min(start+3, len(capabilities))
						ui.Row(ctx).FillWidth().Gap(8).Children(func() {
							for _, capability := range capabilities[start:end] {
								tone, fill := p.TextMuted, p.SurfaceAlt
								if capability.on {
									tone, fill = p.Success, p.SuccessSoft
								}
								c.smallPill(ctx, boolLabel(capability.on, "✓ ", "· ")+capability.label, tone, fill)
							}
						})
					}
				})
			})
			ui.Column(ctx).Grow(2).Basis(0).MinWidth(0).Children(func() {
				c.card(ctx, "about-kernel", func() {
					c.sectionHeader(ctx, "内核", "运行状态与引擎版本", nil)
					ui.Text(ctx, kernelStateLabel(c.snapshot.Kernel.State)).FontSize(18).FontWeight(700).SingleLine()
					ui.Row(ctx).FillWidth().Gap(12).Children(func() {
						c.statTile(ctx, "about-engine", "引擎", valueOrDash(string(c.snapshot.Capabilities.Kind)), valueOrDash(c.snapshot.Capabilities.Version), p.Indigo)
						c.statTile(ctx, "about-kernel-version", "内核版本", shortVersion(c.snapshot.Kernel.Version), "运行实例报告", p.Sky)
					})
					c.pathRow(ctx, "启动时间", fullTime(c.snapshot.Kernel.StartedAt))
					c.pathRow(ctx, "配置版本", valueOrDash(shortVersion(c.snapshot.Kernel.Revision)))
					if c.snapshot.Kernel.LastError != nil {
						ui.Text(ctx, oneLine(c.snapshot.Kernel.LastError.Error())).FontSize(11.5).TextColor(p.Danger).Wrap()
					}
				})
				c.card(ctx, "about-audit", func() {
					c.sectionHeader(ctx, "最近审计", "控制面自动记录的资源变更", nil)
					if len(c.snapshot.Audit) == 0 {
						ui.Row(ctx).FillWidth().Padding(10, 0).AlignItems(ui.Center).Children(func() {
							ui.Text(ctx, "—").FontSize(15).TextColor(p.TextFaint).Width(32)
							ui.Text(ctx, "暂无可展示的审计记录").FontSize(11.5).TextColor(p.TextMuted).SingleLine()
						})
						return
					}
					for _, event := range c.snapshot.Audit[:min(6, len(c.snapshot.Audit))] {
						ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
							ui.Text(ctx, shortTime(event.CreatedAt)).FontSize(10).Font("monospace").TextColor(p.TextFaint).Width(72)
							ui.Text(ctx, event.Action).FontSize(11).FontWeight(600).TextColor(p.TextMuted).Width(150).SingleLine()
							ui.Text(ctx, event.Actor).FontSize(10.5).TextColor(p.TextFaint).Grow(1).SingleLine()
						})
					}
				})
			})
		})
	})
}

func nodeName(snapshot snapshot, id string) string {
	if id == "" {
		return "未选择"
	}
	if node, ok := snapshot.NodesByID[id]; ok {
		return node.Name
	}
	return id
}

func egressOptionLabel(ref control.EgressRef, s snapshot) string {
	switch ref.Kind {
	case control.EgressKindDirect:
		return "直连"
	case control.EgressKindReject:
		return "拒绝"
	case control.EgressKindGroup:
		if group, ok := s.GroupsByID[ref.ID]; ok {
			return "策略组 · " + group.Name
		}
	case control.EgressKindChain:
		if chain, ok := s.ChainsByID[ref.ID]; ok {
			return "链路 · " + chain.Name
		}
	case control.EgressKindNode:
		if node, ok := s.NodesByID[ref.ID]; ok {
			return "节点 · " + node.Name
		}
	}
	return ref.ID
}

func routeMatchSummary(match control.RouteMatch) string {
	parts := make([]string, 0, 8)
	if len(match.Domains) > 0 {
		parts = append(parts, "域名 "+strings.Join(match.Domains, ", "))
	}
	if len(match.DomainSuffixes) > 0 {
		parts = append(parts, "后缀 "+strings.Join(match.DomainSuffixes, ", "))
	}
	if len(match.DomainKeywords) > 0 {
		parts = append(parts, "关键字 "+strings.Join(match.DomainKeywords, ", "))
	}
	if len(match.IPCIDRs) > 0 {
		parts = append(parts, "IP "+strings.Join(match.IPCIDRs, ", "))
	}
	if len(match.RuleSets) > 0 {
		parts = append(parts, "规则集 "+strings.Join(match.RuleSets, ", "))
	}
	if len(match.Ports) > 0 {
		portText := make([]string, 0, len(match.Ports))
		for _, port := range match.Ports {
			portText = append(portText, strconv.Itoa(int(port)))
		}
		parts = append(parts, "端口 "+strings.Join(portText, ", "))
	}
	if len(parts) == 0 {
		return "匹配全部流量"
	}
	return strings.Join(parts, " · ")
}

func dnsSlotLabel(slot string) string {
	switch slot {
	case "domestic":
		return "国内 DNS"
	case "foreign":
		return "国外 DNS"
	case "default":
		return "默认 DNS"
	default:
		return slot
	}
}

func dnsDetourLabel(detour control.DNSDetour) string {
	switch detour {
	case control.DNSDetourDirect:
		return "直连"
	case control.DNSDetourEgress:
		return "指定出口"
	case control.DNSDetourRouteDefault:
		return "跟随路由"
	default:
		return string(detour)
	}
}

func conflictPolicyLabel(value string) string {
	if value == control.SubscriptionConflictFollowRemote {
		return "以远端为准"
	}
	return "保留本地修改"
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

var _ = time.Now
