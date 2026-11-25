package mygo

import (
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

func TestInboundPageExposesListenPortEditor(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageInbounds
	tester := ui.NewTester(controller.View, 1320, 860)
	if !tester.HasText("本地入口") || !tester.HasText("[::]:1080") {
		t.Fatalf("inbound page missing listener summary; texts=%q", tester.Texts())
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
