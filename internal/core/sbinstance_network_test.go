package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sagernet/sing-box/log"
	tun "github.com/sagernet/sing-tun"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/common/x/list"
)

func TestNetworkStartupGateDefersOnlyTheKernelCallback(t *testing.T) {
	var callbacks list.List[tun.DefaultInterfaceUpdateCallback]
	var gotInterface *control.Interface
	var gotFlags, calls, otherCalls int
	kernel := callbacks.PushBack(func(value *control.Interface, flags int) {
		gotInterface, gotFlags = value, flags
		calls++
	})
	other := callbacks.PushBack(func(*control.Interface, int) { otherCalls++ })
	marker := callbacks.PushBack(nil)
	gate := &networkStartupGate{}
	if err := gate.attach(marker); err != nil {
		t.Fatal(err)
	}
	first := new(control.Interface)
	first.Name = "first"
	latest := new(control.Interface)
	latest.Name = "latest"
	kernel.Value(first, 1)
	kernel.Value(latest, 2)
	other.Value(first, 1)
	if calls != 0 || otherCalls != 1 || kernel.Next() != other {
		t.Fatalf("premature kernel calls=%d, other calls=%d", calls, otherCalls)
	}
	gate.start()
	gate.start()
	if calls != 1 || gotInterface != latest || gotFlags != 2 {
		t.Fatalf("pending event was not preserved: calls=%d interface=%v flags=%d", calls, gotInterface, gotFlags)
	}
	kernel.Value(first, 3)
	if calls != 2 || gotInterface != first || gotFlags != 3 {
		t.Fatal("running interface updates must pass through")
	}
	gate.close()
	gate.start()
	kernel.Value(latest, 4)
	if calls != 2 {
		t.Fatal("closed gate dispatched a callback")
	}
}

func TestNetworkStartupGateChecksCallbackLayout(t *testing.T) {
	for _, name := range []string{"missing kernel callback", "empty callback", "marker not last"} {
		t.Run(name, func(t *testing.T) {
			var callbacks list.List[tun.DefaultInterfaceUpdateCallback]
			if name != "missing kernel callback" {
				callbacks.PushBack(nil)
			}
			marker := callbacks.PushBack(nil)
			if name == "marker not last" {
				callbacks.PushBack(nil)
			}
			gate := &networkStartupGate{}
			if err := gate.attach(marker); !errors.Is(err, errNetworkCallbackLayout) {
				t.Fatalf("expected callback layout error, got %v", err)
			}
		})
	}
}

func TestNetworkStartupGateChecksNativeMonitor(t *testing.T) {
	gate, err := newNetworkStartupGate(nil)
	if err != nil {
		t.Fatal(err)
	}
	gate.start()
	gate.close()
	if _, err := newNetworkStartupGate(&unexpectedNetworkMonitor{}); !errors.Is(err, errNetworkCallbackLayout) {
		t.Fatalf("expected unknown monitor error, got %v", err)
	}
	monitor := newUnstartedNativeMonitor(t)
	if _, err := newNetworkStartupGate(monitor); !errors.Is(err, errNetworkCallbackLayout) {
		t.Fatalf("expected missing native callback error, got %v", err)
	}
	var calls int
	token := monitor.RegisterCallback(func(*control.Interface, int) { calls++ })
	gate, err = newNetworkStartupGate(monitor)
	if err != nil {
		t.Fatal(err)
	}
	if token.Next() != nil {
		t.Fatal("bootstrap marker must be removed")
	}
	token.Value(nil, 0)
	gate.close()
	gate.start()
	if calls != 0 {
		t.Fatal("startup failure must discard pending callbacks")
	}
	monitor.UnregisterCallback(token)
}

type unexpectedNetworkMonitor struct {
	tun.DefaultInterfaceMonitor
}

func newUnstartedNativeMonitor(t *testing.T) tun.DefaultInterfaceMonitor {
	t.Helper()
	logger := log.NewNOPFactory().Logger()
	network, err := tun.NewNetworkUpdateMonitor(logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := network.Close(); err != nil {
			t.Error(err)
		}
	})
	monitor, err := tun.NewDefaultInterfaceMonitor(network, logger, tun.DefaultInterfaceMonitorOptions{
		InterfaceFinder: control.NewDefaultInterfaceFinder(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return monitor
}

func TestNetworkStartupGateCloseWaitsForInFlightCallback(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var completed atomic.Bool
	gate := &networkStartupGate{callback: func(*control.Interface, int) {
		close(entered)
		<-release
		completed.Store(true)
	}}
	gate.start()
	go gate.notify(nil, 0)
	<-entered
	closed := make(chan bool, 1)
	go func() {
		gate.close()
		closed <- completed.Load()
	}()
	select {
	case <-closed:
		close(release)
		t.Fatal("close returned while a callback was still running")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case afterCallback := <-closed:
		if !afterCallback {
			t.Fatal("close must wait for in-flight dispatch")
		}
	case <-time.After(time.Second):
		t.Fatal("close did not complete")
	}
}

func TestNetworkStartupGateConcurrentLifecycle(t *testing.T) {
	var calls atomic.Int32
	gate := &networkStartupGate{callback: func(*control.Interface, int) { calls.Add(1) }}
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			gate.notify(nil, 0)
			gate.start()
			gate.notify(nil, 1)
			gate.close()
		})
	}
	workers.Wait()
	count := calls.Load()
	gate.notify(nil, 2)
	if calls.Load() != count {
		t.Fatal("post-close callback was dispatched")
	}
}
