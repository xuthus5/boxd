package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestDNSProbeServiceProbe(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	dnsUDPExchange = func(_ context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		response := new(dns.Msg)
		response.Rcode = dns.RcodeSuccess
		response.Answer = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60}, A: parseTestIPv4("1.2.3.4")}}
		return response, nil
	}
	svc := &ServiceSet{dnsProbe: &DNSProbe{}}
	result, err := svc.DNSProbe().Probe(context.Background(), DNSProbeRequest{
		Type: "udp", Server: "1.1.1.1", Domain: "example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("success = %v, error = %s", result.Success, result.Error)
	}
	if len(result.Answers) != 1 {
		t.Fatalf("answers = %v", result.Answers)
	}
}

func TestDNSProbeServiceProbeCancellation(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	dnsUDPExchange = func(ctx context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := &ServiceSet{dnsProbe: &DNSProbe{}}
	_, err := svc.DNSProbe().Probe(ctx, DNSProbeRequest{Type: "udp", Server: "1.1.1.1"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestDNSProbeServiceProbeInvalidDomain(t *testing.T) {
	svc := &ServiceSet{dnsProbe: &DNSProbe{}}
	result, err := svc.DNSProbe().Probe(context.Background(), DNSProbeRequest{
		Type: "udp", Server: "1.1.1.1", Domain: strings.Repeat("a", 300),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("expected failure for invalid domain")
	}
	if result.ErrorCode != "invalid_input" {
		t.Fatalf("error code = %q", result.ErrorCode)
	}
}

func TestDNSProbeServiceProbeBatch(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	dnsUDPExchange = func(_ context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		response := new(dns.Msg)
		response.Rcode = dns.RcodeSuccess
		return response, nil
	}
	svc := &ServiceSet{dnsProbe: &DNSProbe{}}
	results, err := svc.DNSProbe().ProbeBatch(context.Background(), []DNSProbeRequest{
		{Type: "udp", Server: "1.1.1.1"},
		{Type: "udp", Server: "8.8.8.8"},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d", len(results))
	}
}

func TestDNSProbeServiceProbeBatchEmpty(t *testing.T) {
	svc := &ServiceSet{dnsProbe: &DNSProbe{}}
	_, err := svc.DNSProbe().ProbeBatch(context.Background(), nil, 1)
	if err == nil {
		t.Fatal("expected error for empty items")
	}
}

func TestDNSProbeServiceProbeBatchTooMany(t *testing.T) {
	items := make([]DNSProbeRequest, maxDNSProbeItems+1)
	svc := &ServiceSet{dnsProbe: &DNSProbe{}}
	_, err := svc.DNSProbe().ProbeBatch(context.Background(), items, 1)
	if err == nil {
		t.Fatal("expected error for too many items")
	}
}

func TestDNSProbeServiceProbeBatchCancellation(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	started := make(chan struct{}, 8)
	dnsUDPExchange = func(ctx context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	items := make([]DNSProbeRequest, 8)
	for i := range items {
		items[i] = DNSProbeRequest{Type: "udp", Server: "1.1.1.1"}
	}
	done := make(chan error, 1)
	go func() {
		_, err := (&ServiceSet{dnsProbe: &DNSProbe{}}).DNSProbe().ProbeBatch(ctx, items, 2)
		done <- err
	}()
	waitProbeSignal(t, started)
	waitProbeSignal(t, started)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("batch did not stop after cancellation")
	}
}

func TestDNSProbeExchangesRespectCanceledContext(t *testing.T) {
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "udp", run: func() error {
			_, err := exchangeDNSUDP(ctx, message, "127.0.0.1:1", time.Second)
			return err
		}},
		{name: "tcp", run: func() error {
			_, err := exchangeDNSTCP(ctx, message, "127.0.0.1:1", time.Second)
			return err
		}},
		{name: "tls", run: func() error {
			_, err := exchangeDNSTLS(ctx, message, "127.0.0.1:1", "localhost", time.Second)
			return err
		}},
		{name: "quic", run: func() error {
			_, err := exchangeDNSQUIC(ctx, message, "127.0.0.1:1", "localhost", time.Second)
			return err
		}},
		{name: "h3", run: func() error {
			_, err := exchangeDNSHTTP3(ctx, message, "127.0.0.1", 1, "/dns-query", time.Second)
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v, want context canceled", err)
			}
		})
	}
}

func TestDNSProbeHTTP3PackUnpack(t *testing.T) {
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	wire, err := packDNSHTTP3Message(message)
	if err != nil {
		t.Fatal(err)
	}
	out, err := unpackDNSProbeMessage(wire)
	if err != nil {
		t.Fatal(err)
	}
	if out.Id != 0 {
		t.Fatalf("id = %d", out.Id)
	}
	if _, err := packDNSHTTP3Message(nil); err == nil {
		t.Fatal("expected error for nil message")
	}
	if _, err := packDNSProbeMessage(nil); err == nil {
		t.Fatal("expected error for nil message")
	}
	if _, err := unpackDNSProbeMessage([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected unpack error")
	}
}

func TestDNSProbeDoHPostFallback(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	var calls atomic.Int32
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return &http.Client{Transport: probeRoundTripper(func(request *http.Request) (*http.Response, error) {
			calls.Add(1)
			if request.Method == http.MethodGet {
				return &http.Response{StatusCode: http.StatusBadRequest, Body: nopReadCloser("bad"), Request: request}, nil
			}
			response := new(dns.Msg)
			response.Rcode = dns.RcodeSuccess
			wire, err := response.Pack()
			if err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Body: nopReadCloserBytes(wire), Request: request}, nil
		})}
	}
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	out, err := exchangeDNSDoH(context.Background(), message, "dns.example", 443, "/dns-query", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("nil response")
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestReadDNSMessageBodyTooLarge(t *testing.T) {
	big := make([]byte, maxDNSMessageBytes+1)
	if _, err := readDNSMessageBody(strings.NewReader(string(big))); err == nil {
		t.Fatal("expected too large error")
	}
	if _, err := readDNSMessageBody(nil); err == nil {
		t.Fatal("expected nil error")
	}
}

func waitProbeSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for signal")
	}
}

func parseTestIPv4(s string) net.IP {
	return net.ParseIP(s).To4()
}

type probeRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip probeRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func nopReadCloser(body string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(body))
}

func nopReadCloserBytes(body []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(body))
}
