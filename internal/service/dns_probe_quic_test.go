package service

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/miekg/dns"
	quic "github.com/sagernet/quic-go"
)

type fakeQUICConn struct {
	openErr error
}

func (c *fakeQUICConn) OpenStreamSync(context.Context) (*quic.Stream, error) {
	if c.openErr != nil {
		return nil, c.openErr
	}
	return nil, errors.New("stream unavailable")
}
func (c *fakeQUICConn) CloseWithError(quic.ApplicationErrorCode, string) error {
	return nil
}

func TestExchangeDNSQUICErrors(t *testing.T) {
	original := dialDNSProbeQUIC
	t.Cleanup(func() { dialDNSProbeQUIC = original })
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)

	dialDNSProbeQUIC = func(context.Context, string, *tls.Config, *quic.Config) (dnsProbeQUICConn, error) {
		return nil, errors.New("dial failed")
	}
	if _, err := exchangeDNSQUIC(context.Background(), message, "127.0.0.1:853", "localhost", time.Second); err == nil {
		t.Fatal("expected dial error")
	}

	dialDNSProbeQUIC = func(context.Context, string, *tls.Config, *quic.Config) (dnsProbeQUICConn, error) {
		return &fakeQUICConn{openErr: errors.New("open failed")}, nil
	}
	if _, err := exchangeDNSQUIC(context.Background(), message, "127.0.0.1:853", "localhost", time.Second); err == nil {
		t.Fatal("expected open error")
	}
}

func TestDNSProbeTLSConfig(t *testing.T) {
	original := newDNSProbeTLSConfig
	t.Cleanup(func() { newDNSProbeTLSConfig = original })
	newDNSProbeTLSConfig = func(string) *tls.Config { return nil }
	if _, err := dnsProbeTLSConfig("server", "doq"); err == nil {
		t.Fatal("expected error for nil tls config")
	}
	newDNSProbeTLSConfig = func(name string) *tls.Config {
		return &tls.Config{ServerName: name}
	}
	cfg, err := dnsProbeTLSConfig("server", "doq")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Fatalf("min version = %d", cfg.MinVersion)
	}
	if len(cfg.NextProtos) != 1 || cfg.NextProtos[0] != "doq" {
		t.Fatalf("next protos = %v", cfg.NextProtos)
	}
	cfg2, err := dnsProbeTLSConfig("server", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg2.NextProtos) != 0 {
		t.Fatalf("next protos = %v", cfg2.NextProtos)
	}
}

func TestExchangeDNSHTTP3PackErrors(t *testing.T) {
	message := new(dns.Msg)
	if _, err := exchangeDNSHTTP3(context.Background(), message, "bad server", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected invalid server error")
	}
}
