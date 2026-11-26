package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/logtext"
	"github.com/xuthus5/boxd/kernel"
)

type snapshotView struct {
	ServerTime    time.Time               `json:"server_time"`
	Revision      string                  `json:"revision"`
	Kernel        kernelView              `json:"kernel"`
	Capabilities  capabilitiesView        `json:"capabilities"`
	Scheduler     control.SchedulerStatus `json:"scheduler"`
	Inbounds      []inboundView           `json:"inbounds"`
	Nodes         []nodeView              `json:"nodes"`
	Groups        []groupView             `json:"groups"`
	Chains        []chainView             `json:"chains"`
	Routes        routeView               `json:"routes"`
	DNS           dnsProfileView          `json:"dns"`
	RuleSets      []ruleSetView           `json:"rulesets"`
	Subscriptions []subscriptionView      `json:"subscriptions"`
	Settings      map[string]string       `json:"settings"`
	Connections   []connectionView        `json:"connections"`
	Traffic       trafficView             `json:"traffic"`
	Logs          logsView                `json:"logs"`
	Overview      overviewView            `json:"overview"`
	Audit         map[string]any          `json:"-"`
}

type capabilitiesView struct {
	Kind    kernel.EngineKind   `json:"kind"`
	Version string              `json:"version"`
	Items   kernel.Capabilities `json:"items"`
}

type kernelErrorView struct {
	Code      kernel.ErrorCode `json:"code"`
	Op        string           `json:"op,omitempty"`
	Message   string           `json:"message"`
	Hint      string           `json:"hint,omitempty"`
	Retryable bool             `json:"retryable,omitempty"`
	Temporary bool             `json:"temporary,omitempty"`
}

type kernelView struct {
	Kind          kernel.EngineKind `json:"kind"`
	State         kernel.State      `json:"state"`
	Version       string            `json:"version"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	Revision      string            `json:"revision"`
	UptimeSeconds int64             `json:"uptime_seconds"`
	LastError     *kernelErrorView  `json:"last_error,omitempty"`
}

type inboundView struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Kind        control.InboundKind `json:"kind"`
	Enabled     bool                `json:"enabled"`
	Listen      string              `json:"listen"`
	UDP         bool                `json:"udp"`
	Auth        bool                `json:"auth"`
	MTU         int                 `json:"mtu,omitempty"`
	Stack       string              `json:"stack,omitempty"`
	Connections int                 `json:"connections"`
	Traffic     string              `json:"traffic"`
	Rate        string              `json:"rate"`
	Observed    bool                `json:"observed"`
	Bytes       int64               `json:"bytes"`
	RateBytes   int64               `json:"rate_bytes"`
}

type nodeView struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Protocol       string   `json:"protocol"`
	Server         string   `json:"server"`
	Port           uint16   `json:"port"`
	UDP            bool     `json:"udp"`
	Group          string   `json:"group"`
	Groups         []string `json:"groups"`
	Status         string   `json:"status"`
	Traffic        string   `json:"traffic"`
	Rate           string   `json:"rate"`
	Observed       bool     `json:"observed"`
	TrafficBytes   int64    `json:"traffic_bytes"`
	RateBytes      int64    `json:"rate_bytes"`
	SubscriptionID string   `json:"subscription_id,omitempty"`
}

type groupView struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Kind        control.GroupKind `json:"kind"`
	Selected    string            `json:"selected,omitempty"`
	Members     []string          `json:"members"`
	Description string            `json:"description"`
	Latency     int               `json:"latency"`
	URL         string            `json:"url,omitempty"`
	Interval    int               `json:"interval_seconds,omitempty"`
}

type chainView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Hops        []string `json:"hops"`
	Description string   `json:"description"`
	UsedBy      string   `json:"usedBy"`
	Latency     int      `json:"latency"`
}

type routeRuleView struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Disabled  bool                `json:"disabled"`
	Enabled   bool                `json:"enabled"`
	Condition string              `json:"condition"`
	Type      string              `json:"type"`
	Action    control.RouteAction `json:"route_action"`
	ActionUI  string              `json:"action"`
	Target    string              `json:"target"`
	Hits      string              `json:"hits"`
	Match     control.RouteMatch  `json:"match"`
	Egress    control.EgressRef   `json:"egress"`
}

type routeView struct {
	Mode          control.RouteMode    `json:"mode"`
	DefaultEgress control.EgressRef    `json:"default_egress"`
	DefaultTarget string               `json:"default_target"`
	Presets       control.RoutePresets `json:"presets"`
	Rules         []routeRuleView      `json:"rules"`
}

type dnsSlotView struct {
	Slot     string              `json:"slot"`
	Key      string              `json:"key"`
	Enabled  bool                `json:"enabled"`
	Provider control.DNSProvider `json:"provider"`
	Protocol string              `json:"protocol"`
	Address  string              `json:"address"`
	Detour   string              `json:"detour"`
	Latency  int                 `json:"latency"`
	TestedAt *time.Time          `json:"tested_at,omitempty"`
	Error    string              `json:"error,omitempty"`
	Skipped  bool                `json:"skipped,omitempty"`
	Reason   string              `json:"reason,omitempty"`
}

type dnsProfileView struct {
	Strategy string                  `json:"strategy"`
	Cache    bool                    `json:"cache"`
	Domestic control.DNSServerConfig `json:"domestic"`
	Foreign  control.DNSServerConfig `json:"foreign"`
	Default  control.DNSServerConfig `json:"default"`
	Slots    []dnsSlotView           `json:"slots"`
}

type ruleSetView struct {
	Name        string                `json:"name"`
	URL         string                `json:"url"`
	Format      control.RuleSetFormat `json:"format"`
	Enabled     bool                  `json:"enabled"`
	Builtin     bool                  `json:"builtin"`
	Description string                `json:"description"`
	Size        string                `json:"size"`
	Updated     string                `json:"updated"`
	UpdatedAt   *time.Time            `json:"updated_at,omitempty"`
	SHA         string                `json:"sha"`
	Status      string                `json:"status"`
	Error       string                `json:"error,omitempty"`
}

type subscriptionView struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Provider      string     `json:"provider"`
	URL           string     `json:"url"`
	Nodes         int        `json:"nodes"`
	Enabled       bool       `json:"enabled"`
	Interval      int        `json:"interval"`
	LastRefresh   string     `json:"lastRefresh"`
	LastRefreshAt *time.Time `json:"last_refreshed_at,omitempty"`
	ETag          string     `json:"etag,omitempty"`
	State         string     `json:"state"`
}

type connectionView struct {
	ID            string     `json:"id"`
	NumericID     int64      `json:"numeric_id"`
	Target        string     `json:"target"`
	Inbound       string     `json:"inbound"`
	Network       string     `json:"network"`
	Source        string     `json:"source"`
	Outbound      string     `json:"outbound"`
	Rule          string     `json:"rule"`
	Process       string     `json:"process"`
	Upload        string     `json:"upload"`
	Download      string     `json:"download"`
	Duration      string     `json:"duration"`
	UploadBytes   int64      `json:"upload_bytes"`
	DownloadBytes int64      `json:"download_bytes"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
}

type trafficView struct {
	UploadTotal   int64     `json:"upload_total"`
	DownloadTotal int64     `json:"download_total"`
	UploadRate    int64     `json:"upload_rate"`
	DownloadRate  int64     `json:"download_rate"`
	Available     bool      `json:"available"`
	At            time.Time `json:"at"`
}

type logView struct {
	ID      string         `json:"id"`
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Source  string         `json:"source"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

type logsView struct {
	App    []logView `json:"app"`
	Kernel []logView `json:"kernel"`
}

type statView struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Delta  string `json:"delta"`
	Tone   string `json:"tone"`
	Detail string `json:"detail"`
}

type healthView struct {
	Name    string `json:"name"`
	Meta    string `json:"meta"`
	State   string `json:"state"`
	Latency int    `json:"latency"`
}

type quickActionView struct {
	Icon  string `json:"icon"`
	Label string `json:"label"`
	View  string `json:"view"`
	Tone  string `json:"tone"`
}

type overviewView struct {
	Stats        []statView        `json:"stats"`
	Health       []healthView      `json:"health"`
	QuickActions []quickActionView `json:"quickActions"`
}

func (s *Server) buildSnapshot(ctx context.Context) (snapshotView, error) {
	kernelStatus, err := s.app.Kernel().Status(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	capabilities, err := s.app.Setup().Capabilities(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	schedulerStatus, err := s.app.Scheduler().Status(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	inbounds, err := s.app.Inbounds().List(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	catalog, err := s.app.Egress().Catalog(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	routes, err := s.app.Routes().Get(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	dns, err := s.app.DNS().Get(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	dnsProbes, err := s.app.DNS().ProbeResults(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	ruleSets, err := s.app.RuleSets().List(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	subscriptions, err := s.app.Subscriptions().List(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	settings, err := s.app.Settings().List(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	connections, err := s.app.Observability().Connections(ctx)
	if err != nil && !isNotRunning(err) {
		return snapshotView{}, err
	}
	traffic, err := s.app.Observability().Traffic(ctx)
	if err != nil {
		return snapshotView{}, err
	}

	now := time.Now().UTC()
	groupByNode := make(map[string][]string)
	for _, group := range catalog.Groups {
		for _, member := range group.Members {
			groupByNode[member] = append(groupByNode[member], group.ID)
		}
	}
	connectionViews := makeConnectionViews(connections, now)
	observation, err := s.app.Observability().Snapshot(ctx)
	if err != nil {
		return snapshotView{}, err
	}
	nodeViews := makeNodeViewsWithObservation(catalog.Nodes, groupByNode, catalog.Groups, observation)
	groupViews := makeGroupViews(catalog.Groups, catalog.Nodes)
	routeViews := makeRouteViewsWithObservation(routes, catalog, observation)
	snapshot := snapshotView{
		ServerTime:    now,
		Revision:      kernelStatus.Revision,
		Kernel:        makeKernelView(kernelStatus, now),
		Capabilities:  capabilitiesView{Kind: capabilities.Kind, Version: capabilities.Version, Items: capabilities.Capabilities},
		Scheduler:     schedulerStatus,
		Inbounds:      makeInboundViewsWithObservation(inbounds, connections, observation),
		Nodes:         nodeViews,
		Groups:        groupViews,
		Chains:        makeChainViews(catalog.Chains, catalog.Nodes, routes),
		Routes:        routeViews,
		DNS:           makeDNSViewWithProbes(dns, dnsProbes),
		RuleSets:      makeRuleSetViews(ruleSets, now),
		Subscriptions: makeSubscriptionViews(subscriptions, catalog.Nodes, now),
		Settings:      makeSettingsMap(settings),
		Connections:   connectionViews,
		Traffic:       makeTrafficView(traffic, now, kernelStatus.State == kernel.StateRunning),
		Logs: logsView{
			App:    makeAppLogViews(s.logs.recent(200)),
			Kernel: makeKernelLogViews(s.app.Observability().RecentLogs(200)),
		},
	}
	snapshot.Overview = makeOverview(snapshot, now)
	return snapshot, nil
}

func makeKernelView(status kernel.Status, now time.Time) kernelView {
	view := kernelView{
		Kind:      status.Kind,
		State:     status.State,
		Version:   status.Version,
		Revision:  status.Revision,
		LastError: makeKernelErrorView(status.LastError),
	}
	if !status.StartedAt.IsZero() {
		startedAt := status.StartedAt.UTC()
		view.StartedAt = &startedAt
		if now.After(startedAt) {
			view.UptimeSeconds = int64(now.Sub(startedAt).Seconds())
		}
	}
	return view
}

func makeKernelErrorView(err *kernel.Error) *kernelErrorView {
	if err == nil {
		return nil
	}
	return &kernelErrorView{Code: err.Code, Op: err.Op, Message: err.Message, Hint: err.Hint, Retryable: err.Retryable, Temporary: err.Temporary}
}

func makeInboundViews(inbounds []control.InboundProfile, connections []kernel.Connection) []inboundView {
	return makeInboundViewsWithObservation(inbounds, connections, observationSnapshot{})
}

func makeInboundViewsWithObservation(inbounds []control.InboundProfile, connections []kernel.Connection, observation observationSnapshot) []inboundView {
	result := make([]inboundView, 0, len(inbounds))
	for _, inbound := range inbounds {
		view := inboundView{
			ID: inbound.ID, Name: inbound.Name, Kind: inbound.Kind, Enabled: inbound.Enabled,
			UDP: inbound.UDP, Auth: inbound.Auth != nil, MTU: inbound.Tun.MTU, Stack: inbound.Tun.Stack,
			Traffic: "未采集", Rate: "未采集",
		}
		if !inbound.Enabled {
			view.Traffic = "已停用"
			view.Rate = "—"
		} else if observation.Available {
			traffic := observation.Inbounds[inbound.ID]
			view.Observed = true
			view.Bytes = traffic.Total()
			view.RateBytes = traffic.TotalRate()
			view.Traffic = formatBytes(view.Bytes)
			view.Rate = formatRate(view.RateBytes)
		}
		if inbound.Kind == control.InboundKindTun {
			view.Listen = strings.Join(inbound.Tun.Addresses, " · ")
		} else {
			view.Listen = net.JoinHostPort(inbound.Listen.Host, strconv.Itoa(int(inbound.Listen.Port)))
		}
		for _, connection := range connections {
			if connectionInboundMatches(connection.Inbound, inbound) {
				view.Connections++
			}
		}
		result = append(result, view)
	}
	return result
}

func connectionInboundMatches(value string, inbound control.InboundProfile) bool {
	value = strings.TrimSpace(value)
	if index := strings.LastIndexByte(value, '/'); index >= 0 && index+1 < len(value) {
		value = value[index+1:]
	}
	return value == inbound.ID || value == control.InboundTag(inbound.ID) || strings.EqualFold(value, inbound.Name)
}

func makeNodeViews(nodes []control.Node, groupByNode map[string][]string, groups []control.Group) []nodeView {
	return makeNodeViewsWithObservation(nodes, groupByNode, groups, observationSnapshot{})
}

func makeNodeViewsWithObservation(nodes []control.Node, groupByNode map[string][]string, groups []control.Group, observation observationSnapshot) []nodeView {
	groupNameByID := make(map[string]string, len(groups))
	for _, group := range groups {
		groupNameByID[group.ID] = group.Name
	}
	result := make([]nodeView, 0, len(nodes))
	for _, node := range nodes {
		groupIDs := append([]string(nil), groupByNode[node.ID]...)
		groupName := "未分组"
		if len(groupIDs) > 0 {
			groupName = groupNameByID[groupIDs[0]]
		}
		view := nodeView{
			ID: node.ID, Name: node.Name,
			Protocol: node.Protocol, Server: node.Server, Port: node.Port, UDP: node.NodeSupportsUDP(),
			Group: groupName, Groups: groupIDs, Status: "unknown", Traffic: "未采集",
			SubscriptionID: node.SubscriptionID,
		}
		if observation.Available {
			traffic := observation.Nodes[node.ID]
			view.Observed = true
			view.TrafficBytes = traffic.Total()
			view.RateBytes = traffic.TotalRate()
			view.Traffic = formatBytes(view.TrafficBytes)
			view.Rate = formatRate(view.RateBytes)
		}
		result = append(result, view)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func makeGroupViews(groups []control.Group, nodes []control.Node) []groupView {
	nodeNames := make(map[string]string, len(nodes))
	for _, node := range nodes {
		nodeNames[node.ID] = node.Name
	}
	result := make([]groupView, 0, len(groups))
	for _, group := range groups {
		description := "手动选择当前出口"
		if group.Kind == control.GroupKindURLTest {
			description = "按间隔自动测速并选择延迟最低的成员"
		}
		_ = nodeNames
		result = append(result, groupView{
			ID: group.ID, Name: group.Name, Kind: group.Kind, Selected: group.Selected,
			Members: append([]string(nil), group.Members...), Description: description,
			URL: group.URL, Interval: group.IntervalSeconds,
		})
	}
	return result
}

func makeChainViews(chains []control.ProxyChain, nodes []control.Node, routes control.RouteProfile) []chainView {
	nodeNames := make(map[string]string, len(nodes))
	for _, node := range nodes {
		nodeNames[node.ID] = node.Name
	}
	result := make([]chainView, 0, len(chains))
	for _, chain := range chains {
		hops := make([]string, 0, len(chain.Hops))
		names := make([]string, 0, len(chain.Hops))
		for _, hop := range chain.Hops {
			hops = append(hops, hop.ID)
			name := nodeNames[hop.ID]
			if name == "" {
				name = hop.ID
			}
			names = append(names, name)
		}
		usedBy := "未绑定"
		for _, rule := range routes.Rules {
			if rule.Egress.Kind == control.EgressKindChain && rule.Egress.ID == chain.ID {
				usedBy = rule.Name
				break
			}
		}
		result = append(result, chainView{ID: chain.ID, Name: chain.Name, Hops: hops, Description: strings.Join(names, " → "), UsedBy: usedBy})
	}
	return result
}

func makeRouteViews(routes control.RouteProfile, catalog control.EgressCatalog) routeView {
	return makeRouteViewsWithObservation(routes, catalog, observationSnapshot{})
}

func makeRouteViewsWithObservation(routes control.RouteProfile, catalog control.EgressCatalog, observation observationSnapshot) routeView {
	names := egressNames(catalog)
	result := routeView{Mode: routes.Mode, DefaultEgress: routes.DefaultEgress, DefaultTarget: egressLabel(routes.DefaultEgress, names), Presets: routes.Presets}
	result.Rules = make([]routeRuleView, 0, len(routes.Rules))
	for _, rule := range routes.Rules {
		condition, conditionType := describeMatch(rule.Match)
		action := actionLabel(rule.Action, rule.Egress)
		hits := "未统计"
		switch {
		case rule.Disabled:
			hits = "已停用"
		case observation.Available && routes.Mode == control.RouteModeRules:
			hits = fmt.Sprintf("%d 次", observation.RuleHits[rule.ID])
		case observation.Available:
			hits = "未生效"
		}
		result.Rules = append(result.Rules, routeRuleView{
			ID: rule.ID, Name: rule.Name, Disabled: rule.Disabled, Enabled: !rule.Disabled,
			Condition: condition, Type: conditionType, Action: rule.Action, ActionUI: action,
			Target: egressLabel(rule.Egress, names), Hits: hits, Match: rule.Match, Egress: rule.Egress,
		})
	}
	return result
}

func egressNames(catalog control.EgressCatalog) map[string]string {
	result := make(map[string]string)
	for _, node := range catalog.Nodes {
		result[string(control.EgressKindNode)+":"+node.ID] = node.Name
	}
	for _, group := range catalog.Groups {
		result[string(control.EgressKindGroup)+":"+group.ID] = group.Name
	}
	for _, chain := range catalog.Chains {
		result[string(control.EgressKindChain)+":"+chain.ID] = chain.Name
	}
	return result
}

func egressLabel(ref control.EgressRef, names map[string]string) string {
	switch ref.Kind {
	case control.EgressKindDirect:
		return "直连"
	case control.EgressKindReject:
		return "拒绝"
	}
	if name := names[string(ref.Kind)+":"+ref.ID]; name != "" {
		return name
	}
	if ref.ID == "" {
		return "未设置"
	}
	return ref.ID
}

func actionLabel(action control.RouteAction, egress control.EgressRef) string {
	switch action {
	case control.RouteActionDirect:
		return "direct"
	case control.RouteActionReject:
		return "reject"
	default:
		if egress.Kind == control.EgressKindChain {
			return "chain"
		}
		if egress.Kind == control.EgressKindGroup || egress.Kind == control.EgressKindNode {
			return "group"
		}
		return "node"
	}
}

func describeMatch(match control.RouteMatch) (string, string) {
	parts := make([]string, 0)
	kind := "组合条件"
	add := func(values []string) {
		if len(values) > 0 {
			parts = append(parts, values...)
		}
	}
	if len(match.Domains) > 0 {
		kind = "域名"
		add(match.Domains)
	}
	if len(match.DomainSuffixes) > 0 {
		kind = "域名后缀"
		for _, value := range match.DomainSuffixes {
			parts = append(parts, "*."+strings.TrimPrefix(value, "."))
		}
	}
	if len(match.DomainKeywords) > 0 {
		kind = "域名关键字"
		add(match.DomainKeywords)
	}
	if len(match.IPCIDRs) > 0 {
		kind = "IP CIDR"
		add(match.IPCIDRs)
	}
	if len(match.ProcessNames) > 0 {
		kind = "进程"
		add(match.ProcessNames)
	}
	if len(match.RuleSets) > 0 {
		kind = "规则集"
		add(match.RuleSets)
	}
	if len(match.InboundIDs) > 0 {
		if kind == "组合条件" {
			kind = "入口"
		}
		add(match.InboundIDs)
	}
	if len(match.Ports) > 0 {
		if kind == "组合条件" {
			kind = "端口"
		}
		for _, port := range match.Ports {
			parts = append(parts, strconv.Itoa(int(port)))
		}
	}
	if match.PrivateIP {
		parts = append(parts, "私有地址")
	}
	if match.SourcePrivateIP {
		parts = append(parts, "私有来源")
	}
	if len(parts) == 0 {
		return "所有流量", "默认"
	}
	return strings.Join(parts, " · "), kind
}

func makeDNSView(profile control.DNSProfile) dnsProfileView {
	return makeDNSViewWithProbes(profile, nil)
}

func makeDNSViewWithProbes(profile control.DNSProfile, probes []control.DNSProbeResult) dnsProfileView {
	results := make(map[string]control.DNSProbeResult, len(probes))
	for _, probe := range probes {
		results[probe.Key] = probe
	}
	return dnsProfileView{
		Strategy: profile.Strategy,
		Cache:    profile.Cache,
		Domestic: profile.Domestic,
		Foreign:  profile.Foreign,
		Default:  profile.Default,
		Slots: []dnsSlotView{
			makeDNSSlot("国内 DNS", "domestic", profile.Domestic, results["domestic"]),
			makeDNSSlot("国外 DNS", "foreign", profile.Foreign, results["foreign"]),
			makeDNSSlot("默认 DNS", "default", profile.Default, results["default"]),
		},
	}
}

func makeDNSSlot(label, key string, server control.DNSServerConfig, probe control.DNSProbeResult) dnsSlotView {
	provider := map[control.DNSProvider]string{
		control.DNSProviderAliDNS: "阿里 DNS", control.DNSProviderTencent: "腾讯 DNS",
		control.DNSProviderCloudflare: "Cloudflare", control.DNSProviderGoogle: "Google",
		control.DNSProviderQuad9: "Quad9", control.DNSProviderCustom: "自定义",
	}[server.Provider]
	if provider == "" {
		provider = string(server.Provider)
	}
	detour := map[control.DNSDetour]string{
		control.DNSDetourDirect: "直连", control.DNSDetourRouteDefault: "跟随路由出口", control.DNSDetourEgress: "指定出口",
	}[server.Detour]
	if detour == "" {
		detour = string(server.Detour)
	}
	view := dnsSlotView{
		Slot: label, Key: key, Enabled: server.Enabled, Provider: server.Provider,
		Protocol: strings.ToUpper(string(server.Protocol)), Address: server.Address, Detour: detour,
		Error: probe.Error, Skipped: probe.Skipped, Reason: probe.Reason,
	}
	if !probe.TestedAt.IsZero() {
		testedAt := probe.TestedAt.UTC()
		view.TestedAt = &testedAt
	}
	if probe.Error == "" && !probe.Skipped && probe.Latency > 0 {
		view.Latency = int(probe.Latency.Milliseconds())
		if view.Latency <= 0 {
			view.Latency = 1
		}
	}
	return view
}

func makeRuleSetViews(ruleSets []control.RuleSet, now time.Time) []ruleSetView {
	descriptions := map[string]string{
		"geosite-cn": "中国大陆域名", "geoip-cn": "中国大陆 IP", "geosite-ads": "广告与追踪域名",
	}
	result := make([]ruleSetView, 0, len(ruleSets))
	for _, ruleSet := range ruleSets {
		status := "idle"
		updated := "从未刷新"
		if ruleSet.Status.Error != "" {
			status = "error"
		} else if !ruleSet.Status.UpdatedAt.IsZero() {
			status = "ready"
			updated = relativeTime(ruleSet.Status.UpdatedAt, now)
		}
		sha := ruleSet.Status.SHA256
		if len(sha) > 12 {
			sha = sha[:12]
		}
		if sha == "" {
			sha = "未记录"
		}
		result = append(result, ruleSetView{
			Name: ruleSet.Name, URL: sanitizeURL(ruleSet.URL), Format: ruleSet.Format, Enabled: ruleSet.Enabled,
			Builtin: ruleSet.Builtin, Description: descriptions[ruleSet.Name], Size: ruleSetSize(ruleSet.Status.SizeBytes), Updated: updated,
			SHA: sha, Status: status, Error: ruleSet.Status.Error,
		})
		if !ruleSet.Status.UpdatedAt.IsZero() {
			updatedAt := ruleSet.Status.UpdatedAt.UTC()
			result[len(result)-1].UpdatedAt = &updatedAt
		}
		if descriptions[ruleSet.Name] == "" {
			result[len(result)-1].Description = "自定义规则集"
		}
	}
	return result
}

func ruleSetSize(size int64) string {
	if size <= 0 {
		return "未记录"
	}
	return formatBytes(size)
}

func makeSubscriptionViews(subscriptions []control.Subscription, nodes []control.Node, now time.Time) []subscriptionView {
	nodeCount := make(map[string]int)
	for _, node := range nodes {
		if node.SubscriptionID != "" {
			nodeCount[node.SubscriptionID]++
		}
	}
	result := make([]subscriptionView, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		lastRefresh := "从未刷新"
		var lastRefreshAt *time.Time
		if !subscription.LastRefreshedAt.IsZero() {
			value := subscription.LastRefreshedAt.UTC()
			lastRefreshAt = &value
			lastRefresh = relativeTime(value, now)
		}
		state := subscriptionState(subscription, now)
		etag := subscription.ETag
		if len(etag) > 32 {
			etag = etag[:32] + "…"
		}
		result = append(result, subscriptionView{
			ID: subscription.ID, Name: subscription.Name, Provider: subscriptionProvider(subscription.URL),
			URL: sanitizeURL(subscription.URL), Enabled: subscription.Enabled, Interval: subscription.IntervalSeconds,
			Nodes: nodeCount[subscription.ID], LastRefresh: lastRefresh, LastRefreshAt: lastRefreshAt, ETag: etag, State: state,
		})
	}
	return result
}

func subscriptionState(subscription control.Subscription, now time.Time) string {
	if !subscription.Enabled {
		return "paused"
	}
	if subscription.IntervalSeconds <= 0 {
		return "manual"
	}
	if subscription.LastRefreshedAt.IsZero() {
		return "pending"
	}
	dueAt := subscription.LastRefreshedAt.Add(time.Duration(subscription.IntervalSeconds)*time.Second + 2*time.Minute)
	if now.After(dueAt) {
		return "degraded"
	}
	return "ok"
}

func subscriptionProvider(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return "远端订阅"
}

func sanitizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "—"
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword("***", "***")
	}
	if parsed.RawQuery != "" {
		query := parsed.Query()
		parts := make([]string, 0, len(query))
		for key := range query {
			parts = append(parts, url.QueryEscape(key)+"=***")
		}
		sort.Strings(parts)
		parsed.RawQuery = strings.Join(parts, "&")
	}
	return parsed.String()
}

func makeSettingsMap(settings []control.Setting) map[string]string {
	defaults := map[string]string{
		"theme": "light", "autoRefresh": "on", "autoStart": "off", "requestLog": "on",
		"accessControl": "on", "dnsCache": "on", "compactTable": "off", "runtime-log-level": "info",
		control.SettingSubscriptionMaxAttempts: "3", control.SettingSubscriptionConflictPolicy: control.SubscriptionConflictKeepLocal,
	}
	for _, setting := range settings {
		defaults[setting.Key] = setting.Value
	}
	return defaults
}

func makeConnectionViews(connections []kernel.Connection, now time.Time) []connectionView {
	result := make([]connectionView, 0, len(connections))
	for _, connection := range connections {
		startedAt := connection.StartedAt.UTC()
		duration := time.Duration(0)
		var started *time.Time
		if !startedAt.IsZero() {
			started = &startedAt
			if now.After(startedAt) {
				duration = now.Sub(startedAt)
			}
		}
		result = append(result, connectionView{
			ID: strconv.FormatInt(connection.ID, 10), NumericID: connection.ID, Target: connection.Target,
			Inbound: connection.Inbound, Network: connection.Network, Source: connection.Source,
			Outbound: connection.Outbound, Rule: connection.Rule, Process: connection.Process,
			Upload: formatBytes(connection.Upload), Download: formatBytes(connection.Download), Duration: formatDuration(duration),
			UploadBytes: connection.Upload, DownloadBytes: connection.Download, StartedAt: started,
		})
	}
	return result
}

func makeTrafficView(snapshot kernel.TrafficSnapshot, now time.Time, available bool) trafficView {
	at := snapshot.At
	if at.IsZero() {
		at = now
	}
	return trafficView{
		UploadTotal: snapshot.UploadTotal, DownloadTotal: snapshot.DownloadTotal,
		UploadRate: snapshot.UploadRate, DownloadRate: snapshot.DownloadRate,
		Available: available, At: at,
	}
}

func makeAppLogViews(entries []control.ActivityLogEntry) []logView {
	result := make([]logView, 0, len(entries))
	for _, entry := range entries {
		result = append(result, logView{ID: "app-" + strconv.FormatUint(entry.ID, 10), Time: entry.Time, Level: entry.Level, Source: entry.Source, Message: logtext.StripTerminalFormatting(entry.Message), Fields: entry.Fields})
	}
	return result
}

func makeKernelLogViews(entries []kernel.LogEntry) []logView {
	result := make([]logView, 0, len(entries))
	for index, entry := range entries {
		result = append(result, logView{ID: "kernel-" + strconv.Itoa(index), Time: entry.Time, Level: strings.ToLower(entry.Level), Source: entry.Source, Message: logtext.StripTerminalFormatting(entry.Message), Fields: entry.Fields})
	}
	return result
}

func makeOverview(snapshot snapshotView, now time.Time) overviewView {
	totalBytes := snapshot.Traffic.UploadTotal + snapshot.Traffic.DownloadTotal
	ruleCount := len(snapshot.Routes.Rules)
	connectionValue := strconv.Itoa(len(snapshot.Connections))
	connectionDelta := "实时采样"
	connectionDetail := "来自内核连接快照"
	if !snapshot.Traffic.Available {
		connectionValue = "—"
		connectionDelta = "内核未运行"
		connectionDetail = "启动内核后统计活动连接"
	}
	trafficValue := "未采集"
	trafficDelta := "内核未运行"
	trafficDetail := "启动内核后开始采样"
	if snapshot.Traffic.Available {
		trafficValue = formatBytes(totalBytes)
		trafficDelta = "实时采样"
		trafficDetail = "本次运行上行 " + formatBytes(snapshot.Traffic.UploadTotal)
	}
	stats := []statView{
		{Label: "活动连接", Value: connectionValue, Delta: connectionDelta, Tone: "mint", Detail: connectionDetail},
		{Label: "本次运行流量", Value: trafficValue, Delta: trafficDelta, Tone: "sky", Detail: trafficDetail},
		{Label: "节点总数", Value: strconv.Itoa(len(snapshot.Nodes)), Delta: "已配置", Tone: "amber", Detail: "策略组 " + strconv.Itoa(len(snapshot.Groups)) + " 个"},
		{Label: "路由规则", Value: strconv.Itoa(ruleCount), Delta: strings.ToUpper(string(snapshot.Routes.Mode)), Tone: "violet", Detail: "默认出口 " + snapshot.Routes.DefaultTarget},
	}
	health := make([]healthView, 0, len(snapshot.Subscriptions))
	for _, subscription := range snapshot.Subscriptions {
		health = append(health, healthView{Name: subscription.Name, Meta: fmt.Sprintf("%d 节点 · %s", subscription.Nodes, subscription.LastRefresh), State: subscription.State})
	}
	return overviewView{
		Stats: stats, Health: health,
		QuickActions: []quickActionView{
			{Icon: "rss", Label: "导入订阅", View: "subscriptions", Tone: "sky"},
			{Icon: "nodes", Label: "添加节点", View: "egress", Tone: "mint"},
			{Icon: "route", Label: "编辑规则", View: "routes", Tone: "violet"},
			{Icon: "activity", Label: "查看连接", View: "observability", Tone: "amber"},
		},
	}
}

func relativeTime(value, now time.Time) string {
	delta := now.Sub(value)
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

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exponent := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %cB", float64(value)/float64(div), "KMGTPE"[exponent])
}

func formatRate(value int64) string { return formatBytes(value) + "/s" }

func formatDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	totalSeconds := int64(value.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func isNotRunning(err error) bool {
	var serviceErr *kernel.Error
	return errors.As(err, &serviceErr) && (serviceErr.Code == kernel.ErrCodeNotRunning || serviceErr.Code == kernel.ErrCodeInvalidState)
}
