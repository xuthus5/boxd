package core

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	tun "github.com/sagernet/sing-tun"
	M "github.com/sagernet/sing/common/metadata"
)

func TestTrafficTrackerRoutedFlowAccountsAndCloses(t *testing.T) {
	tracker := NewTrafficTracker()
	flow := tracker.RoutedFlow(context.Background(), adapter.InboundContext{
		Network: "icmp", Inbound: "tun-in", Source: M.ParseSocksaddr("10.0.0.2"),
		Destination: M.ParseSocksaddr("203.0.113.5"),
	}, nil, fakeOutbound{tag: "bridge"})
	if len(tracker.Connections()) != 0 {
		t.Fatal("unattached flow must not expose a connection that cannot be closed")
	}
	handle := &fakeTrafficFlowHandle{onClose: func() { flow.CloseFlow(tun.FlowCloseReset) }}
	flow.AttachFlow(handle)
	flow.FlowEstablished()
	flow.CountForward(100)
	flow.CountForward(-1)
	flow.CountReverse(40)
	flow.CountReverse(0)
	connections := tracker.Connections()
	if len(connections) != 1 {
		t.Fatalf("connections=%v", connections)
	}
	connection := connections[0]
	if connection.Network != "icmp" || connection.Inbound != "tun-in" || connection.Outbound != "bridge" {
		t.Fatalf("flow metadata was lost: %+v", connection)
	}
	if connection.Upload != 100 || connection.Download != 40 {
		t.Fatalf("flow traffic=%+v", connection)
	}
	up, down := tracker.Total()
	if up != 100 || down != 40 {
		t.Fatalf("total up=%d down=%d", up, down)
	}
	if !tracker.CloseConn(connection.ID) || handle.calls.Load() != 1 || len(tracker.Connections()) != 0 {
		t.Fatal("manual close must reach the native flow and remove it")
	}
}

func TestTrafficFlowNaturalCloseAndLateAttach(t *testing.T) {
	for _, reason := range []tun.FlowCloseReason{tun.FlowCloseReset, tun.FlowCloseFinished, tun.FlowCloseTimeout} {
		t.Run(reason.String(), func(t *testing.T) {
			tracker := NewTrafficTracker()
			flow := tracker.RoutedFlow(context.Background(), adapter.InboundContext{Network: "udp"}, nil, fakeOutbound{tag: "bridge"})
			flow.AttachFlow(nil)
			flow.CloseFlow(reason)
			handle := &fakeTrafficFlowHandle{}
			flow.AttachFlow(handle)
			if handle.calls.Load() != 1 || len(tracker.Connections()) != 0 {
				t.Fatal("attaching to a closed flow must close the late handle")
			}
		})
	}
	tracker := NewTrafficTracker()
	flow := tracker.RoutedFlow(context.Background(), adapter.InboundContext{}, nil, fakeOutbound{tag: "bridge"})
	handle := &fakeTrafficFlowHandle{}
	flow.AttachFlow(handle)
	flow.CloseFlow(tun.FlowCloseFinished)
	flow.CloseFlow(tun.FlowCloseFinished)
	if len(tracker.Connections()) != 0 || handle.calls.Load() != 0 {
		t.Fatal("natural close should only remove accounting state")
	}
}

func TestTrafficFlowCloseBeforeAttachAndConcurrentEvents(t *testing.T) {
	tracker := NewTrafficTracker()
	flow := tracker.RoutedFlow(context.Background(), adapter.InboundContext{}, nil, fakeOutbound{tag: "bridge"}).(*trafficFlow)
	if err := flow.close(); err != nil {
		t.Fatal(err)
	}
	handle := &fakeTrafficFlowHandle{}
	flow.AttachFlow(handle)
	if handle.calls.Load() != 1 {
		t.Fatal("a close requested before attachment must not be lost")
	}
	flow = tracker.RoutedFlow(context.Background(), adapter.InboundContext{}, nil, fakeOutbound{tag: "bridge"}).(*trafficFlow)
	var workers sync.WaitGroup
	workers.Go(func() { flow.AttachFlow(&fakeTrafficFlowHandle{}) })
	for range 16 {
		workers.Go(func() {
			flow.CountForward(10)
			flow.CountReverse(5)
			flow.FlowEstablished()
			flow.CloseFlow(tun.FlowCloseTimeout)
		})
	}
	workers.Wait()
	if err := flow.close(); err != nil {
		t.Fatal(err)
	}
	up, down := tracker.Total()
	if up != 160 || down != 80 || len(tracker.Connections()) != 0 {
		t.Fatalf("concurrent flow accounting: up=%d down=%d connections=%v", up, down, tracker.Connections())
	}
}

type fakeTrafficFlowHandle struct {
	calls   atomic.Int32
	onClose func()
}

func (h *fakeTrafficFlowHandle) CloseFlow() {
	h.calls.Add(1)
	if h.onClose != nil {
		h.onClose()
	}
}
