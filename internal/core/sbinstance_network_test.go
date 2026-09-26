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
)

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
	gate, err = newNetworkStartupGate(monitor)
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	gate.callback = func(*control.Interface, int) { calls++ }
	gate.notify(new(control.Interface), 0)
	gate.close()
	gate.start()
	if calls != 0 {
		t.Fatal("startup failure must discard pending callbacks")
	}
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
