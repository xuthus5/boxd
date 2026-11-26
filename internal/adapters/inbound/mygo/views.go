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

// View is the root of the native window. It only reads controller state and
// dispatches host actions; all persisted data comes from control.Application.
func (c *Controller) View(ctx *ui.Context) {
	p := c.applyTheme(ctx)
	ctx.Root().Background(p.Background).TextColor(p.Text)
	width, _ := ctx.Size()
	narrow := width < 1080
	c.setNarrow(narrow)

	ui.Column(ctx).Fill().Children(func() {
		bar := ctx.TitleBar()
		if bar.Height > 0 || bar.Bottom > 0 {
			ui.Row(ctx).Height(max(bar.Height, 38)).FillWidth().Padding(0, bar.Right+18, 0, bar.Left+18).
				Background(p.Background).DragWindow().AlignItems(ui.Center).Children(func() {
				ui.Row(ctx).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(ctx, "BOXD").FontSize(11).FontWeight(700).TextColor(p.TextMuted).SingleLine()
					ui.Text(ctx, "DESKTOP").FontSize(8.5).FontWeight(650).TextColor(p.TextFaint).SingleLine()
				})
			})
		}
		ui.Row(ctx).Fill().AlignItems(ui.Stretch).Children(func() {
			c.sidebar(ctx, narrow)
			ui.Column(ctx).Grow(1).MinWidth(0).Children(func() {
				c.header(ctx)
				if c.notice != "" {
					c.noticeBanner(ctx)
				}
				if c.runtime == nil && c.loading {
					c.bootView(ctx)
					return
				}
				if c.runtime == nil {
					c.unavailableView(ctx)
					return
				}
				c.pageView(ctx)
			})
		})
	})
	c.dialogs(ctx)
}

func (c *Controller) sidebar(ctx *ui.Context, narrow bool) {
	p := c.colors()
	_, height := ctx.Size()
	compact := height < 760
	width := float32(216)
	if narrow {
		width = 82
	}
	itemHeight := float32(38)
	if compact {
		itemHeight = 34
	}
	side := ui.Column(ctx).Width(width).FillHeight().Background(p.Sidebar).
		BorderWidth(0, 0, 0, 1).BorderColor(p.Border).Padding(14, 12).Gap(6).Shrink(0)
	side.Children(func() {
		ui.Row(ctx).Height(52).Padding(0, 4).AlignItems(ui.Center).Gap(11).Children(func() {
			ui.Image(ctx, logoBitmap).Size(34, 34)
			if !narrow {
				ui.Column(ctx).Gap(1).Children(func() {
					ui.Text(ctx, "BOXD").FontSize(17).FontWeight(760).SingleLine()
					ui.Text(ctx, "CONTROL PLANE").FontSize(8.5).FontWeight(700).TextColor(p.TextFaint).SingleLine()
				})
			}
		})
		ui.Box(ctx).Height(10)
		ui.Column(ctx).Grow(1).Gap(3).Children(func() {
			for _, item := range navigation {
				selected := c.page == item.ID
				button := ui.ButtonBase(ctx).Key("nav-"+item.ID).Label(item.Label).Height(itemHeight).
					Padding(0, 11).Radius(10).Gap(11).AlignItems(ui.Center).Tooltip(item.Label)
				if selected {
					button.Background(p.AccentSoft).Border(1, p.Accent.Alpha(0.18)).TextColor(p.Accent)
				} else {
					button.TextColor(p.TextMuted)
				}
				button.Children(func() {
					iconColor := p.TextMuted
					if selected {
						iconColor = p.Accent
					}
					ui.Icon(ctx, navIcon(item.ID)).FontSize(18).TextColor(iconColor)
					if !narrow {
						ui.Text(ctx, item.Label).FontSize(13).FontWeight(func() int {
							if selected {
								return 650
							}
							return 550
						}()).TextColor(iconColor).SingleLine().Grow(1)
					}
				})
				if button.Clicked() {
					c.show(item.ID)
				}
			}
		})
		ui.Column(ctx).Gap(7).Padding(10, 8).Radius(12).Background(p.SurfaceAlt).Children(func() {
			if narrow {
				ui.Text(ctx, compactVersion(c.version())).FontSize(9).TextColor(p.TextFaint).TextAlign(ui.Center).
					SingleLine().Tooltip("BOXD " + c.version())
				return
			}
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(7).Children(func() {
				ui.Box(ctx).Size(6, 6).Radius(6).Background(p.Success)
				ui.Text(ctx, "控制面已连接").FontSize(10.5).FontWeight(600).TextColor(p.TextMuted).SingleLine()
			})
			ui.Text(ctx, c.clock.Now().Local().Format("2006-01-02 15:04:05")).FontSize(10.5).
				Font("monospace").TextColor(p.TextFaint).SingleLine()
		})
	})
}

func (c *Controller) header(ctx *ui.Context) {
	p := c.colors()
	title, hint := c.pageTitle()
	ui.Row(ctx).Height(72).FillWidth().Padding(0, 24).Background(p.Surface).
		BorderWidth(0, 0, 1, 0).BorderColor(p.Border).AlignItems(ui.Center).Gap(16).Shrink(0).Children(func() {
		ui.Column(ctx).Grow(1).Gap(3).Children(func() {
			ui.Text(ctx, title).FontSize(18).FontWeight(700).SingleLine()
			ui.Text(ctx, hint).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
		})
		c.pageHeaderActions(ctx)
		if c.busy {
			ui.Spinner(ctx).Size(18, 18).TextColor(p.Accent)
		}
		c.statusPill(ctx, string(c.snapshot.Kernel.State))
		themeIcon, themeLabel := iconMoon, "切换深色模式"
		if c.dark {
			themeIcon, themeLabel = iconSun, "切换浅色模式"
		}
		themeButton := c.iconAction(ctx, themeIcon, themeLabel, false)
		if themeButton.Clicked() {
			c.toggleTheme()
		}
		refresh := c.iconAction(ctx, iconRefresh, "刷新数据", false)
		if refresh.Clicked() {
			c.refresh(true)
		}
	})
}

func (c *Controller) pageHeaderActions(ctx *ui.Context) {
	switch c.page {
	case pageSubscriptions:
		if c.labeledAction(ctx, iconRefresh, "刷新全部", false).Clicked() {
			c.refreshAllSubscriptions()
		}
		if c.labeledAction(ctx, iconPlus, "添加订阅", true).Clicked() {
			c.openEditor(editorSubscription, "")
		}
	case pageNodes:
		if c.labeledAction(ctx, iconPlus, "导入链接", false).Clicked() {
			c.openEditor(editorImportLinks, "")
		}
		if c.labeledAction(ctx, iconPlus, "添加节点", true).Clicked() {
			c.openEditor(editorNode, "")
		}
	case pageEgress:
		if c.labeledAction(ctx, iconPlus, "新建策略组", false).Clicked() {
			c.openEditor(editorGroup, "")
		}
		if c.labeledAction(ctx, iconPlus, "新建链路", true).Clicked() {
			c.openEditor(editorChain, "")
		}
	case pageRoutes:
		if c.labeledAction(ctx, iconPlus, "添加规则", true).Clicked() {
			c.openEditor(editorRouteRule, "")
		}
	case pageDNS:
		if c.labeledAction(ctx, iconRefresh, "刷新缓存", false).Clicked() {
			c.flushDNS()
		}
		if c.labeledAction(ctx, iconPlay, "测试解析", true).Clicked() {
			c.testDNS()
		}
	case pageRuleSets:
		if c.labeledAction(ctx, iconRefresh, "刷新全部", false).Clicked() {
			c.refreshAllRuleSets()
		}
		if c.labeledAction(ctx, iconPlus, "添加规则集", true).Clicked() {
			c.openEditor(editorRuleSet, "")
		}
	case pageLogs:
		filter := c.logFilter
		if ui.Segmented(ctx, &filter, "全部", "应用", "内核").Label("日志来源").Changed() {
			c.setLogFilter(filter)
		}
	case pageConnections:
		if c.labeledAction(ctx, iconRefresh, "刷新", false).Clicked() {
			c.refresh(true)
		}
		enabled := c.snapshot.Kernel.State == kernel.StateRunning && len(c.snapshot.Connections) > 0
		if c.dangerAction(ctx, "关闭全部").Disabled(!enabled).Clicked() {
			c.closeAllConnections()
		}
	}
}

func (c *Controller) noticeBanner(ctx *ui.Context) {
	p := c.colors()
	tone, fill, border := p.Success, p.SuccessSoft, p.Success.Alpha(0.2)
	if c.noticeErr {
		tone, fill, border = p.Danger, p.DangerSoft, p.Danger.Alpha(0.2)
	}
	ui.Row(ctx).FillWidth().Padding(9, 24).Background(fill).BorderWidth(0, 0, 1, 0).BorderColor(border).
		AlignItems(ui.Center).Gap(10).Children(func() {
		if c.busy {
			ui.Spinner(ctx).Size(14, 14).TextColor(tone)
		} else if c.noticeErr {
			ui.Icon(ctx, iconAlert).FontSize(14).TextColor(tone)
		} else {
			ui.Icon(ctx, iconCheck).FontSize(14).TextColor(tone)
		}
		ui.Text(ctx, c.notice).FontSize(11.5).TextColor(tone).Grow(1).SingleLine().Tooltip(c.notice)
		closeButton := c.iconAction(ctx, iconClose, "关闭提示", false)
		if closeButton.Clicked() {
			c.clearNotice()
		}
	})
}

func (c *Controller) bootView(ctx *ui.Context) {
	p := c.colors()
	ui.Column(ctx).Fill().Center().Gap(14).Children(func() {
		ui.Image(ctx, logoBitmap).Size(52, 52)
		ui.Spinner(ctx).Size(22, 22).TextColor(p.Accent)
		ui.Text(ctx, "正在连接控制面").FontSize(14).FontWeight(650)
		ui.Text(ctx, "正在打开本地数据存储并恢复运行状态…").FontSize(12).TextColor(p.TextMuted)
	})
}

func (c *Controller) unavailableView(ctx *ui.Context) {
	p := c.colors()
	ui.Column(ctx).Fill().Center().Padding(40).Gap(12).Children(func() {
		ui.Image(ctx, logoBitmap).Size(52, 52)
		ui.Text(ctx, "控制面不可用").FontSize(18).FontWeight(700)
		ui.Text(ctx, c.notice).FontSize(12).TextColor(p.Danger).Wrap().TextAlign(ui.Center)
		ui.Button(ctx, "重新初始化").OnClick(c.Boot)
	})
}

func (c *Controller) pageView(ctx *ui.Context) {
	switch c.page {
	case pageOverview:
		c.overviewPage(ctx)
	case pageInbounds:
		c.inboundsPage(ctx)
	case pageSubscriptions:
		c.subscriptionsPage(ctx)
	case pageNodes:
		c.nodesPage(ctx)
	case pageEgress:
		c.egressPage(ctx)
	case pageRoutes:
		c.routesPage(ctx)
	case pageDNS:
		c.dnsPage(ctx)
	case pageRuleSets:
		c.ruleSetsPage(ctx)
	case pageLogs:
		c.logsPage(ctx)
	case pageConnections:
		c.connectionsPage(ctx)
	case pageSettings:
		c.settingsPage(ctx)
	case pageAbout:
		c.aboutPage(ctx)
	default:
		c.overviewPage(ctx)
	}
}

func (c *Controller) pageScroll(ctx *ui.Context, fn func()) {
	ui.Scroll(ctx).Key("page-"+c.page).Grow(1).Padding(22, 24, 28).Gap(16).Children(fn)
}

func (c *Controller) overviewPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.overviewHero(ctx)
		ui.Row(ctx).FillWidth().Gap(10).AlignItems(ui.Stretch).Children(func() {
			running := c.snapshot.Kernel.State == kernel.StateRunning
			upload, download, rate, hint := "—", "—", "—", "内核未启动，未采集数据"
			if running {
				upload = humanBytes(c.snapshot.Traffic.UploadTotal)
				download = humanBytes(c.snapshot.Traffic.DownloadTotal)
				rate = humanRate(c.snapshot.Traffic.UploadRate + c.snapshot.Traffic.DownloadRate)
				hint = "采样于 " + shortTime(c.snapshot.Traffic.At)
			}
			activeConnections, activeHint := "—", "内核未启动，未采集数据"
			if running {
				activeConnections = fmt.Sprintf("%d", len(c.snapshot.Connections))
				activeHint = "当前内核会话"
			}
			c.statTile(ctx, "traffic-up", "累计上传", upload, hint, p.Indigo)
			c.statTile(ctx, "traffic-down", "累计下载", download, hint, p.Sky)
			c.statTile(ctx, "traffic-rate", "实时吞吐", rate, hint, p.Violet)
			c.statTile(ctx, "active-connections", "活动连接", activeConnections, activeHint, p.Success)
		})
		ui.Row(ctx).FillWidth().Gap(14).AlignItems(ui.Stretch).Children(func() {
			ui.Column(ctx).Grow(3).Basis(0).MinWidth(0).Children(func() { c.overviewComponents(ctx) })
			ui.Column(ctx).Grow(2).Basis(0).MinWidth(0).Children(func() { c.overviewActivity(ctx) })
		})
		ui.Row(ctx).FillWidth().Gap(14).AlignItems(ui.Stretch).Children(func() {
			ui.Column(ctx).Grow(3).Basis(0).MinWidth(0).Children(func() { c.overviewConnections(ctx) })
			ui.Column(ctx).Grow(2).Basis(0).MinWidth(0).Children(func() { c.overviewInventory(ctx) })
		})
	})
}

func (c *Controller) overviewHero(ctx *ui.Context) {
	p := c.colors()
	running := c.snapshot.Kernel.State == kernel.StateRunning
	title, subtitle := "内核尚未启动", "启动后即可观察实时流量、连接和规则命中。"
	if running {
		title = "控制面运行正常"
		subtitle = "本地代理内核正在运行，所有指标均来自实时采集。"
	}
	if c.snapshot.Kernel.State == kernel.StateFailed {
		title = "内核需要处理"
		subtitle = "最近一次启动失败，请在下方查看状态并重试。"
	}
	hero := ui.Row(ctx).FillWidth().Padding(22).Radius(18).Border(1, p.Border).AlignItems(ui.Center).Gap(20)
	c.heroGradient(hero)
	hero.Children(func() {
		ui.Column(ctx).Grow(1).Gap(7).Children(func() {
			ui.Row(ctx).AlignItems(ui.Center).Gap(9).Children(func() {
				ui.Box(ctx).Size(9, 9).Radius(9).Background(func() ui.Color {
					if running {
						return p.Success
					}
					return p.Warning
				}())
				ui.Text(ctx, title).FontSize(20).FontWeight(740).SingleLine()
			})
			ui.Text(ctx, subtitle).FontSize(12.5).TextColor(p.TextMuted).SingleLine()
			if c.snapshot.Kernel.LastError != nil {
				ui.Text(ctx, oneLine(c.snapshot.Kernel.LastError.Error())).FontSize(11.5).TextColor(p.Danger).
					SingleLine().Tooltip(c.snapshot.Kernel.LastError.Error())
			}
		})
		ui.Column(ctx).Gap(8).AlignItems(ui.End).Children(func() {
			ui.Row(ctx).Gap(8).Children(func() {
				if running {
					if c.labeledAction(ctx, iconStop, "停止内核", false).Clicked() {
						c.stopKernel()
					}
					if c.labeledAction(ctx, iconRefresh, "重启内核", false).Clicked() {
						c.restartKernel()
					}
				} else {
					if c.labeledAction(ctx, iconPlay, "启动内核", true).Clicked() {
						c.startKernel()
					}
				}
			})
			ui.Text(ctx, kernelRuntimeLabel(c.snapshot)).FontSize(10.5).TextColor(p.TextFaint).SingleLine()
		})
	})
}

func (c *Controller) overviewComponents(ctx *ui.Context) {
	p := c.colors()
	c.card(ctx, "overview-components", func() {
		c.sectionHeader(ctx, "控制面组件", "订阅、策略与数据面状态", nil)
		rows := []struct {
			label, value, hint string
			ok                 bool
		}{
			{"订阅调度", boolLabel(c.snapshot.Scheduler.Running, "运行中", "已停止"), "下次同步 " + shortTime(c.snapshot.Scheduler.NextRunAt), c.snapshot.Scheduler.Running},
			{"DNS 策略", dnsStrategyLabel(c.snapshot.DNS.Strategy), fmt.Sprintf("缓存%s · %d 个槽位启用", boolLabel(c.snapshot.DNS.Cache, "开启", "关闭"), dnsEnabledCount(c.snapshot.DNS)), dnsEnabledCount(c.snapshot.DNS) > 0},
			{"路由策略", routeModeLabel(string(c.snapshot.Routes.Mode)), fmt.Sprintf("%d 条规则 · 默认出口 %s", len(c.snapshot.Routes.Rules), labelForEgress(c.snapshot.Routes.DefaultEgress, c.snapshot)), len(c.snapshot.Routes.Rules) > 0 || c.snapshot.Routes.Mode != control.RouteModeRules},
			{"出口资源", fmt.Sprintf("%d 节点 / %d 策略组", len(c.snapshot.Catalog.Nodes), len(c.snapshot.Catalog.Groups)), fmt.Sprintf("%d 条链路", len(c.snapshot.Catalog.Chains)), len(c.snapshot.Catalog.Nodes) > 0},
		}
		for _, row := range rows {
			ui.Row(ctx).FillWidth().Padding(10, 0).AlignItems(ui.Center).Gap(12).Children(func() {
				ui.Box(ctx).Size(7, 7).Radius(7).Background(func() ui.Color {
					if row.ok {
						return p.Success
					}
					return p.TextFaint
				}())
				ui.Column(ctx).Grow(1).Gap(2).Children(func() {
					ui.Text(ctx, row.label).FontSize(12).FontWeight(600).SingleLine()
					ui.Text(ctx, row.hint).FontSize(10.5).TextColor(p.TextFaint).SingleLine()
				})
				ui.Text(ctx, row.value).FontSize(12).FontWeight(650).TextColor(p.TextMuted).SingleLine()
			})
		}
	})
}

func (c *Controller) overviewActivity(ctx *ui.Context) {
	p := c.colors()
	c.card(ctx, "overview-activity", func() {
		c.sectionHeader(ctx, "最近活动", "来自控制面事件日志", func() {
			if c.labeledAction(ctx, nil, "查看日志", false).Clicked() {
				c.show(pageLogs)
			}
		})
		logs := c.snapshot.mergedLogs(false, false)
		if len(logs) == 0 {
			c.emptyState(ctx, "暂无日志", "控制面操作和内核输出会实时出现在这里。", nil)
			return
		}
		start := max(0, len(logs)-6)
		for _, entry := range logs[start:] {
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(ctx, entry.Time.Local().Format("15:04:05")).FontSize(10).Font("monospace").TextColor(p.TextFaint).Width(58)
				levelColor := logLevelColor(p, entry.Level)
				ui.Box(ctx).Size(6, 6).Radius(6).Background(levelColor)
				ui.Text(ctx, oneLine(entry.Message)).FontSize(11).TextColor(p.TextMuted).Grow(1).SingleLine().Tooltip(entry.Message)
			})
		}
	})
}

func (c *Controller) overviewConnections(ctx *ui.Context) {
	p := c.colors()
	c.card(ctx, "overview-connections", func() {
		c.sectionHeader(ctx, "活动连接", "最近建立的真实连接", func() {
			if c.labeledAction(ctx, nil, "全部连接", false).Clicked() {
				c.show(pageConnections)
			}
		})
		if c.snapshot.Kernel.State != kernel.StateRunning {
			ui.Row(ctx).FillWidth().Padding(12, 0).AlignItems(ui.Center).Children(func() {
				ui.Text(ctx, "—").FontSize(16).TextColor(p.TextFaint).Width(34)
				ui.Text(ctx, "内核未启动，未采集数据").FontSize(11.5).TextColor(p.TextMuted).SingleLine()
			})
			return
		}
		conns := append([]kernel.Connection(nil), c.snapshot.Connections...)
		sort.Slice(conns, func(i, j int) bool { return conns[i].StartedAt.After(conns[j].StartedAt) })
		if len(conns) == 0 {
			c.emptyState(ctx, "当前没有活动连接", "通过代理的流量会即时显示在这里。", nil)
			return
		}
		for _, conn := range conns[:min(6, len(conns))] {
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Text(ctx, conn.Network).FontSize(9.5).FontWeight(650).TextColor(p.Sky).
					Width(40).SingleLine()
				ui.Column(ctx).Grow(1).Gap(1).Children(func() {
					ui.Text(ctx, conn.Target).FontSize(11.5).Font("monospace").SingleLine().Tooltip(conn.Target)
					ui.Text(ctx, conn.Process).FontSize(10).TextColor(p.TextFaint).SingleLine().Tooltip(conn.Process)
				})
				ui.Text(ctx, humanBytes(conn.Upload+conn.Download)).FontSize(10.5).TextColor(p.TextMuted).SingleLine()
			})
		}
	})
}

func (c *Controller) overviewInventory(ctx *ui.Context) {
	p := c.colors()
	c.card(ctx, "overview-inventory", func() {
		c.sectionHeader(ctx, "资源概览", "当前控制面中已保存的对象", nil)
		items := []struct {
			label string
			count int
			page  string
		}{
			{"订阅", len(c.snapshot.Subscriptions), pageSubscriptions},
			{"节点", len(c.snapshot.Catalog.Nodes), pageNodes},
			{"策略组", len(c.snapshot.Catalog.Groups), pageEgress},
			{"链路", len(c.snapshot.Catalog.Chains), pageEgress},
			{"路由规则", len(c.snapshot.Routes.Rules), pageRoutes},
			{"规则集", len(c.snapshot.RuleSets), pageRuleSets},
		}
		for _, item := range items {
			row := ui.ButtonBase(ctx).Label(item.label).FillWidth().Height(36).Padding(0, 10).Radius(8).
				AlignItems(ui.Center).Gap(10)
			row.Children(func() {
				ui.Text(ctx, item.label).FontSize(11.5).TextColor(p.TextMuted).Grow(1).SingleLine()
				ui.Text(ctx, strconv.Itoa(item.count)).FontSize(13).FontWeight(680).SingleLine()
				ui.Icon(ctx, iconArrowRight).FontSize(13).TextColor(p.TextFaint)
			})
			if row.Clicked() {
				c.show(item.page)
			}
		}
	})
}

func (c *Controller) subscriptionsPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "subscriptions-table", func() {
			if len(c.snapshot.Subscriptions) == 0 {
				c.emptyState(ctx, "还没有订阅", "添加远程订阅地址，即可集中维护节点。", func() {
					if c.labeledAction(ctx, iconPlus, "添加第一个订阅", true).Clicked() {
						c.openEditor(editorSubscription, "")
					}
				})
				return
			}
			counts := c.snapshot.nodeCountBySubscription()
			subs := append([]control.Subscription(nil), c.snapshot.Subscriptions...)
			cols := []ui.TableColumn{
				{Title: "名称", Width: 190},
				{Title: "订阅地址", MinWidth: 260},
				{Title: "节点", Width: 70, Align: ui.End},
				{Title: "同步间隔", Width: 92, Align: ui.End},
				{Title: "上次同步", Width: 112},
				{Title: "状态", Width: 80},
				{Title: "操作", Width: 140, Fixed: true},
			}
			state := c.tableState("subscriptions")
			ui.Table(ctx, state, cols, len(subs), func(row, col int) {
				sub := subs[row]
				switch col {
				case 0:
					ui.Text(ctx, sub.Name).FontSize(12.5).FontWeight(600).SingleLine().Tooltip(sub.Name)
				case 1:
					tableMonoCell(ctx, sub.URL, p.TextMuted)
				case 2:
					ui.Text(ctx, strconv.Itoa(counts[sub.ID])).FontSize(12).TextAlign(ui.End)
				case 3:
					ui.Text(ctx, humanInterval(sub.IntervalSeconds)).FontSize(11.5).TextColor(p.TextMuted).TextAlign(ui.End)
				case 4:
					ui.Text(ctx, relativeTime(sub.LastRefreshedAt)).FontSize(11.5).TextColor(p.TextMuted).SingleLine()
				case 5:
					c.smallPill(ctx, boolLabel(sub.Enabled, "启用", "停用"), func() ui.Color {
						if sub.Enabled {
							return p.Success
						}
						return p.TextMuted
					}(), func() ui.Color {
						if sub.Enabled {
							return p.SuccessSoft
						}
						return p.SurfaceAlt
					}())
				case 6:
					ui.Row(ctx).Gap(4).Children(func() {
						if c.iconAction(ctx, iconEdit, "编辑订阅 "+sub.Name, false).Clicked() {
							c.openEditor(editorSubscription, sub.ID)
						}
						if c.iconAction(ctx, iconRefresh, "刷新订阅 "+sub.Name, false).Clicked() {
							c.refreshSubscription(sub.ID)
						}
						if c.iconAction(ctx, iconTrash, "删除订阅 "+sub.Name, true).Clicked() {
							c.askDelete("subscription", sub.ID, sub.Name)
						}
					})
				}
			}).Grow(1).Height(c.tableHeightForRows(ctx, len(subs)))
		})
	})
}

func (c *Controller) nodesPage(ctx *ui.Context) {
	p := c.colors()
	c.pageScroll(ctx, func() {
		c.card(ctx, "nodes-controls", func() {
			ui.Row(ctx).FillWidth().AlignItems(ui.End).Gap(12).Children(func() {
				kindLabels, kindValues := nodeTestOptions()
				current := nodeTestLabel(c.nodeTestKind)
				ui.Column(ctx).Width(170).Gap(6).Children(func() {
					ui.Text(ctx, "测速方式").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
					selectNodeTest := ui.Select(ctx, &current, kindLabels).Label("测速方式")
					if selectNodeTest.Changed() {
						for i, label := range kindLabels {
							if label == current {
								c.nodeTestKind = kindValues[i]
								break
							}
						}
					}
				})
				ui.Column(ctx).Gap(6).Grow(1).Children(func() {
					ui.Text(ctx, "URL 测试地址（仅 URL 模式使用）").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
					ui.TextInput(ctx, &c.nodeTestURL).Placeholder("https://www.gstatic.com/generate_204").Label("URL 测试地址")
				})
				if c.labeledAction(ctx, iconPlay, "测试全部节点", true).Clicked() {
					c.testAllNodes(c.nodeTestKind, c.nodeTestURL)
				}
			})
		})
		c.card(ctx, "nodes-table", func() {
			if len(c.snapshot.Catalog.Nodes) == 0 {
				c.emptyState(ctx, "还没有节点", "添加单个节点，或从分享链接批量导入。", func() {
					if c.labeledAction(ctx, iconPlus, "导入分享链接", true).Clicked() {
						c.openEditor(editorImportLinks, "")
					}
				})
				return
			}
			nodes := append([]control.Node(nil), c.snapshot.Catalog.Nodes...)
			sort.SliceStable(nodes, func(i, j int) bool { return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name) })
			cols := []ui.TableColumn{
				{Title: "名称", Width: 210},
				{Title: "协议", Width: 104},
				{Title: "服务器", MinWidth: 180},
				{Title: "端口", Width: 64, Align: ui.End},
				{Title: "策略组", Width: 160},
				{Title: "连通性", Width: 140},
				{Title: "操作", Width: 106, Fixed: true},
			}
			state := c.tableState("nodes")
			ui.Table(ctx, state, cols, len(nodes), func(row, col int) {
				node := nodes[row]
				switch col {
				case 0:
					ui.Text(ctx, node.Name).FontSize(12.5).FontWeight(600).SingleLine().Tooltip(node.Name)
				case 1:
					c.smallPill(ctx, protocolLabel(node.Protocol), p.Sky, p.SurfaceAlt)
				case 2:
					tableMonoCell(ctx, node.Server, p.TextMuted)
				case 3:
					ui.Text(ctx, strconv.Itoa(int(node.Port))).FontSize(11.5).TextColor(p.TextMuted).TextAlign(ui.End)
				case 4:
					groups := c.snapshot.groupNamesForNode(node.ID)
					text := "未分组"
					if len(groups) > 0 {
						text = strings.Join(groups, "、")
					}
					ui.Text(ctx, text).FontSize(11).TextColor(p.TextMuted).SingleLine().Tooltip(text)
				case 5:
					status := c.nodeTests[node.ID]
					if status == "" {
						status = "未测试"
					}
					tone := p.TextMuted
					if strings.HasPrefix(status, "失败") {
						tone = p.Danger
					} else if status != "未测试" && status != "测速中…" {
						tone = p.Success
					}
					ui.Text(ctx, status).FontSize(11).TextColor(tone).SingleLine().Tooltip(status)
				case 6:
					ui.Row(ctx).Gap(4).Children(func() {
						if c.iconAction(ctx, iconEdit, "编辑节点 "+node.Name, false).Clicked() {
							c.openEditor(editorNode, node.ID)
						}
						if c.iconAction(ctx, iconPlay, "测试节点 "+node.Name, false).Clicked() {
							c.testNodes([]string{node.ID}, c.nodeTestKind, c.nodeTestURL)
						}
						if c.iconAction(ctx, iconTrash, "删除节点 "+node.Name, true).Clicked() {
							c.askDelete("node", node.ID, node.Name)
						}
					})
				}
			}).Grow(1).Height(c.tableHeightForRows(ctx, len(nodes)))
		})
	})
}

func humanInterval(seconds int) string {
	if seconds <= 0 {
		return "手动"
	}
	if seconds%3600 == 0 {
		return fmt.Sprintf("%d 小时", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%d 分钟", seconds/60)
	}
	return fmt.Sprintf("%d 秒", seconds)
}

func boolLabel(value bool, yes, no string) string {
	if value {
		return yes
	}
	return no
}

func dnsEnabledCount(profile control.DNSProfile) int {
	count := 0
	for _, server := range []control.DNSServerConfig{profile.Domestic, profile.Foreign, profile.Default} {
		if server.Enabled {
			count++
		}
	}
	return count
}

func dnsStrategyLabel(strategy string) string {
	if strategy == "" {
		return "自动选择"
	}
	labels, values := dnsStrategyChoices()
	return optionLabel(labels, values, strategy)
}

func dnsStrategyChoices() ([]string, []string) {
	return []string{"自动选择", "偏好 IPv4", "偏好 IPv6"}, []string{"auto", "prefer_ipv4", "prefer_ipv6"}
}

func shortVersion(version string) string {
	if version == "" {
		return "—"
	}
	if len(version) > 16 {
		return version[:16]
	}
	return version
}

func compactVersion(version string) string {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return "—"
	}
	if index := strings.IndexAny(version, "-+"); index >= 0 {
		version = version[:index]
	}
	if version == "" {
		return "—"
	}
	return "v" + version
}

func kernelRuntimeLabel(snapshot snapshot) string {
	kind := strings.TrimSpace(string(snapshot.Kernel.Kind))
	if kind == "" {
		kind = strings.TrimSpace(string(snapshot.Capabilities.Kind))
	}
	if kind == "" {
		kind = string(kernel.EngineSingBox)
	}
	version := strings.TrimSpace(snapshot.Kernel.Version)
	if version == "" {
		version = strings.TrimSpace(snapshot.Capabilities.Version)
	}
	if version == "" || strings.EqualFold(version, "unknown") {
		return kind
	}
	return kind + " · " + shortVersion(version)
}

func (c *Controller) version() string {
	if c.runtime == nil || c.runtime.Version == "" {
		return "0.1.0-dev"
	}
	return c.runtime.Version
}

func nodeTestOptions() ([]string, []control.NodeTestKind) {
	return []string{"URL 测试", "TCP Ping", "ICMP Ping"}, []control.NodeTestKind{
		control.NodeTestKindURL, control.NodeTestKindTCP, control.NodeTestKindICMP,
	}
}

func nodeTestLabel(kind control.NodeTestKind) string {
	labels, values := nodeTestOptions()
	for i, value := range values {
		if value == kind {
			return labels[i]
		}
	}
	return labels[0]
}

func nowClock(clock *clock) string {
	return clock.Now().Local().Format("15:04:05")
}

var _ = time.Second
