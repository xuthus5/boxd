package httpapi

import (
	"testing"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

func TestMakeTrafficViewReportsAvailability(t *testing.T) {
	now := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	unavailable := makeTrafficView(kernel.TrafficSnapshot{}, now, false)
	if unavailable.Available {
		t.Fatal("unavailable traffic must not be marked as available")
	}
	if unavailable.UploadRate != 0 || unavailable.DownloadRate != 0 {
		t.Fatalf("unavailable traffic = %+v", unavailable)
	}

	available := makeTrafficView(kernel.TrafficSnapshot{
		UploadTotal: 10, DownloadTotal: 20, UploadRate: 1, DownloadRate: 2,
		At: now,
	}, now, true)
	if !available.Available || available.UploadTotal != 10 || available.DownloadRate != 2 {
		t.Fatalf("available traffic = %+v", available)
	}
}

func TestOverviewDoesNotInventTrafficWhenKernelIsStopped(t *testing.T) {
	view := makeOverview(snapshotView{
		Traffic: trafficView{Available: false},
		Routes:  routeView{Mode: control.RouteModeRules, DefaultTarget: "直连"},
	}, time.Now())
	if len(view.Stats) != 4 {
		t.Fatalf("stats = %+v", view.Stats)
	}
	if view.Stats[0].Value != "—" || view.Stats[0].Delta != "内核未运行" {
		t.Fatalf("connection stat = %+v", view.Stats[0])
	}
	if view.Stats[1].Value != "未采集" || view.Stats[1].Delta != "内核未运行" {
		t.Fatalf("traffic stat = %+v", view.Stats[1])
	}
}

func TestNodeAndInboundViewsUseObservedTraffic(t *testing.T) {
	inbounds := []control.InboundProfile{{ID: "mixed", Name: "Mixed", Kind: control.InboundKindMixed, Enabled: true}}
	nodes := []control.Node{{ID: "hk", Name: "Hong Kong"}}
	observation := observationSnapshot{
		Available: true,
		Nodes:     map[string]observedTraffic{"hk": {UploadTotal: 1024, DownloadTotal: 2048, UploadRate: 100, DownloadRate: 200}},
		Inbounds:  map[string]observedTraffic{"mixed": {UploadTotal: 1024, DownloadTotal: 2048, UploadRate: 100, DownloadRate: 200}},
	}
	inboundViews := makeInboundViewsWithObservation(inbounds, nil, observation)
	if inboundViews[0].Traffic != "3.0 KB" || inboundViews[0].Rate != "300 B/s" || !inboundViews[0].Observed {
		t.Fatalf("inbound view = %+v", inboundViews[0])
	}
	nodeViews := makeNodeViewsWithObservation(nodes, nil, nil, observation)
	if nodeViews[0].Traffic != "3.0 KB" || nodeViews[0].Rate != "300 B/s" || nodeViews[0].TrafficBytes != 3072 {
		t.Fatalf("node view = %+v", nodeViews[0])
	}
	stopped := makeNodeViewsWithObservation(nodes, nil, nil, observationSnapshot{})
	if stopped[0].Traffic != "未采集" || stopped[0].Observed {
		t.Fatalf("stopped node view = %+v", stopped[0])
	}
}

func TestRouteViewsReportObservedHitsTruthfully(t *testing.T) {
	routes := control.RouteProfile{
		Mode:  control.RouteModeRules,
		Rules: []control.RouteRule{{ID: "rule-1", Name: "One", Action: control.RouteActionDirect}},
	}
	view := makeRouteViewsWithObservation(routes, control.EgressCatalog{}, observationSnapshot{Available: true, RuleHits: map[string]uint64{"rule-1": 3}})
	if view.Rules[0].Hits != "3 次" {
		t.Fatalf("hits = %q", view.Rules[0].Hits)
	}
	stopped := makeRouteViews(routes, control.EgressCatalog{})
	if stopped.Rules[0].Hits != "未统计" {
		t.Fatalf("stopped hits = %q", stopped.Rules[0].Hits)
	}
	routes.Mode = control.RouteModeGlobal
	global := makeRouteViewsWithObservation(routes, control.EgressCatalog{}, observationSnapshot{Available: true})
	if global.Rules[0].Hits != "未生效" {
		t.Fatalf("global hits = %q", global.Rules[0].Hits)
	}
}

func TestRuleSetViewUsesDownloadedSize(t *testing.T) {
	now := time.Now().UTC()
	view := makeRuleSetViews([]control.RuleSet{{
		Name: "custom", Format: control.RuleSetFormatBinary, Enabled: true,
		Status: control.RuleSetStatus{SizeBytes: 1536, SHA256: "abcdef0123456789", UpdatedAt: now},
	}}, now)[0]
	if view.Size != "1.5 KB" {
		t.Fatalf("size = %q", view.Size)
	}
	view = makeRuleSetViews([]control.RuleSet{{Name: "custom"}}, now)[0]
	if view.Size != "未记录" {
		t.Fatalf("unknown size = %q", view.Size)
	}
}
