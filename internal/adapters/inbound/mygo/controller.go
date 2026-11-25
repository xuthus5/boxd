package mygo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const (
	pageOverview      = "overview"
	pageSubscriptions = "subscriptions"
	pageNodes         = "nodes"
	pageEgress        = "egress"
	pageRoutes        = "routes"
	pageDNS           = "dns"
	pageRuleSets      = "rulesets"
	pageLogs          = "logs"
	pageConnections   = "connections"
	pageSettings      = "settings"
	pageAbout         = "about"
)

type navItem struct {
	ID    string
	Label string
	Icon  string
	Hint  string
}

var navigation = []navItem{
	{pageOverview, "总览", "overview", "运行状态与关键指标"},
	{pageSubscriptions, "订阅", "subscriptions", "远程节点来源"},
	{pageNodes, "节点", "nodes", "节点与连通性"},
	{pageEgress, "出口编排", "egress", "策略组与链路"},
	{pageRoutes, "路由", "routes", "分流策略与规则"},
	{pageDNS, "DNS", "dns", "解析策略"},
	{pageRuleSets, "规则集", "rulesets", "规则资源"},
	{pageLogs, "日志", "logs", "应用与内核日志"},
	{pageConnections, "连接", "connections", "实时活动连接"},
	{pageSettings, "设置", "settings", "同步与界面偏好"},
	{pageAbout, "关于", "about", "版本与运行环境"},
}

type Controller struct {
	loader  Loader
	window  *mygo.Window
	options ControllerOptions

	runtime  *Runtime
	snapshot snapshot
	clock    *clock

	page       string
	loading    bool
	busy       bool
	refreshing bool
	notice     string
	noticeErr  bool
	narrow     bool
	dark       bool

	editorOpen bool
	editor     editorState
	deleteOpen bool
	deleteKind string
	deleteID   string
	deleteName string

	logFilter    int
	nodeTests    map[string]string
	nodeTestKind control.NodeTestKind
	nodeTestURL  string

	settingsDraft map[string]string
	lists         map[string]*ui.ListState
	dnsResults    []control.DNSProbeResult

	ctx    context.Context
	cancel context.CancelFunc
	close  sync.Once
}

// ControllerOptions contains host-owned presentation preferences. They are
// deliberately separate from control-plane settings because theme choice is a
// client concern shared by no other ingress.
type ControllerOptions struct {
	Dark          bool
	OnThemeChange func(bool) error
}

type editorKind int

const (
	editorNone editorKind = iota
	editorSubscription
	editorNode
	editorImportLinks
	editorGroup
	editorChain
	editorRouteRule
	editorDNS
	editorRuleSet
)

type editorState struct {
	kind editorKind
	id   string

	name     string
	url      string
	interval string
	enabled  bool

	protocol       string
	server         string
	port           string
	options        string
	udp            bool
	subscriptionID string

	members   string
	groupKind string
	selected  string

	hops string

	rule          control.RouteRule
	matchDomains  string
	matchSuffixes string
	matchRuleSets string
	matchPorts    string
	action        string
	egressKind    string
	egressID      string

	dnsSlot   string
	dnsServer control.DNSServerConfig

	format  string
	builtin bool

	links   string
	groupID string
}

func NewController(loader Loader, options ControllerOptions) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		loader:        loader,
		options:       options,
		clock:         newClock(),
		page:          pageOverview,
		nodeTests:     make(map[string]string),
		nodeTestKind:  control.NodeTestKindURL,
		nodeTestURL:   "https://www.gstatic.com/generate_204",
		settingsDraft: make(map[string]string),
		lists:         make(map[string]*ui.ListState),
		ctx:           ctx,
		cancel:        cancel,
		dark:          options.Dark,
	}
}

func (c *Controller) Attach(window *mygo.Window) {
	c.window = window
}

func (c *Controller) Boot() {
	if c.loader == nil {
		c.setLoadingError(fmt.Errorf("control plane loader is unavailable"))
		return
	}
	c.loading = true
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 45*time.Second)
		defer cancel()
		runtime, err := c.loader(ctx)
		c.window.Update(func() {
			if err != nil {
				c.loading = false
				c.notice = "控制面初始化失败"
				c.noticeErr = true
				c.snapshot = snapshot{}
				c.notice = err.Error()
				return
			}
			c.runtime = runtime
			c.loading = false
			c.noticeErr = false
			c.notice = "控制面已就绪"
			c.refresh(true)
			go c.watch()
		})
	}()
}

func (c *Controller) setLoadingError(err error) {
	c.loading = false
	c.notice = err.Error()
	c.noticeErr = true
	if c.window != nil {
		c.window.Invalidate()
	}
}

func (c *Controller) watch() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	ticks := 0
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			ticks++
			c.window.Update(func() {
				c.clock.Tick()
				if ticks%5 == 0 && c.runtime != nil && !c.busy {
					c.refresh(false)
				}
			})
		}
	}
}

func (c *Controller) closeRuntime() {
	c.close.Do(func() {
		c.cancel()
		if c.runtime != nil && c.runtime.Close != nil {
			_ = c.runtime.Close()
		}
	})
}

// Close releases the host-owned control plane. It is safe to call more than
// once and from the application shutdown callback.
func (c *Controller) Close() {
	c.closeRuntime()
}

func (c *Controller) show(page string) {
	c.page = page
	c.notice = ""
	c.noticeErr = false
	if c.window != nil {
		c.window.Invalidate()
	}
}

func (c *Controller) refresh(showLoading bool) {
	if c.runtime == nil {
		return
	}
	if c.refreshing {
		return
	}
	c.refreshing = true
	if showLoading && len(c.snapshot.Subscriptions) == 0 {
		c.loading = true
	}
	c.window.Invalidate()
	app := c.runtime.Control
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 20*time.Second)
		defer cancel()
		next, err := loadSnapshot(ctx, app)
		c.window.Update(func() {
			c.refreshing = false
			c.loading = false
			if err != nil {
				c.notice = "刷新失败：" + err.Error()
				c.noticeErr = true
			} else {
				c.snapshot = next
				c.syncDrafts()
				if showLoading {
					c.notice = ""
					c.noticeErr = false
				}
			}
		})
	}()
}

func (c *Controller) syncDrafts() {
	for key, value := range c.snapshot.SettingsByKey {
		if _, edited := c.settingsDraft[key]; !edited {
			c.settingsDraft[key] = value
		}
	}
	for _, key := range []string{control.SettingAutoRefresh, control.SettingSubscriptionMaxAttempts, control.SettingSubscriptionConflictPolicy} {
		if _, exists := c.settingsDraft[key]; !exists {
			c.settingsDraft[key] = c.snapshot.SettingsByKey[key]
		}
	}
}

func (c *Controller) run(label string, fn func(context.Context, control.Application) error) {
	if c.runtime == nil || c.busy {
		return
	}
	c.busy = true
	c.notice = label + "中…"
	c.noticeErr = false
	c.window.Invalidate()
	app := c.runtime.Control
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 45*time.Second)
		defer cancel()
		err := fn(ctx, app)
		c.window.Update(func() {
			c.busy = false
			if err != nil {
				c.notice = label + "失败：" + err.Error()
				c.noticeErr = true
			} else {
				c.notice = label + "完成"
				c.noticeErr = false
			}
			c.refresh(false)
		})
	}()
}

func (c *Controller) toggleTheme() {
	c.dark = !c.dark
	if c.options.OnThemeChange != nil {
		if err := c.options.OnThemeChange(c.dark); err != nil {
			c.notice = "主题偏好保存失败：" + err.Error()
			c.noticeErr = true
		}
	}
	if c.window != nil {
		c.window.Invalidate()
	}
}

func (c *Controller) setNarrow(narrow bool) {
	if c.narrow == narrow {
		return
	}
	c.narrow = narrow
	if c.window != nil {
		c.window.Invalidate()
	}
}

func (c *Controller) pageTitle() (string, string) {
	for _, item := range navigation {
		if item.ID == c.page {
			return item.Label, item.Hint
		}
	}
	return "BOXD", ""
}

func (c *Controller) openEditor(kind editorKind, id string) {
	e := editorState{kind: kind, id: id}
	switch kind {
	case editorSubscription:
		if subscription, exists := c.snapshot.SubscriptionsByID[id]; exists {
			e.name, e.url, e.enabled = subscription.Name, subscription.URL, subscription.Enabled
			e.interval = strconv.Itoa(subscription.IntervalSeconds)
		} else {
			e.interval = "3600"
			e.enabled = true
		}
	case editorNode:
		if node, exists := c.snapshot.NodesByID[id]; exists {
			e.name, e.protocol, e.server = node.Name, node.Protocol, node.Server
			e.port = strconv.Itoa(int(node.Port))
			e.udp = node.NodeSupportsUDP()
			e.subscriptionID = node.SubscriptionID
			if node.Options != nil {
				body, _ := json.MarshalIndent(node.Options, "", "  ")
				e.options = string(body)
			} else {
				e.options = "{}"
			}
		} else {
			e.protocol, e.port, e.udp, e.options = "shadowsocks", "443", true, "{}"
		}
	case editorImportLinks:
		if len(c.snapshot.Catalog.Groups) > 0 {
			e.groupID = c.snapshot.Catalog.Groups[0].ID
		}
	case editorGroup:
		if group, exists := c.snapshot.GroupsByID[id]; exists {
			e.name, e.groupKind, e.selected = group.Name, string(group.Kind), group.Selected
			e.members, e.url = strings.Join(group.Members, "\n"), group.URL
			e.interval = strconv.Itoa(group.IntervalSeconds)
			e.enabled = true
		} else {
			e.groupKind, e.interval = string(control.GroupKindSelector), "300"
			e.enabled = true
		}
	case editorChain:
		if chain, exists := c.snapshot.ChainsByID[id]; exists {
			e.name = chain.Name
			lines := make([]string, 0, len(chain.Hops))
			for _, hop := range chain.Hops {
				lines = append(lines, string(hop.Kind)+":"+hop.ID)
			}
			e.hops = strings.Join(lines, "\n")
		}
	case editorRouteRule:
		for _, rule := range c.snapshot.Routes.Rules {
			if rule.ID != id {
				continue
			}
			e.rule = rule
			e.name, e.action = rule.Name, string(rule.Action)
			e.matchDomains = strings.Join(rule.Match.Domains, "\n")
			e.matchSuffixes = strings.Join(rule.Match.DomainSuffixes, "\n")
			e.matchRuleSets = strings.Join(rule.Match.RuleSets, "\n")
			ports := make([]string, 0, len(rule.Match.Ports))
			for _, port := range rule.Match.Ports {
				ports = append(ports, strconv.Itoa(int(port)))
			}
			e.matchPorts = strings.Join(ports, ",")
			e.egressKind, e.egressID = string(rule.Egress.Kind), rule.Egress.ID
			break
		}
		if e.action == "" {
			e.action, e.egressKind = string(control.RouteActionEgress), string(control.EgressKindGroup)
			if len(c.snapshot.Catalog.Groups) > 0 {
				e.egressID = c.snapshot.Catalog.Groups[0].ID
			}
		}
	case editorDNS:
		e.dnsSlot = id
		switch id {
		case "domestic":
			e.dnsServer = c.snapshot.DNS.Domestic
		case "foreign":
			e.dnsServer = c.snapshot.DNS.Foreign
		default:
			e.dnsServer = c.snapshot.DNS.Default
		}
	case editorRuleSet:
		for _, ruleSet := range c.snapshot.RuleSets {
			if ruleSet.Name != id {
				continue
			}
			e.name, e.url, e.enabled, e.builtin = ruleSet.Name, ruleSet.URL, ruleSet.Enabled, ruleSet.Builtin
			e.format = string(ruleSet.Format)
			break
		}
	}
	c.editor = e
	c.editorOpen = true
}

func (c *Controller) askDelete(kind, id, name string) {
	c.deleteKind, c.deleteID, c.deleteName = kind, id, name
	c.deleteOpen = true
}

func (c *Controller) application() control.Application {
	if c.runtime == nil {
		return nil
	}
	return c.runtime.Control
}

func (c *Controller) confirmDelete() {
	kind, id := c.deleteKind, c.deleteID
	c.deleteOpen = false
	c.run("删除", func(ctx context.Context, app control.Application) error {
		switch kind {
		case "subscription":
			return app.Subscriptions().Delete(ctx, id)
		case "node":
			return app.Nodes().Delete(ctx, id)
		case "group":
			return app.Egress().DeleteGroup(ctx, id)
		case "chain":
			return app.Egress().DeleteChain(ctx, id)
		case "rule":
			return app.Routes().DeleteRule(ctx, id)
		case "ruleset":
			return app.RuleSets().Delete(ctx, id)
		default:
			return fmt.Errorf("unsupported delete target %q", kind)
		}
	})
}

func (c *Controller) saveEditor(ctx context.Context, app control.Application, e editorState) error {
	switch e.kind {
	case editorSubscription:
		interval, err := strconv.Atoi(strings.TrimSpace(e.interval))
		if err != nil || interval < 0 {
			return fmt.Errorf("同步间隔必须是大于或等于 0 的整数")
		}
		patch := control.SubscriptionPatch{Name: &e.name, URL: &e.url, Enabled: &e.enabled, IntervalSeconds: &interval}
		if e.id == "" {
			_, err = app.Subscriptions().Save(ctx, control.Subscription{Name: e.name, URL: e.url, Enabled: e.enabled, IntervalSeconds: interval})
			return err
		}
		_, err = app.Subscriptions().Patch(ctx, e.id, patch)
		return err
	case editorNode:
		port, err := strconv.Atoi(strings.TrimSpace(e.port))
		if err != nil || port <= 0 || port > 65535 {
			return fmt.Errorf("端口必须在 1 到 65535 之间")
		}
		options := map[string]any{}
		if strings.TrimSpace(e.options) != "" {
			if err := json.Unmarshal([]byte(e.options), &options); err != nil {
				return fmt.Errorf("协议参数不是有效 JSON：%w", err)
			}
		}
		node := control.Node{
			ID: e.id, Name: e.name, Protocol: e.protocol, Server: e.server,
			Port: uint16(port), Options: options, SupportsUDP: &e.udp, SubscriptionID: e.subscriptionID,
		}
		_, err = app.Nodes().Save(ctx, node)
		return err
	case editorImportLinks:
		links := splitLines(e.links)
		_, err := app.Nodes().ImportLinksIntoGroup(ctx, links, strings.TrimSpace(e.groupID))
		return err
	case editorGroup:
		members := splitLines(e.members)
		group := control.Group{ID: e.id, Name: e.name, Kind: control.GroupKind(e.groupKind), Members: members, Selected: e.selected, URL: e.url}
		group.IntervalSeconds, _ = strconv.Atoi(strings.TrimSpace(e.interval))
		_, err := app.Egress().SaveGroup(ctx, group)
		return err
	case editorChain:
		lines := splitLines(e.hops)
		hops := make([]control.ProxyHop, 0, len(lines))
		for _, line := range lines {
			kind, id, found := strings.Cut(line, ":")
			if !found || strings.TrimSpace(kind) == "" || strings.TrimSpace(id) == "" {
				return fmt.Errorf("链路节点格式应为 node:<id>、group:<id> 或 chain:<id>")
			}
			hops = append(hops, control.ProxyHop{Kind: control.EgressKind(strings.TrimSpace(kind)), ID: strings.TrimSpace(id)})
		}
		_, err := app.Egress().SaveChain(ctx, control.ProxyChain{ID: e.id, Name: e.name, Hops: hops})
		return err
	case editorRouteRule:
		rule := e.rule
		rule.Name = e.name
		rule.Action = control.RouteAction(e.action)
		rule.Match.Domains = splitLines(e.matchDomains)
		rule.Match.DomainSuffixes = splitLines(e.matchSuffixes)
		rule.Match.RuleSets = splitLines(e.matchRuleSets)
		rule.Match.Ports = parsePorts(e.matchPorts)
		rule.Egress = control.EgressRef{Kind: control.EgressKind(e.egressKind), ID: strings.TrimSpace(e.egressID)}
		if e.id == "" {
			_, err := app.Routes().SaveRule(ctx, rule)
			return err
		}
		name, disabled, match, action, egress := rule.Name, rule.Disabled, rule.Match, rule.Action, rule.Egress
		_, err := app.Routes().PatchRule(ctx, e.id, control.RouteRulePatch{
			Name: &name, Disabled: &disabled, Match: &match, Action: &action, Egress: &egress,
		})
		return err
	case editorDNS:
		server := e.dnsServer
		server.Address = strings.TrimSpace(server.Address)
		patch := control.DNSProfilePatch{}
		switch e.dnsSlot {
		case "domestic":
			patch.Domestic = &server
		case "foreign":
			patch.Foreign = &server
		default:
			patch.Default = &server
		}
		_, err := app.DNS().Patch(ctx, patch)
		return err
	case editorRuleSet:
		if e.id == "" {
			_, err := app.RuleSets().Save(ctx, control.RuleSetDefinition{
				Name: e.name, URL: e.url, Format: control.RuleSetFormat(e.format), Enabled: e.enabled,
			})
			return err
		}
		var err error
		if !e.builtin {
			_, err = app.RuleSets().Save(ctx, control.RuleSetDefinition{
				Name: e.id, URL: e.url, Format: control.RuleSetFormat(e.format), Enabled: e.enabled,
			})
			return err
		}
		_, err = app.RuleSets().Patch(ctx, e.id, control.RuleSetPatch{Enabled: &e.enabled})
		return err
	default:
		return fmt.Errorf("unsupported editor")
	}
}

func splitLines(value string) []string {
	raw := strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == '\r' })
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func parsePorts(value string) []uint16 {
	items := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' })
	result := make([]uint16, 0, len(items))
	for _, item := range items {
		port, err := strconv.Atoi(strings.TrimSpace(item))
		if err == nil && port > 0 && port <= 65535 {
			result = append(result, uint16(port))
		}
	}
	return result
}

func egressOptions(s snapshot) []string {
	options := []string{"direct:直连", "reject:拒绝"}
	for _, group := range s.Catalog.Groups {
		options = append(options, "group:"+group.ID+" · "+group.Name)
	}
	for _, chain := range s.Catalog.Chains {
		options = append(options, "chain:"+chain.ID+" · "+chain.Name)
	}
	for _, node := range s.Catalog.Nodes {
		options = append(options, "node:"+node.ID+" · "+node.Name)
	}
	return options
}

func optionValue(value string) string {
	kind, rest, found := strings.Cut(value, ":")
	if !found {
		return value
	}
	id, _, _ := strings.Cut(rest, " · ")
	if id == "" && (kind == "direct" || kind == "reject") {
		return kind
	}
	return kind + ":" + id
}

func labelForEgress(ref control.EgressRef, s snapshot) string {
	switch ref.Kind {
	case control.EgressKindDirect:
		return "直连"
	case control.EgressKindReject:
		return "拒绝"
	}
	switch ref.Kind {
	case control.EgressKindNode:
		if node, exists := s.NodesByID[ref.ID]; exists {
			return node.Name
		}
	case control.EgressKindGroup:
		if group, exists := s.GroupsByID[ref.ID]; exists {
			return group.Name
		}
	case control.EgressKindChain:
		if chain, exists := s.ChainsByID[ref.ID]; exists {
			return chain.Name
		}
	}
	return ref.ID
}

func kernelStateLabel(state kernel.State) string {
	switch state {
	case kernel.StateRunning:
		return "运行中"
	case kernel.StateStarting:
		return "启动中"
	case kernel.StateStopping:
		return "停止中"
	case kernel.StateFailed:
		return "异常"
	default:
		return "未启动"
	}
}
