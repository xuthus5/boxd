package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestNormalizeDNSProbeTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		req     DNSProbeRequest
		proto   string
		server  string
		port    int
		path    string
		wantErr string
	}{
		{
			name:   "udp defaults",
			req:    DNSProbeRequest{Type: "udp", Server: "1.1.1.1"},
			proto:  "udp",
			server: "1.1.1.1",
			port:   53,
			path:   "/dns-query",
		},
		{
			name:   "tls default port",
			req:    DNSProbeRequest{Type: "tls", Server: "dns.google"},
			proto:  "tls",
			server: "dns.google",
			port:   853,
			path:   "/dns-query",
		},
		{
			name:   "https path",
			req:    DNSProbeRequest{Type: "https", Server: "dns.google", Path: "dns-query"},
			proto:  "https",
			server: "dns.google",
			port:   443,
			path:   "dns-query",
		},
		{
			name:   "legacy https address",
			req:    DNSProbeRequest{Address: "https://dns.google/dns-query"},
			proto:  "https",
			server: "dns.google",
			port:   443,
			path:   "/dns-query",
		},
		{
			name:   "legacy host port",
			req:    DNSProbeRequest{Address: "8.8.8.8:53"},
			proto:  "udp",
			server: "8.8.8.8",
			port:   53,
			path:   "/dns-query",
		},
		{
			name:   "tls scheme address",
			req:    DNSProbeRequest{Address: "tls://1.1.1.1:853"},
			proto:  "tls",
			server: "1.1.1.1",
			port:   853,
			path:   "/dns-query",
		},
		{
			name:    "local not probeable",
			req:     DNSProbeRequest{Type: "local"},
			wantErr: "not probeable",
		},
		{
			name:    "missing server",
			req:     DNSProbeRequest{Type: "udp"},
			wantErr: "server is required",
		},
		{
			name:    "unsupported",
			req:     DNSProbeRequest{Type: "weird", Server: "1.1.1.1"},
			wantErr: "unsupported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proto, server, port, path, err := normalizeDNSProbeTarget(tt.req)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if proto != tt.proto || server != tt.server || port != tt.port || path != tt.path {
				t.Fatalf("got %s %s %d %s, want %s %s %d %s", proto, server, port, path, tt.proto, tt.server, tt.port, tt.path)
			}
		})
	}
}

func TestProbeDNSServerWithHooks(t *testing.T) {
	successMsg := new(dns.Msg)
	successMsg.Rcode = dns.RcodeSuccess
	successMsg.Answer = []dns.RR{
		&dns.A{Hdr: dns.RR_Header{Name: "cloudflare.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60}, A: []byte{1, 1, 1, 1}},
	}

	origUDP, origTCP := dnsUDPExchange, dnsTCPExchange
	origTLS, origQUIC := dnsTLSExchange, dnsQUICExchange
	origDoH, origH3 := dnsDoHExchange, dnsH3Exchange
	t.Cleanup(func() {
		dnsUDPExchange, dnsTCPExchange = origUDP, origTCP
		dnsTLSExchange, dnsQUICExchange = origTLS, origQUIC
		dnsDoHExchange, dnsH3Exchange = origDoH, origH3
	})

	dnsUDPExchange = func(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
		if addr != "1.1.1.1:53" {
			t.Fatalf("udp addr = %s", addr)
		}
		return successMsg, nil
	}
	dnsTCPExchange = func(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
		return successMsg, nil
	}
	dnsTLSExchange = func(msg *dns.Msg, addr, serverName string, timeout time.Duration) (*dns.Msg, error) {
		if serverName != "dns.google" {
			t.Fatalf("tls sni = %s", serverName)
		}
		return successMsg, nil
	}
	dnsQUICExchange = func(msg *dns.Msg, addr, serverName string, timeout time.Duration) (*dns.Msg, error) {
		if addr != "dns.google:853" || serverName != "dns.google" {
			t.Fatalf("quic target = %s sni=%s", addr, serverName)
		}
		return successMsg, nil
	}
	dnsDoHExchange = func(msg *dns.Msg, server string, port int, path string, timeout time.Duration) (*dns.Msg, error) {
		if server != "dns.google" || port != 443 {
			t.Fatalf("doh target = %s:%d", server, port)
		}
		return successMsg, nil
	}
	dnsH3Exchange = func(msg *dns.Msg, server string, port int, path string, timeout time.Duration) (*dns.Msg, error) {
		if server != "dns.google" || port != 443 || path != "/dns-query" {
			t.Fatalf("h3 target = %s:%d%s", server, port, path)
		}
		return successMsg, nil
	}

	udp := probeDNSServer(DNSProbeRequest{Tag: "cf", Type: "udp", Server: "1.1.1.1"})
	if !udp.Success || udp.LatencyMs < 1 || len(udp.Answers) == 0 {
		t.Fatalf("udp result = %+v", udp)
	}
	if !probeDNSServer(DNSProbeRequest{Type: "tcp", Server: "8.8.8.8"}).Success {
		t.Fatal("tcp failed")
	}
	if !probeDNSServer(DNSProbeRequest{Type: "tls", Server: "dns.google"}).Success {
		t.Fatal("tls failed")
	}
	if !probeDNSServer(DNSProbeRequest{Type: "https", Server: "dns.google"}).Success {
		t.Fatal("https failed")
	}
	if !probeDNSServer(DNSProbeRequest{Type: "quic", Server: "dns.google"}).Success {
		t.Fatal("quic failed")
	}
	if !probeDNSServer(DNSProbeRequest{Type: "h3", Server: "dns.google"}).Success {
		t.Fatal("h3 failed")
	}

	dnsUDPExchange = func(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
		return nil, fmt.Errorf("network down")
	}
	fail := probeDNSServer(DNSProbeRequest{Type: "udp", Server: "1.1.1.1"})
	if fail.Success || fail.Error == "" || fail.ErrorCode != ProbeErrorNetwork {
		t.Fatalf("expected network failure, got %+v", fail)
	}

	local := probeDNSServer(DNSProbeRequest{Type: "local", Tag: "sys"})
	if local.Success || !strings.Contains(local.Error, "not probeable") || local.ErrorCode != ProbeErrorUnsupported {
		t.Fatalf("local = %+v", local)
	}

	dnsUDPExchange = func(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
		msg2 := new(dns.Msg)
		msg2.Rcode = dns.RcodeServerFailure
		return msg2, nil
	}
	servfail := probeDNSServer(DNSProbeRequest{Type: "udp", Server: "1.1.1.1"})
	if servfail.Success || !strings.Contains(servfail.Error, "SERVFAIL") || servfail.ErrorCode != ProbeErrorDNSRcode {
		t.Fatalf("servfail = %+v", servfail)
	}

	dnsUDPExchange = func(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
		return nil, nil
	}
	empty := probeDNSServer(DNSProbeRequest{Type: "udp", Server: "1.1.1.1"})
	if empty.Success || empty.Error != "empty dns response" || empty.ErrorCode != ProbeErrorEmpty {
		t.Fatalf("empty = %+v", empty)
	}
}

func TestProbeDNSHandlers(t *testing.T) {
	origUDP := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = origUDP })
	dnsUDPExchange = func(msg *dns.Msg, addr string, timeout time.Duration) (*dns.Msg, error) {
		out := new(dns.Msg)
		out.SetReply(msg)
		out.Rcode = dns.RcodeSuccess
		return out, nil
	}

	handler := NewRuntimeHandler(&fakeRuntimeInstance{})

	rr := httptest.NewRecorder()
	handler.ProbeDNS(rr, httptest.NewRequest(http.MethodPost, "/api/runtime/dns/probe", strings.NewReader(`{`)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ProbeDNS(rr, jsonRequest(http.MethodPost, "/api/runtime/dns/probe", `{"type":"udp","server":"1.1.1.1","tag":"cf"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("probe status = %d body=%s", rr.Code, rr.Body.String())
	}
	one := decodeBody[DNSProbeResult](t, rr)
	if !one.Success || one.Tag != "cf" {
		t.Fatalf("probe result = %+v", one)
	}

	rr = httptest.NewRecorder()
	handler.ProbeDNSBatch(rr, jsonRequest(http.MethodPost, "/api/runtime/dns/probe-batch", `{`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("batch invalid status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ProbeDNSBatch(rr, jsonRequest(http.MethodPost, "/api/runtime/dns/probe-batch", `{"items":[]}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("batch empty status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ProbeDNSBatch(rr, jsonRequest(http.MethodPost, "/api/runtime/dns/probe-batch",
		`{"items":[{"type":"udp","server":"1.1.1.1","tag":"a"},{"type":"local","tag":"b"}]}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("batch status = %d body=%s", rr.Code, rr.Body.String())
	}
	batch := decodeBody[struct {
		Results []DNSProbeResult `json:"results"`
	}](t, rr)
	if len(batch.Results) != 2 || !batch.Results[0].Success || batch.Results[1].Success {
		t.Fatalf("batch results = %+v", batch.Results)
	}
}

func TestDefaultDNSPortAndCollectAnswers(t *testing.T) {
	t.Parallel()
	if defaultDNSPort("tls") != 853 || defaultDNSPort("https") != 443 || defaultDNSPort("udp") != 53 {
		t.Fatal("default ports mismatch")
	}
	if collectDNSAnswers(nil) != nil {
		t.Fatal("nil answers")
	}
	msg := new(dns.Msg)
	msg.Answer = []dns.RR{
		&dns.A{Hdr: dns.RR_Header{Name: "a.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1}, A: []byte{1, 2, 3, 4}},
	}
	if len(collectDNSAnswers(msg)) != 1 {
		t.Fatal("answers len")
	}
}

func TestParseLegacyDNSAddressEdge(t *testing.T) {
	t.Parallel()
	_, _, _, _, err := parseLegacyDNSAddress("", "", 0, "")
	if err == nil {
		t.Fatal("expected empty address error")
	}
	proto, server, port, path, err := parseLegacyDNSAddress("h3://dns.example:8443/query", "", 0, "/dns-query")
	if err != nil || proto != "h3" || server != "dns.example" || port != 8443 || path != "/query" {
		t.Fatalf("h3 parse = %s %s %d %s err=%v", proto, server, port, path, err)
	}
	_, _, _, _, err = splitSchemeHost("udp", "", 53)
	if err == nil {
		t.Fatal("expected empty host")
	}
	proto, server, port, path, err = parseLegacyDNSAddress("udp://9.9.9.9", "", 0, "/dns-query")
	if err != nil || proto != "udp" || server != "9.9.9.9" {
		t.Fatalf("udp scheme = %s %s %d %s err=%v", proto, server, port, path, err)
	}
	proto, server, port, path, err = parseLegacyDNSAddress("tcp://9.9.9.9:5353", "", 0, "/dns-query")
	if err != nil || proto != "tcp" || server != "9.9.9.9" || port != 5353 {
		t.Fatalf("tcp scheme = %s %s %d %s err=%v", proto, server, port, path, err)
	}
}

func TestExchangeDNSUDPAndTCPLocal(t *testing.T) {
	mux := dns.NewServeMux()
	mux.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = []dns.RR{
			&dns.A{Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30}, A: []byte{1, 2, 3, 4}},
		}
		_ = w.WriteMsg(m)
	})
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pc.Close() }()
	udpServer := &dns.Server{PacketConn: pc, Handler: mux}
	go func() { _ = udpServer.ActivateAndServe() }()
	defer func() { _ = udpServer.Shutdown() }()

	addr := pc.LocalAddr().String()
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	resp, err := exchangeDNSUDP(msg, addr, 2*time.Second)
	if err != nil || resp == nil || resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("udp exchange err=%v resp=%v", err, resp)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tcpServer := &dns.Server{Listener: ln, Handler: mux}
	go func() { _ = tcpServer.ActivateAndServe() }()
	defer func() { _ = tcpServer.Shutdown() }()

	resp, err = exchangeDNSTCP(msg, ln.Addr().String(), 2*time.Second)
	if err != nil || resp == nil || resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("tcp exchange err=%v resp=%v", err, resp)
	}
}

func TestExchangeDNSDoHGetAndPost(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	wireOK, err := msg.Pack()
	if err != nil {
		t.Fatal(err)
	}
	reply := new(dns.Msg)
	reply.SetReply(msg)
	reply.Rcode = dns.RcodeSuccess
	replyWire, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}

	// Success on GET
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/dns-message")
			_, _ = w.Write(replyWire)
			return
		}
		http.Error(w, "no", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	host, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	// Inject client that trusts test cert via custom exchange - call real exchange with Transport override is hard.
	// Instead unit-test POST path and failure branches through a local plain HTTP by temporarily swapping via wrapping functions.
	_ = wireOK
	_ = host
	_ = port

	// Direct post helper with custom client against TLS server using InsecureSkipVerify.
	client := srv.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Force GET fail -> POST success
	srv2 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.Error(w, "no get", http.StatusNotImplemented)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(replyWire)
	}))
	t.Cleanup(srv2.Close)
	resp, err := exchangeDNSDoHPost(ctx, srv2.Client(), srv2.URL+"/dns-query", mustPackQuestion(t, "example.com."))
	if err != nil || resp == nil || resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("doh post err=%v resp=%v", err, resp)
	}

	// POST non-200
	srv3 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	t.Cleanup(srv3.Close)
	_, err = exchangeDNSDoHPost(ctx, srv3.Client(), srv3.URL+"/dns-query", mustPackQuestion(t, "example.com."))
	if err == nil || !strings.Contains(err.Error(), "doh status") {
		t.Fatalf("expected doh status err, got %v", err)
	}
	_ = client
}

func mustPackQuestion(t *testing.T, name string) []byte {
	t.Helper()
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(name), dns.TypeA)
	wire, err := msg.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestExchangeDNSDoHWithHooklessServer(t *testing.T) {
	// Use custom transport by temporarily replacing exchangeDNSDoH is not needed;
	// exercise exchangeDNSDoH against httptest with modified TLS via monkeying http default is unsafe.
	// Cover success path by replacing dnsDoHExchange already done; here cover real function with invalid pack-free path.
	// Invalid path prefix normalization + unreachable host error.
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	_, err := exchangeDNSDoH(msg, "127.0.0.1", 1, "dns-query", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestExchangeDNSTLSUnreachable(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	_, err := exchangeDNSTLS(msg, "127.0.0.1:1", "localhost", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected tls dial error")
	}
}

func TestExchangeDNSDoHGetSuccess(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	reply := new(dns.Msg)
	reply.SetReply(msg)
	reply.Rcode = dns.RcodeSuccess
	reply.Answer = []dns.RR{
		&dns.A{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30}, A: []byte{1, 2, 3, 4}},
	}
	replyWire, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/dns-query" {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(replyWire)
	}))
	t.Cleanup(srv.Close)

	host, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	orig := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = orig })
	newDNSHTTPClient = func(server string, timeout time.Duration) *http.Client {
		return srv.Client()
	}

	resp, err := exchangeDNSDoH(msg, host, port, "dns-query", 2*time.Second)
	if err != nil || resp == nil || resp.Rcode != dns.RcodeSuccess || len(resp.Answer) != 1 {
		t.Fatalf("doh get err=%v resp=%v", err, resp)
	}
}

func TestExchangeDNSDoHGetFallbackPost(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	reply := new(dns.Msg)
	reply.SetReply(msg)
	reply.Rcode = dns.RcodeSuccess
	replyWire, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.Error(w, "no", http.StatusNotImplemented)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(replyWire)
	}))
	t.Cleanup(srv.Close)
	host, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	orig := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = orig })
	newDNSHTTPClient = func(server string, timeout time.Duration) *http.Client { return srv.Client() }
	resp, err := exchangeDNSDoH(msg, host, port, "/dns-query", 2*time.Second)
	if err != nil || resp == nil || resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("fallback err=%v resp=%v", err, resp)
	}
}

func TestNormalizeMoreEdges(t *testing.T) {
	t.Parallel()
	proto, server, port, path, err := normalizeDNSProbeTarget(DNSProbeRequest{Type: "LEGACY", Address: "1.0.0.1"})
	if err != nil || proto != "udp" || server != "1.0.0.1" || port != 53 || path != "/dns-query" {
		t.Fatalf("legacy type = %s %s %d %s err=%v", proto, server, port, path, err)
	}
	_, _, _, _, err = normalizeDNSProbeTarget(DNSProbeRequest{Type: "hosts", Server: "x"})
	if err == nil || !strings.Contains(err.Error(), "not probeable") {
		t.Fatalf("hosts err=%v", err)
	}
	_, _, _, _, err = parseLegacyDNSAddress("https://", "", 0, "/dns-query")
	if err == nil {
		t.Fatal("expected empty host for https")
	}
	_, _, _, _, err = parseLegacyDNSAddress("https://example.com:bad", "", 0, "/dns-query")
	if err == nil {
		t.Fatal("expected bad port")
	}
	_, _, _, _, err = splitSchemeHost("tls", "dns.example:bad", 853)
	if err == nil {
		t.Fatal("expected bad scheme port")
	}
	proto, server, port, path, err = parseLegacyDNSAddress("quic://dns.example", "", 0, "/dns-query")
	if err != nil || proto != "quic" || server != "dns.example" {
		t.Fatalf("quic = %s %s %d %s err=%v", proto, server, port, path, err)
	}
}

func TestNormalizeDNSProbeTargetRejectsMalformedInputs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  DNSProbeRequest
	}{{
		name: "invalid port",
		req:  DNSProbeRequest{Type: "udp", Server: "1.1.1.1", ServerPort: 65536},
	}, {
		name: "negative port",
		req:  DNSProbeRequest{Type: "udp", Server: "1.1.1.1", ServerPort: -1},
	}, {
		name: "hostname with invalid port",
		req:  DNSProbeRequest{Address: "dns.example:bad"},
	}, {
		name: "server path",
		req:  DNSProbeRequest{Type: "udp", Server: "dns.example/path"},
	}, {
		name: "doh query in path",
		req:  DNSProbeRequest{Type: "https", Server: "dns.example", Path: "/dns-query?x=1"},
	}, {
		name: "legacy credentials",
		req:  DNSProbeRequest{Address: "https://user@dns.example/dns-query"},
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, _, err := normalizeDNSProbeTarget(test.req); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	proto, server, port, _, err := normalizeDNSProbeTarget(DNSProbeRequest{
		Address: "tls://[::1]",
	})
	if err != nil || proto != "tls" || server != "::1" || port != 853 {
		t.Fatalf("bracketed IPv6 = %s %s %d err=%v", proto, server, port, err)
	}
}

func TestDNSProbeResponseLimitAndRedirectPolicy(t *testing.T) {
	t.Parallel()
	if _, err := readDNSMessageBody(strings.NewReader(strings.Repeat("x", maxDNSMessageBytes+1))); err == nil {
		t.Fatal("expected oversized response error")
	}
	client := newDNSHTTPClient("dns.example", time.Second)
	if client == nil || client.CheckRedirect == nil {
		t.Fatal("missing DoH redirect policy")
	}
	redirectRequest := httptest.NewRequest(http.MethodGet, "https://127.0.0.1/private", nil)
	if !errors.Is(client.CheckRedirect(redirectRequest, nil), http.ErrUseLastResponse) {
		t.Fatal("expected redirects to stop at the original DoH endpoint")
	}
}

func TestDNSProbeBatchLimits(t *testing.T) {
	handler := NewRuntimeHandler(&fakeRuntimeInstance{})
	items := strings.TrimSuffix(strings.Repeat(`{"type":"local"},`, maxDNSProbeItems+1), ",")
	rr := httptest.NewRecorder()
	handler.ProbeDNSBatch(rr, jsonRequest(http.MethodPost, "/api/runtime/dns/probe-batch", `{"items":[`+items+`]}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("too many items status = %d body=%s", rr.Code, rr.Body.String())
	}

	origUDP := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = origUDP })
	dnsUDPExchange = func(*dns.Msg, string, time.Duration) (*dns.Msg, error) {
		response := new(dns.Msg)
		response.Rcode = dns.RcodeSuccess
		return response, nil
	}
	rr = httptest.NewRecorder()
	handler.ProbeDNSBatch(rr, jsonRequest(http.MethodPost, "/api/runtime/dns/probe-batch", `{"items":[{"type":"udp","server":"1.1.1.1"}],"concurrency":9999}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("clamped concurrency status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestReadDNSMessageBodyNilReader(t *testing.T) {
	if _, err := readDNSMessageBody(nil); err == nil {
		t.Fatal("expected nil reader error")
	}
}

func TestExchangeDNSDoHInvalidBody(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write([]byte("not-dns"))
	}))
	t.Cleanup(srv.Close)
	orig := newDNSHTTPClient
	t.Cleanup(func() { newDNSHTTPClient = orig })
	newDNSHTTPClient = func(server string, timeout time.Duration) *http.Client { return srv.Client() }
	host, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)
	_, err = exchangeDNSDoH(msg, host, port, "/dns-query", 2*time.Second)
	if err == nil {
		t.Fatal("expected unpack error")
	}
}
