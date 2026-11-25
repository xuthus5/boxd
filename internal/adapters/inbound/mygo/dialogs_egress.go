package mygo

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
)

type egressPick struct {
	Kind   control.EgressKind
	ID     string
	Name   string
	Detail string
	Search string
}

func (c *Controller) groupEditorFields(ctx *ui.Context, e *editorState) {
	d := &e.group
	if d.Kind == "" {
		d.Kind = control.GroupKindSelector
	}
	d.ensureSelection()
	p := c.colors()

	ui.Row(ctx).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
		ui.Column(ctx).Grow(1).Basis(0).Children(func() {
			ui.Field(ctx, "名称", func() { ui.TextInput(ctx, &d.Name).AutoFocus().Label("策略组名称") })
		})
		ui.Column(ctx).Width(156).Children(func() {
			ui.Field(ctx, "类型", func() {
				labels := []string{"手动选择", "自动测速"}
				values := []string{string(control.GroupKindSelector), string(control.GroupKindURLTest)}
				current := groupKindLabel(string(d.Kind))
				selector := ui.Select(ctx, &current, labels).Label("策略组类型")
				if selector.Changed() {
					d.Kind = control.GroupKind(optionValueByLabel(labels, values, current))
					d.ensureSelection()
				}
			})
		})
		if d.Kind == control.GroupKindURLTest {
			ui.Column(ctx).Width(126).Children(func() {
				ui.Field(ctx, "测速间隔（秒）", func() { ui.TextInput(ctx, &d.Interval).Label("测速间隔（秒）") })
			})
		}
	})

	ui.Row(ctx).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
		c.smallPill(ctx, fmt.Sprintf("%d 个节点", len(d.Members)), p.Accent, p.AccentSoft)
		summary := groupDraftSummary(*d, c.snapshot)
		ui.Text(ctx, summary).FontSize(11.5).TextColor(p.TextMuted).SingleLine().Tooltip(summary)
		ui.Box(ctx).Grow(1)
		ui.Text(ctx, "拖动成员可调整顺序").FontSize(10.5).TextColor(p.TextFaint).SingleLine()
	})

	ui.Row(ctx).FillWidth().Gap(14).AlignItems(ui.Start).Children(func() {
		c.groupAvailableNodes(ctx, d)
		c.groupSelectedMembers(ctx, d)
	})

	if d.Kind == control.GroupKindSelector {
		ui.Column(ctx).FillWidth().Gap(6).Children(func() {
			ui.Text(ctx, "当前出口").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
			if len(d.Members) == 0 {
				ui.Text(ctx, "添加成员后可选择当前出口").FontSize(11).TextColor(p.TextFaint).SingleLine()
			} else {
				names, values := groupMemberChoices(c.snapshot, d.Members)
				selected := optionLabel(names, values, d.Selected)
				selector := ui.Select(ctx, &selected, names).Label("当前出口")
				if selector.Changed() {
					d.Selected = optionValueByLabel(names, values, selected)
				}
			}
		})
	} else {
		ui.Field(ctx, "测速 URL", func() {
			ui.TextInput(ctx, &d.URL).Placeholder("https://www.gstatic.com/generate_204").Label("测速 URL")
		})
	}

	c.egressIssues(ctx, d.issues(c.snapshot), nil)
}

func (c *Controller) groupAvailableNodes(ctx *ui.Context, d *groupDraft) {
	p := c.colors()
	nodes := c.filteredGroupNodes(d)
	ui.Column(ctx).Grow(1).Basis(0).MinWidth(330).Gap(7).Children(func() {
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(ctx, "可用节点").FontSize(11).FontWeight(650).TextColor(p.TextMuted)
			ui.Box(ctx).Grow(1)
			ui.Text(ctx, fmt.Sprintf("%d 个可添加", len(nodes))).FontSize(10.5).TextColor(p.TextFaint).SingleLine()
		})
		ui.SearchField(ctx, &d.Query).Placeholder("搜索名称、协议、服务器或订阅").Label("搜索可用节点")
		if len(nodes) == 0 {
			title, hint := "没有可添加的节点", "调整搜索条件，或先到节点页面添加节点。"
			if strings.TrimSpace(d.Query) == "" && len(c.snapshot.Catalog.Nodes) > 0 {
				title, hint = "所有节点都已加入成员", "如需调整，请先从右侧移除节点。"
			}
			ui.Column(ctx).FillWidth().Height(196).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10).
				Padding(16).Gap(5).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Text(ctx, title).FontSize(12).FontWeight(600).TextColor(p.TextMuted)
				ui.Text(ctx, hint).FontSize(10.5).TextColor(p.TextFaint).Wrap().TextAlign(ui.Center)
			})
		} else {
			state := c.egressListState("egress-group-available")
			state.RowHeight = 46
			state.Key = func(row int) any { return nodes[row].ID }
			state.Label = func(row int) string { return nodes[row].Name }
			ui.List(ctx, state, len(nodes), func(row int) {
				pick := nodes[row]
				ui.Row(ctx).FillWidth().Height(44).Padding(6, 8).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Column(ctx).Grow(1).Basis(0).Gap(2).Children(func() {
						ui.Text(ctx, pick.Name).FontSize(12).FontWeight(600).SingleLine().Tooltip(pick.Name)
						ui.Text(ctx, pick.Detail).FontSize(10.5).TextColor(p.TextFaint).SingleLine().Tooltip(pick.Detail)
					})
					if c.iconAction(ctx, iconPlus, "添加节点 "+pick.Name, false).Clicked() {
						d.addMember(pick.ID)
					}
				})
			}).FillWidth().Height(196).Padding(3).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10)
		}
		button := c.labeledAction(ctx, iconPlus, "添加全部筛选节点", false)
		if len(nodes) == 0 {
			button.Disabled(true)
		} else if button.Clicked() {
			ids := make([]string, 0, len(nodes))
			for _, node := range nodes {
				ids = append(ids, node.ID)
			}
			d.addMembers(ids)
		}
	})
}

func (c *Controller) groupSelectedMembers(ctx *ui.Context, d *groupDraft) {
	p := c.colors()
	ui.Column(ctx).Grow(1).Basis(0).MinWidth(330).Gap(7).Children(func() {
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(ctx, "已选成员").FontSize(11).FontWeight(650).TextColor(p.TextMuted)
			ui.Box(ctx).Grow(1)
			ui.Text(ctx, fmt.Sprintf("%d / %d", len(d.Members), len(c.snapshot.Catalog.Nodes))).FontSize(10.5).TextColor(p.TextFaint).SingleLine()
		})
		if len(d.Members) == 0 {
			ui.Column(ctx).FillWidth().Height(225).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10).
				Padding(16).Gap(5).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Text(ctx, "还没有成员").FontSize(12).FontWeight(600).TextColor(p.TextMuted)
				ui.Text(ctx, "从左侧搜索并添加节点。成员顺序会原样写入内核。").FontSize(10.5).TextColor(p.TextFaint).Wrap().TextAlign(ui.Center)
			})
		} else {
			state := c.egressListState("egress-group-members")
			state.RowHeight = 54
			state.Label = func(row int) string { return nodeName(c.snapshot, d.Members[row]) }
			state.Reorder = func(rows []int, to int) {
				d.reorderMembers(rows, to)
				d.ensureSelection()
			}
			ui.List(ctx, state, len(d.Members), func(row int) {
				nodeID := d.Members[row]
				node, exists := c.snapshot.NodesByID[nodeID]
				name := nodeName(c.snapshot, nodeID)
				detail := nodePickDetail(c.snapshot, node)
				if !exists {
					detail = "节点已不存在"
				}
				ui.Row(ctx).FillWidth().Height(52).Padding(5, 8).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Text(ctx, strconv.Itoa(row+1)).Width(18).FontSize(10.5).FontWeight(650).TextColor(p.TextFaint).TextAlign(ui.End)
					ui.Icon(ctx, iconGrip).FontSize(14).TextColor(p.TextFaint)
					ui.Column(ctx).Grow(1).Basis(0).Gap(2).Children(func() {
						ui.Row(ctx).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Text(ctx, name).FontSize(12).FontWeight(600).SingleLine().Tooltip(name)
							if d.Kind == control.GroupKindSelector && nodeID == d.Selected {
								c.smallPill(ctx, "当前", p.Accent, p.AccentSoft)
							}
						})
						ui.Text(ctx, detail).FontSize(10.5).TextColor(p.TextFaint).SingleLine().Tooltip(detail)
					})
					if c.iconAction(ctx, iconChevronUp, "上移节点 "+name, false).Disabled(row == 0).Clicked() {
						d.moveMember(row, -1)
					}
					if c.iconAction(ctx, iconChevronDown, "下移节点 "+name, false).Disabled(row == len(d.Members)-1).Clicked() {
						d.moveMember(row, 1)
					}
					if c.iconAction(ctx, iconTrash, "移除节点 "+name, true).Clicked() {
						d.removeMember(nodeID)
					}
				})
			}).FillWidth().Height(196).Padding(3).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10)
		}
		clear := c.labeledAction(ctx, iconTrash, "清空成员", false)
		if len(d.Members) == 0 {
			clear.Disabled(true)
		} else if clear.Clicked() {
			d.Members = nil
			d.Selected = ""
		}
	})
}

func (c *Controller) chainEditorFields(ctx *ui.Context, e *editorState) {
	d := &e.chain
	p := c.colors()

	ui.Row(ctx).FillWidth().Gap(12).AlignItems(ui.End).Children(func() {
		ui.Column(ctx).Grow(1).Basis(0).Children(func() {
			ui.Field(ctx, "名称", func() { ui.TextInput(ctx, &d.Name).AutoFocus().Label("链路名称") })
		})
		c.smallPill(ctx, fmt.Sprintf("%d / %d 跳点", len(d.Hops), maxChainHops), p.Accent, p.AccentSoft)
	})
	summary := chainPathSummary(*d, c.snapshot)
	ui.Text(ctx, summary).FontSize(12).FontWeight(600).TextColor(p.Text).SingleLine().Tooltip(summary)

	ui.Row(ctx).FillWidth().Gap(14).AlignItems(ui.Start).Children(func() {
		c.chainResourcePicker(ctx, d)
		c.chainPathEditor(ctx, d)
	})
	c.egressIssues(ctx, d.issues(c.snapshot), c.chainUDPWarnings(*d))
}

func (c *Controller) chainResourcePicker(ctx *ui.Context, d *chainDraft) {
	p := c.colors()
	nodes := c.filteredChainNodes(d)
	groups := c.filteredChainGroups(d)
	full := len(d.Hops) >= maxChainHops
	ui.Column(ctx).Width(330).Gap(7).Children(func() {
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(ctx, "添加跳点").FontSize(11).FontWeight(650).TextColor(p.TextMuted)
			ui.Box(ctx).Grow(1)
			if full {
				ui.Text(ctx, "已达上限").FontSize(10.5).FontWeight(600).TextColor(p.Warning)
			}
		})
		ui.SearchField(ctx, &d.Query).Placeholder("搜索节点或策略组").Label("搜索跳点资源")
		if !full && len(nodes) > 0 {
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(ctx, "节点").FontSize(10.5).FontWeight(650).TextColor(p.TextFaint)
				ui.Box(ctx).Grow(1)
			})
			state := c.egressListState("egress-chain-nodes")
			state.RowHeight = 44
			state.Key = func(row int) any { return nodes[row].ID }
			state.Label = func(row int) string { return nodes[row].Name }
			ui.List(ctx, state, len(nodes), func(row int) {
				c.chainPickRow(ctx, nodes[row], d)
			}).FillWidth().Height(112).Padding(3).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10)
		}
		if !full && len(groups) > 0 {
			ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(ctx, "策略组").FontSize(10.5).FontWeight(650).TextColor(p.TextFaint)
				ui.Box(ctx).Grow(1)
			})
			state := c.egressListState("egress-chain-groups")
			state.RowHeight = 44
			state.Key = func(row int) any { return groups[row].ID }
			state.Label = func(row int) string { return groups[row].Name }
			ui.List(ctx, state, len(groups), func(row int) {
				c.chainPickRow(ctx, groups[row], d)
			}).FillWidth().Height(100).Padding(3).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10)
		}
		if full {
			c.emptyHint(ctx, "链路最多 4 个跳点", "移除已有跳点后才能继续添加。", 210)
		} else if len(nodes) == 0 && len(groups) == 0 {
			title, hint := "没有匹配的资源", "调整搜索条件，或先创建节点和策略组。"
			if strings.TrimSpace(d.Query) == "" && len(c.snapshot.Catalog.Nodes)+len(c.snapshot.Catalog.Groups) > 0 {
				title, hint = "所有资源都已加入链路", "移除已有跳点后可重新添加。"
			}
			c.emptyHint(ctx, title, hint, 210)
		}
	})
}

func (c *Controller) chainPickRow(ctx *ui.Context, pick egressPick, d *chainDraft) {
	p := c.colors()
	ui.Row(ctx).FillWidth().Height(42).Padding(5, 8).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(ctx).Grow(1).Basis(0).Gap(1).Children(func() {
			ui.Text(ctx, pick.Name).FontSize(12).FontWeight(600).SingleLine().Tooltip(pick.Name)
			ui.Text(ctx, pick.Detail).FontSize(10).TextColor(p.TextFaint).SingleLine().Tooltip(pick.Detail)
		})
		if c.iconAction(ctx, iconPlus, "添加跳点 "+pick.Name, false).Clicked() {
			d.addHop(pick.Kind, pick.ID)
		}
	})
}

func (c *Controller) chainPathEditor(ctx *ui.Context, d *chainDraft) {
	p := c.colors()
	ui.Column(ctx).Grow(1).Basis(0).MinWidth(350).Gap(7).Children(func() {
		ui.Row(ctx).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(ctx, "链路路径").FontSize(11).FontWeight(650).TextColor(p.TextMuted)
			ui.Box(ctx).Grow(1)
			ui.Text(ctx, "本地 → 目标网络").FontSize(10.5).TextColor(p.TextFaint).SingleLine()
		})
		ui.Row(ctx).FillWidth().Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Box(ctx).Size(7, 7).Radius(7).Background(p.Sky)
			ui.Text(ctx, "本地").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
		})
		if len(d.Hops) == 0 {
			c.emptyHint(ctx, "链路还没有跳点", "从左侧添加节点或策略组，顺序即数据流经过的顺序。", 220)
		} else {
			state := c.egressListState("egress-chain-hops")
			state.RowHeight = 57
			state.Label = func(row int) string {
				return labelForEgress(control.EgressRef{Kind: d.Hops[row].Kind, ID: d.Hops[row].ID}, c.snapshot)
			}
			state.Reorder = func(rows []int, to int) { d.reorderHops(rows, to) }
			ui.List(ctx, state, len(d.Hops), func(row int) {
				hop := d.Hops[row]
				name := labelForEgress(control.EgressRef{Kind: hop.Kind, ID: hop.ID}, c.snapshot)
				detail := c.hopDetail(hop)
				ui.Row(ctx).FillWidth().Height(55).Padding(6, 8).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(ctx, iconGrip).FontSize(14).TextColor(p.TextFaint)
					ui.Column(ctx).Width(22).Gap(1).AlignItems(ui.Center).Children(func() {
						ui.Text(ctx, strconv.Itoa(row+1)).FontSize(11).FontWeight(700).TextColor(p.Accent)
						ui.Box(ctx).Width(1).Height(12).Background(p.Border)
					})
					ui.Column(ctx).Grow(1).Basis(0).Gap(2).Children(func() {
						ui.Row(ctx).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Text(ctx, name).FontSize(12).FontWeight(600).SingleLine().Tooltip(name)
							c.smallPill(ctx, hopKindLabel(hop.Kind), p.Sky, p.SurfaceAlt)
						})
						ui.Text(ctx, detail).FontSize(10.5).TextColor(p.TextFaint).SingleLine().Tooltip(detail)
					})
					if c.iconAction(ctx, iconChevronUp, "上移跳点 "+name, false).Disabled(row == 0).Clicked() {
						d.moveHop(row, -1)
					}
					if c.iconAction(ctx, iconChevronDown, "下移跳点 "+name, false).Disabled(row == len(d.Hops)-1).Clicked() {
						d.moveHop(row, 1)
					}
					if c.iconAction(ctx, iconTrash, "移除跳点 "+name, true).Clicked() {
						d.removeHop(row)
					}
				})
			}).FillWidth().Height(220).Padding(3).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10)
		}
		ui.Row(ctx).FillWidth().Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Box(ctx).Size(7, 7).Radius(7).Background(p.Success)
			ui.Text(ctx, "目标网络").FontSize(11).FontWeight(600).TextColor(p.TextMuted)
		})
	})
}

func (c *Controller) emptyHint(ctx *ui.Context, title, hint string, height float32) {
	p := c.colors()
	ui.Column(ctx).FillWidth().Height(height).Background(p.SurfaceAlt).Border(1, p.BorderSoft).Radius(10).
		Padding(16).Gap(5).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(ctx, title).FontSize(12).FontWeight(600).TextColor(p.TextMuted)
		ui.Text(ctx, hint).FontSize(10.5).TextColor(p.TextFaint).Wrap().TextAlign(ui.Center)
	})
}

func (c *Controller) egressIssues(ctx *ui.Context, issues, warnings []string) {
	if len(issues) == 0 && len(warnings) == 0 {
		return
	}
	p := c.colors()
	ui.Column(ctx).FillWidth().Gap(4).Padding(9, 11).Radius(10).Background(p.WarningSoft).
		Border(1, p.Warning.Alpha(0.25)).Children(func() {
		for _, issue := range issues {
			ui.Text(ctx, "• "+issue).FontSize(11).TextColor(p.Danger).Wrap().LineHeight(1.45)
		}
		for _, warning := range warnings {
			ui.Text(ctx, "• "+warning).FontSize(11).TextColor(p.Warning).Wrap().LineHeight(1.45)
		}
	})
}

func (c *Controller) editorErrorBanner(ctx *ui.Context, message string) {
	p := c.colors()
	ui.Row(ctx).FillWidth().Padding(10, 12).Radius(10).Background(p.DangerSoft).
		Border(1, p.Danger.Alpha(0.25)).AlignItems(ui.Start).Gap(8).Children(func() {
		ui.Icon(ctx, iconClose).FontSize(13).TextColor(p.Danger)
		ui.Text(ctx, message).FontSize(11).TextColor(p.Danger).Grow(1).Wrap().LineHeight(1.45)
	})
}

func (c *Controller) egressListState(key string) *ui.ListState {
	state := c.lists[key]
	if state == nil {
		state = &ui.ListState{}
		c.lists[key] = state
	}
	return state
}

func (c *Controller) filteredGroupNodes(d *groupDraft) []egressPick {
	picks := make([]egressPick, 0, len(c.snapshot.Catalog.Nodes))
	for _, node := range c.snapshot.Catalog.Nodes {
		if containsString(d.Members, node.ID) {
			continue
		}
		pick := nodePick(c.snapshot, node)
		if matchesEgressSearch(d.Query, pick) {
			picks = append(picks, pick)
		}
	}
	sort.SliceStable(picks, func(i, j int) bool { return strings.ToLower(picks[i].Name) < strings.ToLower(picks[j].Name) })
	return picks
}

func (c *Controller) filteredChainNodes(d *chainDraft) []egressPick {
	picks := make([]egressPick, 0, len(c.snapshot.Catalog.Nodes))
	for _, node := range c.snapshot.Catalog.Nodes {
		if chainContainsHop(*d, control.EgressKindNode, node.ID) {
			continue
		}
		pick := nodePick(c.snapshot, node)
		if matchesEgressSearch(d.Query, pick) {
			picks = append(picks, pick)
		}
	}
	sort.SliceStable(picks, func(i, j int) bool { return strings.ToLower(picks[i].Name) < strings.ToLower(picks[j].Name) })
	return picks
}

func (c *Controller) filteredChainGroups(d *chainDraft) []egressPick {
	picks := make([]egressPick, 0, len(c.snapshot.Catalog.Groups))
	for _, group := range c.snapshot.Catalog.Groups {
		if chainContainsHop(*d, control.EgressKindGroup, group.ID) {
			continue
		}
		pick := egressPick{
			Kind: control.EgressKindGroup, ID: group.ID, Name: group.Name,
			Detail: fmt.Sprintf("策略组 · %s · %d 个成员", groupKindLabel(string(group.Kind)), len(group.Members)),
			Search: strings.ToLower(strings.Join([]string{group.Name, group.ID, groupKindLabel(string(group.Kind))}, " ")),
		}
		if matchesEgressSearch(d.Query, pick) {
			picks = append(picks, pick)
		}
	}
	sort.SliceStable(picks, func(i, j int) bool { return strings.ToLower(picks[i].Name) < strings.ToLower(picks[j].Name) })
	return picks
}

func nodePick(snapshot snapshot, node control.Node) egressPick {
	detail := nodePickDetail(snapshot, node)
	return egressPick{
		Kind: control.EgressKindNode, ID: node.ID, Name: node.Name, Detail: detail,
		Search: strings.ToLower(strings.Join([]string{node.Name, node.ID, node.Protocol, node.Server, detail}, " ")),
	}
}

func nodePickDetail(snapshot snapshot, node control.Node) string {
	if node.ID == "" {
		return "节点已不存在"
	}
	parts := []string{strings.ToUpper(node.Protocol), fmt.Sprintf("%s:%d", node.Server, node.Port)}
	if node.SubscriptionID != "" {
		parts = append(parts, subscriptionName(snapshot, node.SubscriptionID))
	}
	return strings.Join(parts, " · ")
}

func subscriptionName(snapshot snapshot, id string) string {
	if subscription, exists := snapshot.SubscriptionsByID[id]; exists && subscription.Name != "" {
		return subscription.Name
	}
	for _, subscription := range snapshot.Subscriptions {
		if subscription.ID == id && subscription.Name != "" {
			return subscription.Name
		}
	}
	return "订阅节点"
}

func matchesEgressSearch(query string, pick egressPick) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	return query == "" || strings.Contains(pick.Search, query)
}

func chainContainsHop(d chainDraft, kind control.EgressKind, id string) bool {
	for _, hop := range d.Hops {
		if hop.Kind == kind && hop.ID == id {
			return true
		}
	}
	return false
}

func chainPathSummary(d chainDraft, snapshot snapshot) string {
	parts := []string{"本地"}
	for _, hop := range d.Hops {
		parts = append(parts, labelForEgress(control.EgressRef{Kind: hop.Kind, ID: hop.ID}, snapshot))
	}
	parts = append(parts, "目标网络")
	return strings.Join(parts, " → ")
}

func groupDraftSummary(d groupDraft, snapshot snapshot) string {
	mode := groupKindLabel(string(d.Kind))
	if d.Kind == control.GroupKindSelector {
		selected := "尚未选择"
		if d.Selected != "" {
			selected = labelForEgress(control.EgressRef{Kind: control.EgressKindNode, ID: d.Selected}, snapshot)
		}
		return fmt.Sprintf("%d 个节点 · %s · 当前出口 %s", len(d.Members), mode, selected)
	}
	interval, _ := strconv.Atoi(strings.TrimSpace(d.Interval))
	if interval <= 0 {
		interval = 300
	}
	return fmt.Sprintf("%d 个节点 · %s · 每 %s测速", len(d.Members), mode, humanInterval(interval))
}

func groupMemberChoices(snapshot snapshot, members []string) ([]string, []string) {
	names := make([]string, 0, len(members))
	values := make([]string, 0, len(members))
	for _, member := range members {
		names = append(names, labelForEgress(control.EgressRef{Kind: control.EgressKindNode, ID: member}, snapshot))
		values = append(values, member)
	}
	return names, values
}

func (c *Controller) hopDetail(hop control.ProxyHop) string {
	switch hop.Kind {
	case control.EgressKindNode:
		if node, exists := c.snapshot.NodesByID[hop.ID]; exists {
			return nodePickDetail(c.snapshot, node)
		}
	case control.EgressKindGroup:
		if group, exists := c.snapshot.GroupsByID[hop.ID]; exists {
			return fmt.Sprintf("策略组 · %s · %d 个成员", groupKindLabel(string(group.Kind)), len(group.Members))
		}
	}
	return "资源已不存在"
}

func hopKindLabel(kind control.EgressKind) string {
	switch kind {
	case control.EgressKindNode:
		return "节点"
	case control.EgressKindGroup:
		return "策略组"
	default:
		return string(kind)
	}
}

func (c *Controller) chainUDPWarnings(d chainDraft) []string {
	udpEnabled := false
	for _, inbound := range c.snapshot.Inbounds {
		if inbound.Enabled && inbound.Kind == control.InboundKindMixed && inbound.UDP {
			udpEnabled = true
			break
		}
	}
	if !udpEnabled {
		return nil
	}
	seen := make(map[string]struct{})
	warnings := make([]string, 0, 2)
	addNode := func(nodeID string) {
		if _, exists := seen[nodeID]; exists {
			return
		}
		node, exists := c.snapshot.NodesByID[nodeID]
		if !exists || node.NodeSupportsUDP() {
			return
		}
		seen[nodeID] = struct{}{}
		warnings = append(warnings, fmt.Sprintf("节点“%s”未启用 UDP，但 Mixed 入口已开启 UDP；UDP 流量将无法通过该跳点。", nodeName(c.snapshot, nodeID)))
	}
	for _, hop := range d.Hops {
		switch hop.Kind {
		case control.EgressKindNode:
			addNode(hop.ID)
		case control.EgressKindGroup:
			if group, exists := c.snapshot.GroupsByID[hop.ID]; exists {
				for _, member := range group.Members {
					addNode(member)
				}
			}
		}
	}
	return warnings
}
