package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestProbeDNSHandlerStopsOnRequestCancellation(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	started := make(chan struct{})
	dnsUDPExchange = func(ctx context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}

	ctx, cancel := context.WithCancel(t.Context())
	request := jsonRequest(http.MethodPost, "/api/runtime/dns/probe", `{"type":"udp","server":"1.1.1.1"}`).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		NewRuntimeHandler(&fakeRuntimeInstance{}).ProbeDNS(recorder, request)
		close(done)
	}()
	waitDNSProbeSignal(t, started)
	cancel()
	waitDNSProbeSignal(t, done)
	if recorder.Body.Len() != 0 {
		t.Fatalf("unexpected response after cancellation: %s", recorder.Body.String())
	}
}

func TestProbeDNSBatchStopsQueuedWorkOnCancellation(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	var started atomic.Int32
	startedSignal := make(chan struct{}, 8)
	dnsUDPExchange = func(ctx context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		started.Add(1)
		startedSignal <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	}

	items := strings.TrimSuffix(strings.Repeat(`{"type":"udp","server":"1.1.1.1"},`, 8), ",")
	ctx, cancel := context.WithCancel(t.Context())
	request := jsonRequest(http.MethodPost, "/api/runtime/dns/probe-batch", `{"items":[`+items+`],"concurrency":2}`).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		NewRuntimeHandler(&fakeRuntimeInstance{}).ProbeDNSBatch(recorder, request)
		close(done)
	}()
	waitDNSProbeSignal(t, startedSignal)
	waitDNSProbeSignal(t, startedSignal)
	cancel()
	waitDNSProbeSignal(t, done)
	if count := started.Load(); count != 2 {
		t.Fatalf("started probes = %d, want 2", count)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("unexpected batch response after cancellation: %s", recorder.Body.String())
	}
}

func TestExchangeDNSDoHUsesParentContext(t *testing.T) {
	original := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = original })
	started := make(chan struct{}, 1)
	var calls atomic.Int32
	newDNSHTTPClient = func(string, time.Duration) *http.Client {
		return &http.Client{Transport: dnsProbeRoundTripper(func(request *http.Request) (*http.Response, error) {
			calls.Add(1)
			select {
			case started <- struct{}{}:
			default:
			}
			<-request.Context().Done()
			return nil, request.Context().Err()
		})}
	}

	ctx, cancel := context.WithCancel(t.Context())
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	done := make(chan error, 1)
	go func() {
		_, err := exchangeDNSDoH(ctx, message, "dns.example", 443, "/dns-query", time.Second)
		done <- err
	}()
	waitDNSProbeSignal(t, started)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("exchange error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("DoH exchange did not stop after cancellation")
	}
	if count := calls.Load(); count != 1 {
		t.Fatalf("DoH requests after cancellation = %d, want 1", count)
	}
}

func TestDNSProbeExchangesRespectCanceledContext(t *testing.T) {
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	tests := []struct {
		name     string
		exchange func(context.Context) error
	}{
		{name: "udp", exchange: func(ctx context.Context) error {
			_, err := exchangeDNSUDP(ctx, message, "127.0.0.1:1", time.Second)
			return err
		}},
		{name: "tcp", exchange: func(ctx context.Context) error {
			_, err := exchangeDNSTCP(ctx, message, "127.0.0.1:1", time.Second)
			return err
		}},
		{name: "tls", exchange: func(ctx context.Context) error {
			_, err := exchangeDNSTLS(ctx, message, "127.0.0.1:1", "localhost", time.Second)
			return err
		}},
		{name: "quic", exchange: func(ctx context.Context) error {
			_, err := exchangeDNSQUIC(ctx, message, "127.0.0.1:1", "localhost", time.Second)
			return err
		}},
		{name: "h3", exchange: func(ctx context.Context) error {
			_, err := exchangeDNSHTTP3(ctx, message, "127.0.0.1", 1, "/dns-query", time.Second)
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if err := test.exchange(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("exchange error = %v, want context canceled", err)
			}
		})
	}
}

type dnsProbeRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip dnsProbeRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func waitDNSProbeSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for DNS probe signal")
	}
}
