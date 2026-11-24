package dnsprobe

import (
	"context"
	"net"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/xuthus5/boxd/control"
)

func TestProbeUDPMeasuresRealRoundTrip(t *testing.T) {
	address, stop := startUDPDNSServer(t)
	defer stop()
	latency, err := New().Probe(context.Background(), control.DNSServerConfig{
		Enabled:        true,
		Protocol:       control.DNSProtocolUDP,
		Address:        address,
		TimeoutSeconds: 2,
	})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if latency <= 0 {
		t.Fatalf("latency = %v, want a positive measured duration", latency)
	}
}

func TestParseDoHURLDefaultsDNSQueryPath(t *testing.T) {
	parsed, err := parseDoHURL("dns.example.com")
	if err != nil {
		t.Fatalf("parseDoHURL() error = %v", err)
	}
	if parsed.Scheme != "https" || parsed.Path != "/dns-query" || parsed.Hostname() != "dns.example.com" {
		t.Fatalf("parsed DoH URL = %s", parsed)
	}
	if _, err := parseDoHURL("http://dns.example.com/dns-query"); err == nil {
		t.Fatal("parseDoHURL() accepted a non-TLS endpoint")
	}
}

func startUDPDNSServer(t *testing.T) (string, func()) {
	t.Helper()
	connection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen UDP DNS: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		buffer := make([]byte, 1500)
		length, remote, readErr := connection.ReadFrom(buffer)
		if readErr != nil {
			return
		}
		var query dnsmessage.Message
		if query.Unpack(buffer[:length]) != nil {
			return
		}
		response := dnsmessage.Message{
			Header: dnsmessage.Header{
				ID:                 query.ID,
				Response:           true,
				RecursionAvailable: true,
			},
			Questions: query.Questions,
			Answers: []dnsmessage.Resource{{
				Header: dnsmessage.ResourceHeader{
					Name:  query.Questions[0].Name,
					Type:  dnsmessage.TypeA,
					Class: dnsmessage.ClassINET,
					TTL:   60,
				},
				Body: &dnsmessage.AResource{A: [4]byte{192, 0, 2, 1}},
			}},
		}
		body, packErr := response.Pack()
		if packErr == nil {
			_, _ = connection.WriteTo(body, remote)
		}
	}()
	return connection.LocalAddr().String(), func() {
		_ = connection.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}
