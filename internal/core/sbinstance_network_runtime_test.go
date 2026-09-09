package core

import (
	"context"
	"io"
	"net"
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

func TestRealBoxInitialNetworkDoesNotResetFirstConnection(t *testing.T) {
	instance, probe := newNativeStartupProbe(t)
	if err := instance.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-probe.reset:
		t.Fatal("initial interface notification reset a newly started outbound")
	case <-time.After(20 * time.Millisecond):
	}
	if err := probe.client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := probe.server.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	writeResult := make(chan error, 1)
	go func() {
		_, err := probe.client.Write([]byte("ready"))
		writeResult <- err
	}()
	buffer := make([]byte, len("ready"))
	if _, err := io.ReadFull(probe.server, buffer); err != nil {
		t.Fatalf("first connection was reset: %v", err)
	}
	if err := <-writeResult; err != nil {
		t.Fatal(err)
	}
	instance.gate.notify(new(control.Interface), 0)
	select {
	case <-probe.reset:
	case <-time.After(time.Second):
		t.Fatal("runtime network changes no longer reach native outbounds")
	}
}

func newNativeStartupProbe(t *testing.T) (realBox, *startupProbeOutbound) {
	t.Helper()
	ctx := include.Context(context.Background())
	probe := &startupProbeOutbound{fakeOutbound: fakeOutbound{tag: "first"}, reset: make(chan struct{}, 1)}
	registry := service.FromContext[adapter.OutboundRegistry](ctx).(*outbound.Registry)
	outbound.Register[option.StubOptions](registry, "startup-probe", func(
		context.Context, adapter.Router, log.ContextLogger, string, option.StubOptions,
	) (adapter.Outbound, error) {
		return probe, nil
	})
	instance, err := newRealBox(box.Options{
		Context: ctx,
		Options: option.Options{
			Log: &option.LogOptions{Disabled: true},
			Outbounds: []option.Outbound{{
				Type: "startup-probe", Tag: "first", Options: &option.StubOptions{},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = instance.Close() })
	return instance, probe
}

type startupProbeOutbound struct {
	fakeOutbound
	client net.Conn
	server net.Conn
	reset  chan struct{}
}

func (o *startupProbeOutbound) Start(stage adapter.StartStage) error {
	if stage == adapter.StartStateStarted {
		o.client, o.server = net.Pipe()
	}
	return nil
}

func (o *startupProbeOutbound) Close() error {
	if o.client != nil {
		_ = o.client.Close()
		_ = o.server.Close()
	}
	return nil
}

func (o *startupProbeOutbound) InterfaceUpdated(context.Context) {
	if o.client != nil {
		_ = o.client.Close()
	}
	select {
	case o.reset <- struct{}{}:
	default:
	}
}
