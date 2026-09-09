package core

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/service"
)

func TestRealBoxCloseWaitsForNetworkWorker(t *testing.T) {
	probe := &blockingNetworkOutbound{
		fakeOutbound: fakeOutbound{tag: "blocking"}, entered: make(chan struct{}),
		release: make(chan struct{}), completed: make(chan struct{}),
	}
	instance := newBlockingNetworkRuntime(t, probe)
	release := sync.OnceFunc(func() { close(probe.release) })
	t.Cleanup(release)
	if err := instance.Start(); err != nil {
		t.Fatal(err)
	}
	instance.gate.notify(new(control.Interface), 0)
	select {
	case <-probe.entered:
	case <-time.After(time.Second):
		t.Fatal("native interface update worker did not run")
	}
	closed := make(chan error, 1)
	go func() { closed <- instance.Close() }()
	select {
	case err := <-closed:
		release()
		t.Errorf("Close returned before native worker completed: %v", err)
	case <-time.After(20 * time.Millisecond):
		release()
		if err := <-closed; err != nil {
			t.Error(err)
		}
	}
	select {
	case <-probe.completed:
	case <-time.After(time.Second):
		t.Fatal("native interface update worker did not finish")
	}
}

type blockingNetworkOutbound struct {
	fakeOutbound
	entered   chan struct{}
	release   chan struct{}
	completed chan struct{}
}

func (o *blockingNetworkOutbound) InterfaceUpdated(context.Context) {
	close(o.entered)
	<-o.release
	close(o.completed)
}

func newBlockingNetworkRuntime(t *testing.T, probe adapter.Outbound) realBox {
	t.Helper()
	ctx := include.Context(context.Background())
	registry := service.FromContext[adapter.OutboundRegistry](ctx).(*outbound.Registry)
	outbound.Register[option.StubOptions](registry, "blocking-test", func(
		context.Context, adapter.Router, log.ContextLogger, string, option.StubOptions,
	) (adapter.Outbound, error) {
		return probe, nil
	})
	instance, err := newRealBox(box.Options{
		Context: ctx,
		Options: option.Options{Log: &option.LogOptions{Disabled: true}, Outbounds: []option.Outbound{{
			Type: "blocking-test", Tag: probe.Tag(), Options: &option.StubOptions{},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Error(err)
		}
	})
	return instance
}

func TestNetworkUpdateWorkerKeepsLatestNotification(t *testing.T) {
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	observed := make(chan string, 2)
	worker := newNetworkUpdateWorker(context.Background(), func(ctx context.Context, value *control.Interface, _ int) {
		if value.Name == "first" {
			close(entered)
			<-ctx.Done()
			close(canceled)
			<-release
			return
		}
		observed <- value.Name
	})
	t.Cleanup(worker.close)
	releaseFirst := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseFirst)
	first, second, latest := new(control.Interface), new(control.Interface), new(control.Interface)
	first.Name, second.Name, latest.Name = "first", "second", "latest"
	worker.notify(first, 0)
	waitNetworkSignal(t, entered)
	worker.notify(second, 0)
	worker.notify(latest, 0)
	waitNetworkSignal(t, canceled)
	releaseFirst()
	select {
	case name := <-observed:
		if name != "latest" {
			t.Fatalf("latest notification was lost: %s", name)
		}
	case <-time.After(time.Second):
		t.Fatal("queued network update did not run")
	}
	worker.close()
	worker.notify(first, 0)
	if ctx, _, _ := worker.takePending(); ctx != nil {
		t.Fatal("closed worker accepted a notification")
	}
}

func TestNetworkUpdateWorkerStopsWithParent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker := newNetworkUpdateWorker(ctx, func(context.Context, *control.Interface, int) {
		t.Error("canceled worker dispatched a notification")
	})
	cancel()
	worker.notify(nil, 0)
	waitNetworkSignal(t, worker.done)
	worker.close()
}

func waitNetworkSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("network worker did not reach the expected state")
	}
}
