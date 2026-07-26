package api

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/miekg/dns"
	quic "github.com/sagernet/quic-go"
)

func TestExchangeDNSQUICUsesDoQFraming(t *testing.T) {
	certificate, roots := newDNSProbeTestCertificate(t)
	listener, err := quic.ListenAddr("127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"doq"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	installDNSProbeTestRoots(t, roots)

	serverResult := make(chan error, 1)
	go func() { serverResult <- serveDNSQUICOnce(t.Context(), listener) }()

	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	response, err := exchangeDNSQUIC(t.Context(), message, listener.Addr().String(), "127.0.0.1", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || response.Rcode != dns.RcodeSuccess || len(response.Answer) != 1 {
		t.Fatalf("response = %+v", response)
	}
	if err := <-serverResult; err != nil {
		t.Fatal(err)
	}
}

func TestExchangeDNSHTTP3UsesHTTP3Post(t *testing.T) {
	requestResult := make(chan error, 1)
	port, roots := startDNSHTTP3TestServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestResult <- writeDNSHTTP3TestResponse(writer, request)
	}))
	installDNSProbeTestRoots(t, roots)

	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	response, err := exchangeDNSHTTP3(t.Context(), message, "127.0.0.1", port, "/custom-dns", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || response.Rcode != dns.RcodeSuccess || len(response.Answer) != 1 {
		t.Fatalf("response = %+v", response)
	}
	if err := <-requestResult; err != nil {
		t.Fatal(err)
	}
}

func TestDNSProbeQUICHelpersRejectInvalidInputs(t *testing.T) {
	if _, err := packDNSQUICFrame(nil); err == nil {
		t.Fatal("expected nil DoQ message error")
	}
	if _, err := packDNSHTTP3Message(nil); err == nil {
		t.Fatal("expected nil DoH3 message error")
	}
	if _, err := unpackDNSProbeMessage([]byte("invalid")); err == nil {
		t.Fatal("expected invalid DNS response error")
	}
	original := newDNSProbeTLSConfig
	t.Cleanup(func() { newDNSProbeTLSConfig = original })
	newDNSProbeTLSConfig = func(string) *tls.Config { return nil }
	if _, err := dnsProbeTLSConfig("dns.example", "doq"); err == nil {
		t.Fatal("expected unavailable TLS config error")
	}
}

func TestDNSHTTP3EndpointValidation(t *testing.T) {
	endpoint, err := dnsHTTP3Endpoint("dns.example", 0, "dns-query")
	if err != nil || endpoint != "https://dns.example:443/dns-query" {
		t.Fatalf("endpoint = %q err=%v", endpoint, err)
	}
	requests := []struct {
		name   string
		server string
		port   int
		path   string
	}{
		{name: "invalid server", server: "dns.example/path", path: "/dns-query"},
		{name: "invalid port", server: "dns.example", port: 65536, path: "/dns-query"},
		{name: "invalid path", server: "dns.example", path: "/dns-query?x=1"},
	}
	for _, request := range requests {
		t.Run(request.name, func(t *testing.T) {
			if _, err := dnsHTTP3Endpoint(request.server, request.port, request.path); err == nil {
				t.Fatal("expected endpoint validation error")
			}
		})
	}
}

func TestReadDNSQUICResponseRejectsInvalidFrames(t *testing.T) {
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	message.Id = 42
	wire, err := message.Pack()
	if err != nil {
		t.Fatal(err)
	}
	nonZeroID := make([]byte, 2+len(wire))
	binary.BigEndian.PutUint16(nonZeroID, uint16(len(wire)))
	copy(nonZeroID[2:], wire)
	tests := []struct {
		name  string
		frame []byte
	}{
		{name: "missing length"},
		{name: "empty response", frame: []byte{0, 0}},
		{name: "truncated response", frame: []byte{0, 2, 1}},
		{name: "invalid message", frame: []byte{0, 3, 'b', 'a', 'd'}},
		{name: "non-zero id", frame: nonZeroID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := readDNSQUICResponse(bytes.NewReader(test.frame)); err == nil {
				t.Fatal("expected invalid DoQ response error")
			}
		})
	}
}

func TestExchangeDNSHTTP3RejectsProtocolErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
	}{
		{name: "status", handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			http.Error(writer, "unavailable", http.StatusServiceUnavailable)
		})},
		{name: "invalid message", handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte("not-dns"))
		})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			port, roots := startDNSHTTP3TestServer(t, test.handler)
			installDNSProbeTestRoots(t, roots)
			message := new(dns.Msg)
			message.SetQuestion("example.com.", dns.TypeA)
			if _, err := exchangeDNSHTTP3(t.Context(), message, "127.0.0.1", port, "/dns-query", time.Second); err == nil {
				t.Fatal("expected DoH3 protocol error")
			}
		})
	}
}

func TestStopDNSProbeRedirect(t *testing.T) {
	if !errors.Is(stopDNSProbeRedirect(nil, nil), http.ErrUseLastResponse) {
		t.Fatal("expected redirects to remain disabled")
	}
}
