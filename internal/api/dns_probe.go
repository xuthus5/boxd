package api

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	defaultDNSProbeDomain  = "cloudflare.com"
	defaultDNSProbeTimeout = 5 * time.Second
)

// DNSProbeRequest 探测单个 DNS 服务器可达性与解析延迟。
type DNSProbeRequest struct {
	Tag        string `json:"tag"`
	Type       string `json:"type"`
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
	Address    string `json:"address"`
	Domain     string `json:"domain"`
	Path       string `json:"path"`
}

// DNSProbeResult 单次 DNS 探测结果。
type DNSProbeResult struct {
	Tag       string   `json:"tag"`
	Type      string   `json:"type"`
	Success   bool     `json:"success"`
	LatencyMs float64  `json:"latency_ms,omitempty"`
	Error     string   `json:"error,omitempty"`
	ErrorCode string   `json:"error_code,omitempty"`
	Domain    string   `json:"domain,omitempty"`
	Answers   []string `json:"answers,omitempty"`
}

// 可注入钩子，便于单测覆盖协议分支而不依赖真实网络。
var (
	dnsUDPExchange  = exchangeDNSUDP
	dnsTCPExchange  = exchangeDNSTCP
	dnsTLSExchange  = exchangeDNSTLS
	dnsQUICExchange = exchangeDNSQUIC
	dnsDoHExchange  = exchangeDNSDoH
	dnsH3Exchange   = exchangeDNSHTTP3
)

func probeDNSServer(ctx context.Context, req DNSProbeRequest) DNSProbeResult {
	tag := firstNonEmpty(req.Tag, req.Server)
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		domain = defaultDNSProbeDomain
	}
	result := DNSProbeResult{Tag: tag, Domain: domain}
	if err := validateDNSProbeDomain(domain); err != nil {
		return failedDNSProbeResult(result, err.Error(), err)
	}

	proto, server, port, path, err := normalizeDNSProbeTarget(req)
	if err != nil {
		return failedDNSProbeResult(result, err.Error(), err)
	}
	result.Type = proto

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	start := time.Now()
	var resp *dns.Msg
	switch proto {
	case "udp":
		resp, err = dnsUDPExchange(ctx, msg, joinHostPort(server, port), defaultDNSProbeTimeout)
	case "tcp":
		resp, err = dnsTCPExchange(ctx, msg, joinHostPort(server, port), defaultDNSProbeTimeout)
	case "tls":
		resp, err = dnsTLSExchange(ctx, msg, joinHostPort(server, port), server, defaultDNSProbeTimeout)
	case "quic":
		resp, err = dnsQUICExchange(ctx, msg, joinHostPort(server, port), server, defaultDNSProbeTimeout)
	case "https":
		resp, err = dnsDoHExchange(ctx, msg, server, port, path, defaultDNSProbeTimeout)
	case "h3":
		resp, err = dnsH3Exchange(ctx, msg, server, port, path, defaultDNSProbeTimeout)
	default:
		err = fmt.Errorf("dns type %q is not probeable", proto)
	}
	latency := float64(time.Since(start).Milliseconds())
	if latency < 1 && err == nil {
		latency = 1
	}
	if err != nil {
		out := failedDNSProbeResult(result, err.Error(), err)
		out.LatencyMs = latency
		return out
	}
	if resp == nil {
		return failedDNSProbeResult(result, "empty dns response", nil)
	}
	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		out := failedDNSProbeResult(result, fmt.Sprintf("dns rcode %s", dns.RcodeToString[resp.Rcode]), nil)
		out.LatencyMs = latency
		return out
	}
	result.Success = true
	result.LatencyMs = latency
	result.Answers = collectDNSAnswers(resp)
	return result
}

func normalizeDNSProbeTarget(req DNSProbeRequest) (proto, server string, port int, path string, err error) {
	proto = strings.ToLower(strings.TrimSpace(req.Type))
	server = strings.TrimSpace(req.Server)
	port = req.ServerPort
	path = strings.TrimSpace(req.Path)
	if path == "" {
		path = "/dns-query"
	}

	if req.Address != "" || proto == "legacy" || proto == "" {
		return "", "", 0, "", fmt.Errorf("unsupported legacy DNS format; sing-box 1.14 requires type and server")
	}
	if proto == "http3" {
		proto = "h3"
	}
	switch proto {
	case "local", "hosts", "dhcp", "fakeip", "tailscale", "resolved", "mdns", "openvpn", "openconnect":
		return "", "", 0, "", fmt.Errorf("dns type %q is not probeable", proto)
	case "udp", "tcp", "tls", "quic", "https", "h3":
	default:
		return "", "", 0, "", fmt.Errorf("unsupported dns type %q", proto)
	}
	if server == "" {
		return "", "", 0, "", fmt.Errorf("server is required")
	}
	server, err = normalizeDNSProbeServer(server)
	if err != nil {
		return "", "", 0, "", err
	}
	port, err = normalizeDNSProbePort(proto, port)
	if err != nil {
		return "", "", 0, "", err
	}
	if err := validateDNSProbePath(path); err != nil {
		return "", "", 0, "", err
	}
	return proto, server, port, path, nil
}

func defaultDNSPort(proto string) int {
	switch proto {
	case "tls", "quic":
		return 853
	case "https", "h3":
		return 443
	default:
		return 53
	}
}

func joinHostPort(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func collectDNSAnswers(resp *dns.Msg) []string {
	if resp == nil || len(resp.Answer) == 0 {
		return nil
	}
	out := make([]string, 0, len(resp.Answer))
	for _, rr := range resp.Answer {
		out = append(out, rr.String())
	}
	return out
}
