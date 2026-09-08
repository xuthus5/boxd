package core

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
)

func TestRouteDefaultsRuntimeHijacksTCPDNS(t *testing.T) {
	cases := []struct {
		name string
		port uint16
	}{
		{name: "standard port before sniff", port: 53},
		{name: "nonstandard port after sniff", port: 5353},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runtime := newPolicyRuntime(t, policyDefaultsFixture())
			response := policyTCPDNSExchange(t, runtime, test.port)
			if len(response.Answer) != 1 || runtime.remote.calls.Load() != 1 || runtime.direct.calls.Load() != 0 {
				t.Fatalf("hijacked DNS response=%v, remote=%d, direct=%d", response.Answer,
					runtime.remote.calls.Load(), runtime.direct.calls.Load())
			}
			if runtime.trace.outbound != "" {
				t.Fatalf("DNS connection escaped to outbound %q", runtime.trace.outbound)
			}
		})
	}
}

func policyTCPDNSExchange(t *testing.T, runtime *policyRuntime, port uint16) *mdns.Msg {
	t.Helper()
	ctx, cancel := context.WithTimeout(runtime.ctx, time.Second)
	defer cancel()
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go runtime.instance.Router().RouteConnectionEx(ctx, server, adapter.InboundContext{
		Destination: M.SocksaddrFrom(netip.MustParseAddr("192.0.2.53"), port),
	}, func(err error) { done <- err })
	connection := &mdns.Conn{Conn: client}
	query := new(mdns.Msg)
	query.SetQuestion("unknown.test.", mdns.TypeA)
	if err := connection.WriteMsg(query); err != nil {
		t.Fatal(err)
	}
	response, err := connection.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("DNS hijack did not finish after connection close")
	}
	return response
}

func TestRouteDefaultsRuntimeHijacksUDPDNS(t *testing.T) {
	cases := []struct {
		name string
		port uint16
	}{
		{name: "standard port before sniff", port: 53},
		{name: "nonstandard port after sniff", port: 5353},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runtime := newPolicyRuntime(t, policyDefaultsFixture())
			response := policyUDPDNSExchange(t, runtime, test.port)
			if len(response.Answer) != 1 || runtime.remote.calls.Load() != 1 || runtime.direct.calls.Load() != 0 {
				t.Fatalf("hijacked UDP DNS answer=%v remote=%d direct=%d", response.Answer,
					runtime.remote.calls.Load(), runtime.direct.calls.Load())
			}
			if runtime.trace.outbound != "" {
				t.Fatalf("UDP DNS escaped to outbound %q", runtime.trace.outbound)
			}
		})
	}
}

func policyUDPDNSExchange(t *testing.T, runtime *policyRuntime, port uint16) *mdns.Msg {
	t.Helper()
	ctx, cancel := context.WithTimeout(runtime.ctx, time.Second)
	defer cancel()
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	client, err := net.Dial("udp", server.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go runtime.instance.Router().RoutePacketConnectionEx(ctx, bufio.NewPacketConn(server), adapter.InboundContext{
		Destination: M.SocksaddrFrom(netip.MustParseAddr("192.0.2.53"), port),
	}, func(err error) { done <- err })
	connection := &mdns.Conn{Conn: client}
	query := new(mdns.Msg)
	query.SetQuestion("unknown.test.", mdns.TypeA)
	if err := connection.WriteMsg(query); err != nil {
		t.Fatal(err)
	}
	response, err := connection.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("UDP DNS hijack did not finish after connection close")
	}
	return response
}
