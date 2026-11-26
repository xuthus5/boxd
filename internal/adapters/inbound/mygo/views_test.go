package mygo

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

func TestNativePagesRenderFromControlPlaneSnapshot(t *testing.T) {
	for _, item := range navigation {
		t.Run(item.ID, func(t *testing.T) {
			controller := newViewTestController(ControllerOptions{})
			controller.page = item.ID
			tester := ui.NewTester(controller.View, 1320, 860)
			if !tester.HasText(item.Label) {
				t.Fatalf("page %s did not render its title; texts=%q", item.ID, tester.Texts())
			}
		})
	}
}

func TestOverviewUsesUnavailablePlaceholders(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	tester := ui.NewTester(controller.View, 1320, 860)
	if !tester.HasText("内核未启动，未采集数据") {
		t.Fatalf("overview placeholder missing; texts=%q", tester.Texts())
	}
	if !tester.HasText("—") {
		t.Fatalf("overview dash placeholder missing; texts=%q", tester.Texts())
	}
}

func TestOverviewActivityShowsTenMostRecentLogs(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	controller.snapshot.Activity = make([]control.ActivityLogEntry, 15)
	for i := range controller.snapshot.Activity {
		controller.snapshot.Activity[i] = control.ActivityLogEntry{
			ID: uint64(i + 1), Time: base.Add(time.Duration(i) * time.Minute), Level: "info",
			Source: "desktop", Message: fmt.Sprintf("activity event %02d", i),
		}
	}
	tester := ui.NewTester(controller.View, 1320, 860)
	for i := 5; i < 15; i++ {
		if !tester.HasText(fmt.Sprintf("activity event %02d", i)) {
			t.Fatalf("recent activity omitted event %d; texts=%q", i, tester.Texts())
		}
	}
	for i := 0; i < 5; i++ {
		if tester.HasText(fmt.Sprintf("activity event %02d", i)) {
			t.Fatalf("recent activity kept old event %d; texts=%q", i, tester.Texts())
		}
	}
}

func TestOverviewPairsRelatedCardsByRow(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	tester := ui.NewTester(controller.View, 1320, 860)
	components, ok := tester.Find("订阅、策略与数据面状态")
	if !ok {
		t.Fatalf("overview components card missing; texts=%q", tester.Texts())
	}
	inventory, ok := tester.Find("当前控制面中已保存的对象")
	if !ok {
		t.Fatalf("inventory card missing; texts=%q", tester.Texts())
	}
	connections, ok := tester.Find("最近建立的真实连接")
	if !ok {
		t.Fatalf("connections card missing; texts=%q", tester.Texts())
	}
	activity, ok := tester.Find("来自控制面事件日志")
	if !ok {
		t.Fatalf("activity card missing; texts=%q", tester.Texts())
	}
	if components.Y != inventory.Y {
		t.Fatalf("components and inventory are not in the same row: %.1f vs %.1f", components.Y, inventory.Y)
	}
	if connections.Y != activity.Y {
		t.Fatalf("connections and activity are not in the same row: %.1f vs %.1f", connections.Y, activity.Y)
	}
	if connections.Y <= components.Y {
		t.Fatalf("connections/activity row did not move below components/inventory: %.1f <= %.1f", connections.Y, components.Y)
	}
}

func TestNavigationAndNarrowSidebar(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	tester := ui.NewTester(controller.View, 1000, 800)
	if !controller.narrow {
		t.Fatal("sidebar did not switch to its compact width")
	}
	if err := tester.Click("订阅"); err != nil {
		t.Fatalf("click subscriptions: %v", err)
	}
	if controller.page != pageSubscriptions || !tester.HasText("订阅管理") {
		t.Fatalf("page = %q, texts=%q", controller.page, tester.Texts())
	}
	tester.SetSize(1320, 860)
	tester.Frame()
	if controller.narrow {
		t.Fatal("sidebar did not expand with the window")
	}
}

func TestSettingsEmbedInboundsAndExposeListenPortEditor(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageSettings
	tester := ui.NewTester(controller.View, 1320, 1200)
	if !tester.HasText("入口管理") || !tester.HasText("[::]:1080") {
		t.Fatalf("settings page missing inbound section; texts=%q", tester.Texts())
	}
	if err := tester.Click("编辑入口 Mixed"); err != nil {
		t.Fatalf("open inbound editor: %v", err)
	}
	if !controller.editorOpen || controller.editor.kind != editorInbound || controller.editor.listenPort != "1080" {
		t.Fatalf("inbound editor = %+v, open=%v", controller.editor, controller.editorOpen)
	}
	tester.Frame()
	if !tester.HasText("监听端口") || !tester.HasText("监听地址") {
		t.Fatalf("inbound editor missing listen fields; texts=%q", tester.Texts())
	}
}

func TestNavigationOmitsStandaloneInboundsPage(t *testing.T) {
	for _, item := range navigation {
		if item.ID == "inbounds" {
			t.Fatalf("navigation still contains the standalone inbounds page: %+v", item)
		}
	}
}

func TestLongTableValueKeepsSingleLineHeight(t *testing.T) {
	longName := "Very long node name " + strings.Repeat("that must not expand the table row ", 4)
	controller := newViewTestController(ControllerOptions{})
	controller.snapshot.Catalog.Nodes[0].Name = longName
	controller.page = pageNodes
	tester := ui.NewTester(controller.View, 1320, 860)
	bounds, ok := tester.Find(longName)
	if !ok {
		t.Fatalf("long table value was not rendered; texts=%q", tester.Texts())
	}
	if bounds.H > 30 {
		t.Fatalf("long cell expanded to %.1f px", bounds.H)
	}
}

func TestNarrowTablesKeepRowActionsInsideViewport(t *testing.T) {
	tests := []struct {
		page   string
		action string
	}{
		{page: pageSubscriptions, action: "编辑订阅 Primary subscription"},
		{page: pageNodes, action: "编辑节点 Tokyo Edge"},
		{page: pageEgress, action: "编辑策略组 Auto Select"},
		{page: pageRoutes, action: "编辑规则 Private domains"},
		{page: pageRuleSets, action: "编辑规则集 geoip-cn"},
		{page: pageConnections, action: "关闭连接 api.github.com:443"},
	}
	for _, test := range tests {
		t.Run(test.page, func(t *testing.T) {
			controller := newViewTestController(ControllerOptions{})
			controller.page = test.page
			if test.page == pageConnections {
				controller.snapshot.Kernel.State = kernel.StateRunning
				controller.snapshot.Connections = []kernel.Connection{{
					ID: 1, Network: "tcp", Source: "127.0.0.1:51622", Target: "api.github.com:443",
					Outbound: "Auto Select", Rule: "GitHub", Process: "Code",
				}}
			}
			tester := ui.NewTester(controller.View, 960, 720)
			bounds, ok := tester.Find(test.action)
			if !ok {
				t.Fatalf("row action %q was not rendered; texts=%q", test.action, tester.Texts())
			}
			if right := bounds.X + bounds.W; right > 940 {
				t.Fatalf("row action %q ended at %.1fpx, outside the narrow content area", test.action, right)
			}
		})
	}
}

func TestThemeToggleNotifiesDesktopHost(t *testing.T) {
	var persisted bool
	controller := newViewTestController(ControllerOptions{OnThemeChange: func(dark bool) error {
		persisted = dark
		return nil
	}})
	tester := ui.NewTester(controller.View, 1320, 860)
	if err := tester.Click("切换深色模式"); err != nil {
		t.Fatalf("click theme toggle: %v", err)
	}
	if !controller.dark || !persisted {
		t.Fatalf("dark = %v, persisted = %v", controller.dark, persisted)
	}
	tester.Frame()
	if tester.HasText("切换深色模式") {
		t.Fatalf("dark mode rendered both theme controls; texts=%q", tester.Texts())
	}
	if !tester.HasText("切换浅色模式") {
		t.Fatalf("light-mode control missing; texts=%q", tester.Texts())
	}
}

func newViewTestController(options ControllerOptions) *Controller {
	controller := NewController(nil, options)
	controller.runtime = &Runtime{
		DataDir:      "/tmp/boxd-desktop-test",
		DatabasePath: "/tmp/boxd-desktop-test/boxd.sqlite",
		Version:      "0.1.0-test",
		Commit:       "test",
		BuiltAt:      "2026-11-25T00:00:00Z",
	}
	controller.snapshot = snapshot{
		Kernel: kernel.Status{State: kernel.StateStopped},
		Inbounds: []control.InboundProfile{
			{ID: "mixed", Name: "Mixed", Kind: control.InboundKindMixed, Enabled: true, Listen: control.ListenAddress{Host: "::", Port: 1080}, UDP: true},
			{ID: "tun", Name: "TUN", Kind: control.InboundKindTun, Enabled: false, Tun: control.TunOptions{Addresses: []string{"172.19.0.1/30"}, MTU: 9000, Stack: "mixed", AutoRoute: true}},
		},
		Scheduler: control.SchedulerStatus{
			Running:         true,
			IntervalSeconds: 60,
			NextRunAt:       time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC),
		},
		Subscriptions: []control.Subscription{{
			ID: "sub-1", Name: "Primary subscription", URL: "https://subscription.example/feed", Enabled: true, IntervalSeconds: 3600,
		}},
		Catalog: control.EgressCatalog{
			Nodes: []control.Node{{
				ID: "node-1", Name: "Tokyo Edge", Protocol: "vless", Server: "edge.example", Port: 443,
			}},
			Groups: []control.Group{{
				ID: "group-1", Name: "Auto Select", Kind: control.GroupKindSelector, Members: []string{"node-1"}, Selected: "node-1", IntervalSeconds: 300,
			}},
			Chains: []control.ProxyChain{{
				ID: "chain-1", Name: "Two hop", Hops: []control.ProxyHop{{Kind: control.EgressKindNode, ID: "node-1"}},
			}},
		},
		Routes: control.RouteProfile{
			Mode:          control.RouteModeRules,
			DefaultEgress: control.EgressRef{Kind: control.EgressKindDirect},
			Rules: []control.RouteRule{{
				ID: "rule-1", Name: "Private domains", Action: control.RouteActionDirect,
				Match: control.RouteMatch{DomainSuffixes: []string{"internal.example"}},
			}},
		},
		DNS: control.DNSProfile{
			Strategy: "prefer_ipv4",
			Cache:    true,
			Domestic: control.DNSServerConfig{Enabled: true, Protocol: control.DNSProtocolDoH, Address: "https://dns.example/dns-query", Detour: control.DNSDetourDirect},
			Foreign:  control.DNSServerConfig{Enabled: true, Protocol: control.DNSProtocolDoH, Address: "https://1.1.1.1/dns-query", Detour: control.DNSDetourRouteDefault},
			Default:  control.DNSServerConfig{Enabled: true, Protocol: control.DNSProtocolDoH, Detour: control.DNSDetourRouteDefault, FollowForeign: true},
		},
		RuleSets: []control.RuleSet{{
			Name: "geoip-cn", URL: "https://rules.example/geoip-cn.srs", Format: control.RuleSetFormatBinary, Enabled: true, Builtin: true,
		}},
		Activity: []control.ActivityLogEntry{{
			ID: 1, Time: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC), Level: "info", Source: "desktop", Message: "control plane ready",
		}},
	}
	controller.snapshot.SubscriptionsByID = map[string]control.Subscription{}
	for _, subscription := range controller.snapshot.Subscriptions {
		controller.snapshot.SubscriptionsByID[subscription.ID] = subscription
	}
	controller.snapshot.NodesByID = map[string]control.Node{}
	for _, node := range controller.snapshot.Catalog.Nodes {
		controller.snapshot.NodesByID[node.ID] = node
	}
	controller.snapshot.GroupsByID = map[string]control.Group{}
	for _, group := range controller.snapshot.Catalog.Groups {
		controller.snapshot.GroupsByID[group.ID] = group
	}
	controller.snapshot.ChainsByID = map[string]control.ProxyChain{}
	for _, chain := range controller.snapshot.Catalog.Chains {
		controller.snapshot.ChainsByID[chain.ID] = chain
	}
	controller.snapshot.SettingsByKey = map[string]string{}
	return controller
}
