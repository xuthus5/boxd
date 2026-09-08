package core

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	sbdns "github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/block"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
)

type policyRuntime struct {
	ctx      context.Context
	instance *box.Box
	direct   *policyDNSTransport
	remote   *policyDNSTransport
	mode     *policyClashMode
	trace    *policyRouteTrace
}

func newPolicyRuntime(t *testing.T, cfg map[string]any) *policyRuntime {
	t.Helper()
	runtime := &policyRuntime{
		ctx:    include.Context(t.Context()),
		direct: &policyDNSTransport{TransportAdapter: sbdns.NewTransportAdapter("policy-test", "dns-direct", nil)},
		remote: &policyDNSTransport{TransportAdapter: sbdns.NewTransportAdapter("policy-test", "dns-remote", nil)},
		mode:   &policyClashMode{mode: "Rule"},
		trace:  &policyRouteTrace{},
	}
	runtime.registerTransports()
	options := runtime.options(t, cfg)
	instance, err := box.New(box.Options{Context: runtime.ctx, Options: options})
	if err != nil {
		t.Fatal(err)
	}
	runtime.instance = instance
	instance.Router().AppendTracker(runtime.trace)
	t.Cleanup(func() {
		if err := instance.Close(); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	})
	if err := instance.Start(); err != nil {
		t.Fatal(err)
	}
	return runtime
}

func (r *policyRuntime) registerTransports() {
	registry := service.FromContext[adapter.DNSTransportRegistry](r.ctx).(*sbdns.TransportRegistry)
	sbdns.RegisterTransport[option.StubOptions](registry, "policy-test", func(
		_ context.Context, _ log.ContextLogger, tag string, _ option.StubOptions,
	) (adapter.DNSTransport, error) {
		if tag == "dns-direct" {
			return r.direct, nil
		}
		return r.remote, nil
	})
	// 出站一律由内核 block 实现终止，验证真实分流但不连接测试机外部网络。
	outboundRegistry := service.FromContext[adapter.OutboundRegistry](r.ctx).(*outbound.Registry)
	outbound.Register[option.DirectOutboundOptions](outboundRegistry, "direct", func(
		ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, _ option.DirectOutboundOptions,
	) (adapter.Outbound, error) {
		return block.New(ctx, router, logger, tag, option.StubOptions{})
	})
	service.MustRegister[adapter.ClashServer](r.ctx, r.mode)
}

func (r *policyRuntime) options(t *testing.T, cfg map[string]any) option.Options {
	t.Helper()
	dnsResult, err := NewDefaultDNSInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	routeResult, err := NewDefaultRouteInstaller().Install(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg["dns"] = dnsResult.DNS
	cfg["log"] = map[string]any{"disabled": true}
	route := cfg["route"].(map[string]any)
	route["rules"] = routeResult.Rules
	route["default_domain_resolver"] = dnsResult.DefaultDomainResolver
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var options option.Options
	if err := options.UnmarshalJSONContext(r.ctx, body); err != nil {
		t.Fatal(err)
	}
	for index, server := range options.DNS.Servers {
		options.DNS.Servers[index] = option.DNSServerOptions{
			Type: "policy-test", Tag: server.Tag, Options: &option.StubOptions{},
		}
	}
	return options
}

func (r *policyRuntime) query(domain string) (*mdns.Msg, error) {
	message := new(mdns.Msg)
	message.SetQuestion(mdns.Fqdn(domain), mdns.TypeA)
	router := service.FromContext[adapter.DNSRouter](r.ctx)
	return router.Exchange(r.ctx, message, adapter.DNSQueryOptions{})
}

func (r *policyRuntime) route(t *testing.T, domain string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(r.ctx, time.Second)
	defer cancel()
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()
	r.trace.outbound = ""
	done := make(chan error, 1)
	r.instance.Router().RouteConnectionEx(ctx, server, adapter.InboundContext{
		Destination: M.ParseSocksaddr(net.JoinHostPort(domain, "443")), Protocol: "tls",
	}, func(err error) { done <- err })
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		t.Fatal("route did not complete")
		return ctx.Err()
	}
}

type policyDNSTransport struct {
	sbdns.TransportAdapter
	calls atomic.Int32
	fail  bool
}

func (s *policyDNSTransport) Start(_ adapter.StartStage) error { return nil }

func (s *policyDNSTransport) Close() error { return nil }

func (s *policyDNSTransport) Reset() {}

func (s *policyDNSTransport) Exchange(_ context.Context, message *mdns.Msg) (*mdns.Msg, error) {
	s.calls.Add(1)
	if s.fail {
		return nil, errors.New("test DNS upstream unavailable")
	}
	response := new(mdns.Msg)
	response.SetReply(message)
	if message.Question[0].Qtype != mdns.TypeA {
		return response, nil
	}
	address := "198.51.100.1"
	switch message.Question[0].Name {
	case "cn-only.test.":
		address = "203.0.113.1"
	case "private-only.test.":
		address = "10.1.2.3"
	}
	response.Answer = []mdns.RR{&mdns.A{
		Hdr: mdns.RR_Header{Name: message.Question[0].Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
		A:   net.ParseIP(address).To4(),
	}}
	return response, nil
}

type policyClashMode struct {
	adapter.ClashServer
	mode string
}

func (s *policyClashMode) Mode() string { return s.mode }

type policyRouteTrace struct {
	adapter.ConnectionTracker
	outbound string
	metadata adapter.InboundContext
}

func (s *policyRouteTrace) RoutedConnection(
	_ context.Context, conn net.Conn, metadata adapter.InboundContext, _ adapter.Rule, selected adapter.Outbound,
) net.Conn {
	s.outbound = selected.Tag()
	s.metadata = metadata
	return conn
}

func (s *policyRouteTrace) RoutedPacketConnection(
	_ context.Context, conn N.PacketConn, _ adapter.InboundContext, _ adapter.Rule, selected adapter.Outbound,
) N.PacketConn {
	s.outbound = selected.Tag()
	return conn
}
