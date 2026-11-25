package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

func TestSamplingOutlivesInitializationContext(t *testing.T) {
	service := NewObservabilityService(nil, nil)
	initCtx, cancelInit := context.WithCancel(context.Background())
	cancelInit()
	if err := service.StartSampling(initCtx); err != nil {
		t.Fatalf("StartSampling() error = %v", err)
	}
	t.Cleanup(service.StopSampling)

	time.Sleep(25 * time.Millisecond)
	service.mu.Lock()
	done := service.done
	service.mu.Unlock()
	if done == nil {
		t.Fatal("sampling was not started")
	}
	select {
	case <-done:
		t.Fatal("sampling stopped with the short-lived initialization context")
	default:
	}

	service.StopSampling()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sampling did not stop after StopSampling")
	}
}

func TestLiveObservationAggregatesOnlyRealConnectionDeltas(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	status := kernel.Status{State: kernel.StateRunning, StartedAt: now.Add(-time.Minute), Revision: "rev-1"}
	inbounds := []control.InboundProfile{{ID: "mixed", Name: "Mixed", Kind: control.InboundKindMixed, Enabled: true}}
	nodes := []control.Node{{ID: "hk", Name: "Hong Kong"}}
	routes := control.RouteProfile{
		Mode:          control.RouteModeRules,
		DefaultEgress: control.EgressRef{Kind: control.EgressKindDirect},
		Rules: []control.RouteRule{{
			ID: "rule-hk", Name: "Hong Kong", Action: control.RouteActionDirect,
		}},
	}
	observation := newLiveObservation()
	connections := []kernel.Connection{{
		ID: 1, Inbound: "mixed/inbound-mixed", Outbound: control.NodeTag("hk"),
		RuleIndex: 1, Upload: 100, Download: 200,
	}}

	first := observation.update(status, connections, nil, routes, inbounds, nodes, nil, now)
	if !first.Available {
		t.Fatal("running observation must be available")
	}
	if first.Nodes["hk"].Total() != 300 || first.Inbounds["mixed"].Total() != 300 {
		t.Fatalf("first totals = nodes %+v inbounds %+v", first.Nodes["hk"], first.Inbounds["mixed"])
	}
	if first.RuleHits["rule-hk"] != 1 {
		t.Fatalf("first rule hits = %+v", first.RuleHits)
	}

	connections[0].Upload = 175
	connections[0].Download = 275
	second := observation.update(status, connections, nil, routes, inbounds, nodes, nil, now.Add(2*time.Second))
	if second.Nodes["hk"].Total() != 450 || second.Inbounds["mixed"].Total() != 450 {
		t.Fatalf("second totals = nodes %+v inbounds %+v", second.Nodes["hk"], second.Inbounds["mixed"])
	}
	if second.RuleHits["rule-hk"] != 1 {
		t.Fatalf("second rule hits = %+v", second.RuleHits)
	}
	if second.Nodes["hk"].TotalRate() != 74 || second.Inbounds["mixed"].TotalRate() != 74 {
		t.Fatalf("second rates = nodes %+v inbounds %+v", second.Nodes["hk"], second.Inbounds["mixed"])
	}

	connections = append(connections, kernel.Connection{
		ID: 2, Inbound: "mixed/inbound-mixed", Outbound: control.NodeTag("hk"),
		RuleIndex: 1, Upload: 10, Download: 20,
	})
	third := observation.update(status, connections, nil, routes, inbounds, nodes, nil, now.Add(3*time.Second))
	if third.Nodes["hk"].Total() != 480 || third.RuleHits["rule-hk"] != 2 {
		t.Fatalf("third observation = %+v", third)
	}
}

func TestLiveObservationResetsWhenKernelStopsOrRestarts(t *testing.T) {
	now := time.Now().UTC()
	observation := newLiveObservation()
	routes := control.RouteProfile{Mode: control.RouteModeRules}
	status := kernel.Status{State: kernel.StateRunning, StartedAt: now, Revision: "rev-1"}
	connection := kernel.Connection{ID: 1, Upload: 20, Download: 40}
	_ = observation.update(status, []kernel.Connection{connection}, nil, routes, nil, nil, nil, now)
	if stopped := observation.update(kernel.Status{State: kernel.StateStopped}, nil, nil, routes, nil, nil, nil, now); stopped.Available {
		t.Fatal("stopped kernel must not expose observation data")
	}
	restarted := observation.update(kernel.Status{State: kernel.StateRunning, StartedAt: now.Add(time.Second), Revision: "rev-2"}, []kernel.Connection{connection}, nil, routes, nil, nil, nil, now.Add(time.Second))
	if !restarted.Available || restarted.Nodes != nil && len(restarted.Nodes) != 0 {
		t.Fatalf("restarted observation = %+v", restarted)
	}
}

func TestLiveObservationIncludesConnectionsClosedBetweenSamples(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	status := kernel.Status{State: kernel.StateRunning, StartedAt: now.Add(-time.Minute), Revision: "rev-1"}
	inbounds := []control.InboundProfile{{ID: "mixed", Name: "Mixed", Kind: control.InboundKindMixed, Enabled: true}}
	nodes := []control.Node{{ID: "hk", Name: "Hong Kong"}}
	routes := control.RouteProfile{Mode: control.RouteModeRules, Rules: []control.RouteRule{{ID: "rule-hk"}}}
	closed := []kernel.Connection{{
		ID: 7, Inbound: "mixed/inbound-mixed", Outbound: control.NodeTag("hk"),
		RuleIndex: 1, Upload: 80, Download: 120,
	}}

	observation := newLiveObservation()
	first := observation.update(status, nil, closed, routes, inbounds, nodes, nil, now)
	if first.Nodes["hk"].Total() != 200 || first.Inbounds["mixed"].Total() != 200 {
		t.Fatalf("closed connection totals = nodes %+v inbounds %+v", first.Nodes["hk"], first.Inbounds["mixed"])
	}
	if first.RuleHits["rule-hk"] != 1 {
		t.Fatalf("closed connection rule hits = %+v", first.RuleHits)
	}
	second := observation.update(status, nil, closed, routes, inbounds, nodes, nil, now.Add(2*time.Second))
	if second.Nodes["hk"].Total() != 200 || second.RuleHits["rule-hk"] != 1 {
		t.Fatalf("repeated closed snapshot double-counted data: %+v", second)
	}
}

func TestCompiledRouteRuleIDsMirrorCompilerOrder(t *testing.T) {
	routes := control.RouteProfile{
		Mode: control.RouteModeRules,
		Presets: control.RoutePresets{
			BypassLAN: true, ChinaDirect: true, AdBlock: true,
		},
		Rules: []control.RouteRule{
			{ID: "disabled", Disabled: true},
			{ID: "first"},
			{ID: "second"},
		},
	}
	inbounds := []control.InboundProfile{
		{ID: "mixed", Enabled: true, Egress: &control.EgressRef{Kind: control.EgressKindDirect}},
		{ID: "tun", Enabled: false, Egress: &control.EgressRef{Kind: control.EgressKindDirect}},
	}
	indexes := compiledRouteRuleIDs(routes, inbounds)
	if indexes[2] != "" {
		t.Fatalf("index 2 = %q, want no user rule", indexes[2])
	}
	if indexes[5] != "first" || indexes[6] != "second" || len(indexes) != 2 {
		t.Fatalf("compiled rule indexes = %+v", indexes)
	}
}
