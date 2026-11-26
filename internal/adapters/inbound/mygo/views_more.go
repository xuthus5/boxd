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
		c.card(ctx, "egress-workspace", func() {
			c.sectionHeader(ctx, "出口资源", "策略组决定出口选择方式，代理链路定义实际经过的跳点顺序", nil)
			c.sectionHeader(ctx, "策略组", fmt.Sprintf("%d 个策略组 · 可手动选择或自动测速", len(c.snapshot.Catalog.Groups)), nil)
			if len(c.snapshot.Catalog.Groups) == 0 {
				c.emptyState(ctx, "尚未创建策略组", "策略组可以把多个节点组合为手动选择或自动测速出口。", nil)
			} else {
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
			}
			c.sectionDivider(ctx)
			c.sectionHeader(ctx, "代理链路", fmt.Sprintf("%d 条链路 · 按顺序串联多个出口", len(c.snapshot.Catalog.Chains)), nil)
			if len(c.snapshot.Catalog.Chains) == 0 {
				c.emptyState(ctx, "还没有代理链路", "链路用于串联多个出口，例如本地节点 → 中转节点 → 目标网络。", nil)
			} else {
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
			}
		})
	})
}

func (c *Controller) routesPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "routes-workspace", func() {
			c.sectionHeader(ctx, "基础策略", "运行模式与默认出口决定未命中规则时的流量去向", nil)
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
			c.sectionDivider(ctx)
			c.sectionHeader(ctx, "自定义规则", fmt.Sprintf("%d 条规则 · 按显示顺序自上而下匹配", len(c.snapshot.Routes.Rules)), nil)
			if len(c.snapshot.Routes.Rules) == 0 {
				c.emptyState(ctx, "还没有自定义规则", "预设策略会继续生效；添加规则可实现更精细的分流。", nil)
				return
			}
			rules := c.snapshot.Routes.Rules
			orderWidth, nameWidth, matchWidth, actionWidth, targetWidth, statusWidth, operationWidth := float32(72), float32(180), float32(280), float32(92), float32(160), float32(72), float32(76)
			if c.narrow {
				orderWidth, nameWidth, matchWidth, actionWidth, targetWidth, statusWidth, operationWidth = 56, 140, 210, 70, 110, 60, 80
			}
			cols := []ui.TableColumn{
				{Title: "顺序", Width: orderWidth, Fixed: true},
				{Title: "名称", Width: nameWidth},
				{Title: "匹配条件", MinWidth: matchWidth},
				{Title: "动作", Width: actionWidth},
				{Title: "目标出口", Width: targetWidth},
				{Title: "状态", Width: statusWidth},
				{Title: "操作", Width: operationWidth, Fixed: true},
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
		c.card(ctx, "dns-workspace", func() {
			c.sectionHeader(ctx, "全局解析策略", "统一控制地址族偏好、本地缓存与三个解析槽位的启停", nil)
			ui.Row(ctx).FillWidth().Gap(18).AlignItems(ui.End).Children(func() {
				ui.Column(ctx).Width(200).Gap(6).Children(func() {
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
				ui.Box(ctx).Grow(1)
				c.smallPill(ctx, fmt.Sprintf("%d / 3 个槽位启用", dnsEnabledCount(c.snapshot.DNS)), p.TextMuted, p.SurfaceAlt)
			})
			c.sectionDivider(ctx)
			c.sectionHeader(ctx, "解析槽位", "每类场景独立维护协议、地址与出口策略", nil)
			ui.Row(ctx).FillWidth().AlignItems(ui.Stretch).Gap(18).Children(func() {
				c.dnsSlotBlock(ctx, "domestic", "国内 DNS", "国内域名与直连场景", c.snapshot.DNS.Domestic)
				ui.Box(ctx).Width(1).Height(112).Background(p.BorderSoft)
				c.dnsSlotBlock(ctx, "foreign", "国外 DNS", "代理场景与海外域名", c.snapshot.DNS.Foreign)
				ui.Box(ctx).Width(1).Height(112).Background(p.BorderSoft)
				c.dnsSlotBlock(ctx, "default", "默认 DNS", "前两个槽位未命中时使用", c.snapshot.DNS.Default)
			})
			c.sectionDivider(ctx)
			c.sectionHeader(ctx, "最近探测", "点击页头的“测试解析”后显示真实请求结果", nil)
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

func (c *Controller) dnsSlotBlock(ctx *ui.Context, slot, title, hint string, server control.DNSServerConfig) {
	p := c.colors()
	ui.Column(ctx).Key("dns-" + slot).Grow(1).Basis(0).MinWidth(0).Gap(9).Children(func() {
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Column(ctx).Grow(1).Gap(3).Children(func() {
				ui.Text(ctx, title).FontSize(13).FontWeight(660).SingleLine()
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
		ui.Text(ctx, valueOrDash(server.Address)).FontSize(12).Font("monospace").TextColor(p.TextMuted).
			SingleLine().Tooltip(server.Address)
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
			enabled := 0
			for _, ruleSet := range c.snapshot.RuleSets {
				if ruleSet.Enabled {
					enabled++
				}
			}
			c.sectionHeader(ctx, "规则资源", fmt.Sprintf("%d 个规则集 · %d 个已启用", len(c.snapshot.RuleSets), enabled), nil)
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
	ui.Column(ctx).Grow(1).FillWidth().MinWidth(0).Padding(28).Gap(12).Children(func() {
		logs := c.snapshot.mergedLogs(c.logFilter == 1, c.logFilter == 2)
		for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
			logs[i], logs[j] = logs[j], logs[i]
		}
		panel := ui.Column(ctx).Grow(1).MinWidth(0).Background(p.Surface).Border(1, p.Border).Radius(14).Clip()
		panel.Children(func() {
			ui.Row(ctx).FillWidth().Height(40).Padding(0, 14).Background(p.SurfaceAlt).
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
	// The page padding and log panel together account for 58px, and the panel
	// has a 1px border on both sides. Below the minimum, ScrollBoth keeps the
	// full log line available through horizontal scrolling.
	return max(minimum, width-c.sidebarWidth()-58)
}

func (c *Controller) connectionsPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "connections-table", func() {
			c.sectionHeader(ctx, "连接明细", "数据来自当前内核连接表，并按建立时间倒序排列", nil)
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
			networkWidth, sourceWidth, targetWidth, outboundWidth := float32(62), float32(180), float32(240), float32(130)
			ruleWidth, processWidth, trafficWidth, actionWidth := float32(130), float32(140), float32(90), float32(58)
			if c.narrow {
				networkWidth, sourceWidth, targetWidth, outboundWidth = 64, 120, 160, 90
				ruleWidth, processWidth, trafficWidth, actionWidth = 80, 80, 64, 50
			}
			cols := []ui.TableColumn{
				{Title: "网络", Width: networkWidth},
				{Title: "来源", Width: sourceWidth},
				{Title: "目标", MinWidth: targetWidth},
				{Title: "出口", Width: outboundWidth},
				{Title: "规则", Width: ruleWidth},
				{Title: "进程", Width: processWidth},
				{Title: "流量", Width: trafficWidth, Align: ui.End},
				{Title: "操作", Width: actionWidth, Fixed: true},
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
		c.card(ctx, "settings-workspace", func() {
			c.sectionHeader(ctx, "订阅同步", "统一管理所有订阅的后台刷新、重试与节点冲突处理", func() {
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
			ui.Row(ctx).FillWidth().Gap(18).AlignItems(ui.End).Children(func() {
				ui.Column(ctx).Width(180).Gap(6).Children(func() {
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
			c.sectionDivider(ctx)
			c.inboundsSettingsSection(ctx)
			c.sectionDivider(ctx)
			c.sectionHeader(ctx, "外观", "界面偏好即时生效并保存在本地", nil)
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
			c.sectionDivider(ctx)
			c.sectionHeader(ctx, "本地环境", "应用数据与配置仅保存在当前设备", nil)
			c.pathRow(ctx, "数据目录", c.runtime.DataDir)
			c.pathRow(ctx, "数据库", c.runtime.DatabasePath)
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
		c.card(ctx, "about-product", func() {
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(16).Children(func() {
				ui.Image(ctx, logoBitmap).Size(56, 56)
				ui.Column(ctx).Grow(1).Gap(4).Children(func() {
					ui.Text(ctx, "BOXD").FontSize(24).FontWeight(740).SingleLine()
					ui.Text(ctx, "本地网络控制面 · 原生桌面客户端").FontSize(12).TextColor(p.TextMuted).SingleLine()
				})
				c.smallPill(ctx, c.project.License, p.TextMuted, p.SurfaceAlt)
			})
			ui.Text(ctx, "统一管理订阅、节点、出口编排、路由与 DNS，让网络配置保持清晰、可审计，并将运行状态留在当前设备。").
				FontSize(12.5).TextColor(p.TextMuted).Wrap().LineHeight(1.6)
			c.sectionDivider(ctx)
			c.sectionHeader(ctx, "版本信息", "当前安装的桌面客户端构建", nil)
			ui.Row(ctx).FillWidth().Gap(24).AlignItems(ui.Stretch).Children(func() {
				c.aboutMeta(ctx, "版本", valueOrDash(c.runtime.Version))
				c.aboutMeta(ctx, "提交", shortVersion(c.runtime.Commit))
				c.aboutMeta(ctx, "构建时间", fullTimeFromText(c.runtime.BuiltAt))
			})
			c.sectionDivider(ctx)
			c.aboutRepositorySection(ctx)
		})
	})
}

func (c *Controller) aboutMeta(ctx *ui.Context, label, value string) {
	p := c.colors()
	ui.Column(ctx).Grow(1).Basis(0).MinWidth(0).Gap(5).Children(func() {
		ui.Text(ctx, label).FontSize(10.5).FontWeight(600).TextColor(p.TextFaint).SingleLine()
		ui.Text(ctx, value).FontSize(12.5).Font("monospace").TextColor(p.TextMuted).
			SingleLine().Tooltip(value)
	})
}

func (c *Controller) aboutRepositorySection(ctx *ui.Context) {
	p := c.colors()
	c.sectionHeader(ctx, "GitHub 仓库", "查看源码、反馈问题并获取正式版本", nil)
	ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Row(ctx).Size(36, 36).Radius(10).Background(p.SurfaceAlt).Center().Children(func() {
			ui.Icon(ctx, iconRepository).FontSize(18).TextColor(p.Accent)
		})
		ui.Column(ctx).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(ctx, c.project.Slug).FontSize(14).FontWeight(680).SingleLine().Tooltip(c.project.URL)
			ui.Text(ctx, strings.TrimPrefix(c.project.URL, "https://")).FontSize(10.5).
				Font("monospace").TextColor(p.TextFaint).SingleLine().Tooltip(c.project.URL)
		})
	})
	c.aboutUpdateStatus(ctx)
	ui.Row(ctx).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		if c.labeledAction(ctx, iconRepository, "打开仓库", false).Clicked() {
			c.openExternal(ctx, c.project.URL, "打开仓库")
		}
		if c.labeledAction(ctx, iconArrowRight, "问题反馈", false).Clicked() {
			c.openExternal(ctx, c.project.IssuesURL, "打开问题反馈")
		}
		ui.Box(ctx).Grow(1)
		checking := c.update.state == updateChecking || c.options.UpdateChecker == nil
		if c.labeledAction(ctx, iconRefresh, "检查更新", true).Disabled(checking).Clicked() {
			c.checkForUpdates()
		}
	})
}

func (c *Controller) aboutUpdateStatus(ctx *ui.Context) {
	p := c.colors()
	tone, fill := p.TextMuted, p.SurfaceAlt
	switch c.update.state {
	case updateChecking:
		tone, fill = p.Accent, p.AccentSoft
	case updateCurrent:
		tone, fill = p.Success, p.SuccessSoft
	case updateAvailable:
		tone, fill = p.Success, p.SuccessSoft
	case updateNoRelease, updateUnknown:
		tone, fill = p.Warning, p.WarningSoft
	case updateFailed:
		tone, fill = p.Danger, p.DangerSoft
	}
	if c.options.UpdateChecker == nil {
		tone, fill = p.TextMuted, p.SurfaceAlt
	}
	ui.Row(ctx).FillWidth().Padding(12, 14).Radius(12).Background(fill).
		Border(1, tone.Alpha(0.16)).AlignItems(ui.Center).Gap(10).Children(func() {
		if c.update.state == updateChecking {
			ui.Spinner(ctx).Size(16, 16).TextColor(tone)
		} else {
			icon := iconRefresh
			switch c.update.state {
			case updateCurrent, updateAvailable:
				icon = iconCheck
			case updateNoRelease, updateUnknown, updateFailed:
				icon = iconAlert
			}
			ui.Icon(ctx, icon).FontSize(16).TextColor(tone)
		}
		ui.Column(ctx).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(ctx, c.updateTitle()).FontSize(12).FontWeight(650).TextColor(tone).SingleLine()
			ui.Text(ctx, c.updateDetail()).FontSize(10.5).TextColor(p.TextMuted).SingleLine().Tooltip(c.updateDetail())
		})
		switch c.update.state {
		case updateAvailable, updateUnknown:
			if c.tonedAction(ctx, iconArrowRight, "查看版本", tone, fill).Clicked() {
				c.openExternal(ctx, c.updateReleaseURL(), "打开发布版本")
			}
		case updateNoRelease:
			if c.tonedAction(ctx, iconArrowRight, "发布记录", tone, fill).Clicked() {
				c.openExternal(ctx, c.project.ReleasesURL, "打开发布记录")
			}
		}
	})
}

func (c *Controller) openExternal(ctx *ui.Context, url, action string) {
	ctx.OpenURLThen(url, func(err error) {
		if err != nil {
			c.setNotice(action+"失败："+err.Error(), true)
		}
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

func fullTimeFromText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "—"
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return fullTime(parsed)
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

var _ = time.Now
