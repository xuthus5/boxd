package mygo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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
	pageInbounds      = "inbounds"
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

	sidebarWidth       = 216
	sidebarNarrowWidth = 82
)

type navItem struct {
	ID    string
	Label string
	Title string
	Icon  string
	Hint  string
}

var navigation = []navItem{
	{pageOverview, "总览", "运行总览", "overview", "运行状态与关键指标"},
	{pageInbounds, "入口", "入口管理", "inbounds", "本地监听、端口与 TUN 接入"},
	{pageSubscriptions, "订阅", "订阅管理", "subscriptions", "远程节点来源与统一同步策略"},
	{pageNodes, "节点", "节点管理", "nodes", "节点编辑、分组归属与连通性测试"},
	{pageEgress, "出口编排", "出口编排", "egress", "策略组负责选择，链路负责串联出口路径"},
	{pageRoutes, "路由", "路由策略", "routes", "运行模式、默认出口与有序自定义规则"},
	{pageDNS, "DNS", "DNS 解析", "dns", "固定三个解析槽位，按场景选择国内、国外与默认解析"},
	{pageRuleSets, "规则集", "规则集", "rulesets", "规则资源的下载、启用与刷新状态"},
	{pageLogs, "日志", "运行日志", "logs", "应用事件与内核输出统一按时间排列，点击任意行即可复制"},
	{pageConnections, "连接", "活动连接", "connections", "实时数据来自当前内核连接表"},
	{pageSettings, "设置", "设置", "settings", "控制面同步行为、界面偏好与本地运行环境"},
	{pageAbout, "关于", "关于 BOXD", "about", "版本、内核与运行环境信息"},
}

type Controller struct {
	loader  Loader
	window  *mygo.Window
	options ControllerOptions

	runtime  *Runtime
	snapshot snapshot
	clock    *clock

	page        string
	loading     bool
	busy        bool
	refreshing  bool
	notice      string
	noticeErr   bool
	noticeUntil time.Time
	narrow      bool
	dark        bool

	editorOpen bool
	editor     editorState
	editorErr  string
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
	editorInbound
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

	group groupDraft
	chain chainDraft

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

	inboundKind    control.InboundKind
	listenHost     string
	listenPort     string
	tunInterface   string
	tunAddresses   string
	tunMTU         string
	tunStack       string
	tunAutoRoute   bool
	tunStrictRoute bool

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
				c.setStickyNotice("控制面初始化失败："+err.Error(), true)
				c.snapshot = snapshot{}
				return
			}
			c.runtime = runtime
			c.loading = false
			c.setNotice("控制面已就绪", false)
			c.refresh(true)
			go c.watch()
		})
	}()
}

func (c *Controller) setLoadingError(err error) {
	c.loading = false
	c.setStickyNotice(err.Error(), true)
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
				c.expireNotice()
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
	c.clearNotice()
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
				c.setNotice("刷新失败："+err.Error(), true)
			} else {
				c.snapshot = next
				c.syncDrafts()
				if showLoading {
					c.clearNotice()
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
	defaults := map[string]string{
		control.SettingAutoRefresh:                "true",
		control.SettingSubscriptionMaxAttempts:    "3",
		control.SettingSubscriptionConflictPolicy: control.SubscriptionConflictKeepLocal,
	}
	for key, fallback := range defaults {
		if _, exists := c.settingsDraft[key]; !exists {
			c.settingsDraft[key] = fallback
		}
	}
}

func (c *Controller) run(label string, fn func(context.Context, control.Application) error) {
	if c.runtime == nil || c.busy {
		return
	}
	c.busy = true
	c.setNotice(label+"中…", false)
	c.window.Invalidate()
	app := c.runtime.Control
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 45*time.Second)
		defer cancel()
		err := fn(ctx, app)
		c.window.Update(func() {
			c.busy = false
			if err != nil {
				c.setNotice(label+"失败："+err.Error(), true)
			} else {
				c.setNotice(label+"完成", false)
			}
			c.refresh(false)
		})
	}()
}

const (
	noticeSuccessTTL = 4 * time.Second
	noticeErrorTTL   = 8 * time.Second
)

// setNotice publishes a transient result. Messages shown while an operation is
// running stay visible until that operation publishes its final result.
func (c *Controller) setNotice(message string, isError bool) {
	c.notice = message
	c.noticeErr = isError
	c.noticeUntil = time.Time{}
	if message == "" || c.busy {
		return
	}
	ttl := noticeSuccessTTL
	if isError {
		ttl = noticeErrorTTL
	}
	c.noticeUntil = time.Now().Add(ttl)
}

// setStickyNotice is for failures that explain an unavailable screen. They
// remain until the user retries or dismisses them.
func (c *Controller) setStickyNotice(message string, isError bool) {
	c.notice = message
	c.noticeErr = isError
	c.noticeUntil = time.Time{}
}

func (c *Controller) clearNotice() {
	c.notice = ""
	c.noticeErr = false
	c.noticeUntil = time.Time{}
}

func (c *Controller) expireNotice() {
	if c.notice == "" || c.noticeUntil.IsZero() || time.Now().Before(c.noticeUntil) {
		return
	}
	c.clearNotice()
}

func (c *Controller) toggleTheme() {
	c.dark = !c.dark
	if c.options.OnThemeChange != nil {
		if err := c.options.OnThemeChange(c.dark); err != nil {
			c.setNotice("主题偏好保存失败："+err.Error(), true)
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
			return item.Title, item.Hint
		}
	}
	return "BOXD", ""
}

func (c *Controller) openEditor(kind editorKind, id string) {
	e := editorState{kind: kind, id: id}
	c.editorErr = ""
	switch kind {
	case editorInbound:
		for _, inbound := range c.snapshot.Inbounds {
			if inbound.ID != id {
				continue
			}
			e.inboundKind = inbound.Kind
			e.name = inbound.Name
			e.enabled = inbound.Enabled
			e.udp = inbound.UDP
			e.listenHost = inbound.Listen.Host
			e.listenPort = strconv.Itoa(int(inbound.Listen.Port))
			e.tunInterface = inbound.Tun.InterfaceName
			e.tunAddresses = strings.Join(inbound.Tun.Addresses, "\n")
			e.tunMTU = strconv.Itoa(inbound.Tun.MTU)
			e.tunStack = inbound.Tun.Stack
			e.tunAutoRoute = inbound.Tun.AutoRoute
			e.tunStrictRoute = inbound.Tun.StrictRoute
			break
		}
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
			e.group = groupDraft{
				ID: group.ID, Name: group.Name, Kind: group.Kind,
				Members: append([]string(nil), group.Members...), Selected: group.Selected,
				URL: group.URL, Interval: strconv.Itoa(group.IntervalSeconds),
			}
		} else {
			e.group = groupDraft{Kind: control.GroupKindSelector, Interval: "300"}
		}
	case editorChain:
		if chain, exists := c.snapshot.ChainsByID[id]; exists {
			e.chain = chainDraft{ID: chain.ID, Name: chain.Name, Hops: append([]control.ProxyHop(nil), chain.Hops...)}
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
	case editorInbound:
		inbound, err := app.Inbounds().Get(ctx, e.id)
		if err != nil {
			return err
		}
		inbound.Name = strings.TrimSpace(e.name)
		inbound.Enabled = e.enabled
		if inbound.Name == "" {
			return fmt.Errorf("入口名称不能为空")
		}
		if inbound.Kind == control.InboundKindMixed {
			host := strings.TrimSpace(e.listenHost)
			if net.ParseIP(host) == nil {
				return fmt.Errorf("监听地址必须是有效 IP，例如 ::、0.0.0.0 或 127.0.0.1")
			}
			port, err := strconv.Atoi(strings.TrimSpace(e.listenPort))
			if err != nil || port <= 0 || port > 65535 {
				return fmt.Errorf("监听端口必须在 1 到 65535 之间")
			}
			inbound.Listen = control.ListenAddress{Host: host, Port: uint16(port)}
			inbound.UDP = e.udp
		} else {
			mtu, err := strconv.Atoi(strings.TrimSpace(e.tunMTU))
			if err != nil || mtu < 0 || mtu > 65535 {
				return fmt.Errorf("TUN MTU 必须在 0 到 65535 之间")
			}
			inbound.Tun = control.TunOptions{
				InterfaceName: strings.TrimSpace(e.tunInterface),
				Addresses:     splitLines(e.tunAddresses),
				MTU:           mtu,
				Stack:         strings.TrimSpace(e.tunStack),
				AutoRoute:     e.tunAutoRoute,
				StrictRoute:   e.tunStrictRoute,
			}
		}
		_, err = app.Inbounds().Save(ctx, inbound)
		return err
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
		group := e.group.controlGroup()
		diagnostics, err := app.Egress().ValidateGroup(ctx, group)
		if err != nil {
			return err
		}
		if len(diagnostics) > 0 {
			return validationDiagnosticsError(diagnostics)
		}
		_, err = app.Egress().SaveGroup(ctx, group)
		return err
	case editorChain:
		chain := e.chain.chain()
		diagnostics, err := app.Egress().ValidateChain(ctx, chain)
		if err != nil {
			return err
		}
		if len(diagnostics) > 0 {
			return validationDiagnosticsError(diagnostics)
		}
		_, err = app.Egress().SaveChain(ctx, chain)
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
	options := []string{"直连", "拒绝"}
	for _, group := range s.Catalog.Groups {
		options = append(options, "策略组 · "+group.Name)
	}
	for _, chain := range s.Catalog.Chains {
		options = append(options, "链路 · "+chain.Name)
	}
	for _, node := range s.Catalog.Nodes {
		options = append(options, "节点 · "+node.Name)
	}
	return options
}

func egressRefFromLabel(label string, s snapshot) control.EgressRef {
	switch label {
	case "直连":
		return control.EgressRef{Kind: control.EgressKindDirect}
	case "拒绝":
		return control.EgressRef{Kind: control.EgressKindReject}
	}
	kind, name, found := strings.Cut(label, " · ")
	if !found {
		return control.EgressRef{}
	}
	switch kind {
	case "策略组":
		for _, group := range s.Catalog.Groups {
			if group.Name == name {
				return control.EgressRef{Kind: control.EgressKindGroup, ID: group.ID}
			}
		}
	case "链路":
		for _, chain := range s.Catalog.Chains {
			if chain.Name == name {
				return control.EgressRef{Kind: control.EgressKindChain, ID: chain.ID}
			}
		}
	case "节点":
		for _, node := range s.Catalog.Nodes {
			if node.Name == name {
				return control.EgressRef{Kind: control.EgressKindNode, ID: node.ID}
			}
		}
	}
	return control.EgressRef{}
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
		return "内核运行中"
	case kernel.StateStarting:
		return "内核启动中"
	case kernel.StateStopping:
		return "内核停止中"
	case kernel.StateFailed:
		return "内核异常"
	default:
		return "内核未启动"
	}
}
