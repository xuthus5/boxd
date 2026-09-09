package core

import (
	"context"
	"errors"
	"testing"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
)

func TestNetworkStartupStateInitializesWithoutReset(t *testing.T) {
	ctx := pause.WithDefaultManager(context.Background())
	manager := service.FromContext[pause.Manager](ctx)
	network := &fakeNetworkStartupUpdater{}
	state := networkStartupState{ctx: ctx, manager: manager, network: network, logger: log.NewNOPFactory().Logger()}
	state.apply(nil, 0)
	if !manager.IsNetworkPaused() || network.interfaces != 0 || network.wifi != 0 {
		t.Fatal("missing route must pause network activity without resetting connections")
	}
	state.apply(new(control.Interface), 0)
	if manager.IsNetworkPaused() || network.interfaces != 1 || network.wifi != 1 || network.ctx != ctx {
		t.Fatal("initial interface must publish current network and WIFI state")
	}
	network.err = errors.New("interface enumeration failed")
	state.apply(new(control.Interface), 0)
	if network.interfaces != 2 || manager.IsNetworkPaused() {
		t.Fatal("interface refresh failure should retain the detected default route")
	}
}

func TestNetworkStartupGateSeparatesInitializationFromRuntimeUpdates(t *testing.T) {
	var initialized, runtimeUpdates int
	gate := &networkStartupGate{
		initialize: func(*control.Interface, int) { initialized++ },
		callback:   func(*control.Interface, int) { runtimeUpdates++ },
	}
	gate.notify(nil, 0)
	gate.start()
	if initialized != 1 || runtimeUpdates != 0 {
		t.Fatal("startup must initialize state without invoking the asynchronous network reset")
	}
	gate.notify(nil, 0)
	if initialized != 1 || runtimeUpdates != 1 {
		t.Fatal("subsequent updates must reach the native network callback")
	}
}

func TestNetworkStateRuntimeUpdateRespectsCancellation(t *testing.T) {
	for _, stage := range []string{"before", "wifi", "interfaces", "active"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(pause.WithDefaultManager(context.Background()))
			defer cancel()
			network := &fakeNetworkStartupUpdater{}
			switch stage {
			case "before":
				cancel()
			case "wifi":
				network.onWIFI = cancel
			case "interfaces":
				network.onUpdate = cancel
			}
			state := networkStartupState{
				ctx: ctx, network: network, manager: service.FromContext[pause.Manager](ctx),
				logger: log.NewNOPFactory().Logger(),
			}
			state.update(ctx, new(control.Interface), 0)
			if stage == "active" && network.resets != 1 || stage != "active" && network.resets != 0 {
				t.Fatalf("canceled update reset connections: stage=%s resets=%d", stage, network.resets)
			}
			if stage == "wifi" && network.interfaces != 0 {
				t.Fatal("canceled WIFI refresh continued updating the environment")
			}
		})
	}
}

func TestNetworkStartupGatePausesOfflineBeforeReady(t *testing.T) {
	ctx := pause.WithDefaultManager(context.Background())
	manager := service.FromContext[pause.Manager](ctx)
	network := &fakeNetworkStartupUpdater{}
	state := networkStartupState{ctx: ctx, manager: manager, network: network, logger: log.NewNOPFactory().Logger()}
	gate := &networkStartupGate{initialize: state.apply, callback: state.apply}
	gate.notify(new(control.Interface), 0)
	gate.notify(nil, 0)
	if !manager.IsNetworkPaused() {
		t.Fatal("startup must pause network consumers immediately when no route exists")
	}
	gate.start()
	if !manager.IsNetworkPaused() || network.interfaces != 0 {
		t.Fatal("a stale startup interface must not override a newer missing-route event")
	}
	gate.notify(new(control.Interface), 0)
	if manager.IsNetworkPaused() || network.interfaces != 1 {
		t.Fatal("a later interface must restore network activity")
	}
}

type fakeNetworkStartupUpdater struct {
	interfaces int
	wifi       int
	ctx        context.Context
	err        error
	resets     int
	onWIFI     func()
	onUpdate   func()
}

func (u *fakeNetworkStartupUpdater) UpdateInterfaces() error {
	u.interfaces++
	if u.onUpdate != nil {
		u.onUpdate()
	}
	return u.err
}

func (u *fakeNetworkStartupUpdater) UpdateWIFIState(ctx context.Context) {
	u.wifi++
	u.ctx = ctx
	if u.onWIFI != nil {
		u.onWIFI()
	}
}

func (u *fakeNetworkStartupUpdater) ResetNetwork(context.Context) { u.resets++ }
