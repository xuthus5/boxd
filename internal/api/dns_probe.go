package api

import (
	"fmt"
	"net"
	"net/url"
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
	Domain    string   `json:"domain,omitempty"`
	Answers   []string `json:"answers,omitempty"`
}

// 可注入钩子，便于单测覆盖协议分支而不依赖真实网络。
var (
	dnsUDPExchange = exchangeDNSUDP
	dnsTCPExchange = exchangeDNSTCP
	dnsTLSExchange = exchangeDNSTLS
	dnsDoHExchange = exchangeDNSDoH
)

func probeDNSServer(req DNSProbeRequest) DNSProbeResult {
	tag := firstNonEmpty(req.Tag, req.Server, req.Address)
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		domain = defaultDNSProbeDomain
	}
	result := DNSProbeResult{Tag: tag, Domain: domain}

	proto, server, port, path, err := normalizeDNSProbeTarget(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Type = proto

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	start := time.Now()
	var resp *dns.Msg
	switch proto {
	case "udp":
		resp, err = dnsUDPExchange(msg, joinHostPort(server, port), defaultDNSProbeTimeout)
	case "tcp":
		resp, err = dnsTCPExchange(msg, joinHostPort(server, port), defaultDNSProbeTimeout)
	case "tls", "quic":
		// quic/DoQ 在本路径以 DoT 近似验证主机可达；完整 DoQ 依赖内核。
		resp, err = dnsTLSExchange(msg, joinHostPort(server, port), server, defaultDNSProbeTimeout)
	case "https", "h3":
		resp, err = dnsDoHExchange(msg, server, port, path, defaultDNSProbeTimeout)
	default:
		err = fmt.Errorf("dns type %q is not probeable", proto)
	}
	latency := float64(time.Since(start).Milliseconds())
	if latency < 1 && err == nil {
		latency = 1
	}
	if err != nil {
		result.Error = err.Error()
		result.LatencyMs = latency
		return result
	}
	if resp == nil {
		result.Error = "empty dns response"
		return result
	}
	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		result.Error = fmt.Sprintf("dns rcode %s", dns.RcodeToString[resp.Rcode])
		result.LatencyMs = latency
		return result
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

	if server == "" && strings.TrimSpace(req.Address) != "" {
		proto, server, port, path, err = parseLegacyDNSAddress(req.Address, proto, port, path)
		if err != nil {
			return "", "", 0, "", err
		}
	}
	if proto == "" {
		proto = "udp"
	}
	switch proto {
	case "local", "hosts", "dhcp", "fakeip", "tailscale":
		return "", "", 0, "", fmt.Errorf("dns type %q is not probeable", proto)
	case "udp", "tcp", "tls", "quic", "https", "h3", "legacy":
		if proto == "legacy" {
			proto = "udp"
		}
	default:
		return "", "", 0, "", fmt.Errorf("unsupported dns type %q", proto)
	}
	if server == "" {
		return "", "", 0, "", fmt.Errorf("server is required")
	}
	if port <= 0 {
		port = defaultDNSPort(proto)
	}
	return proto, server, port, path, nil
}

func parseLegacyDNSAddress(address, proto string, port int, path string) (string, string, int, string, error) {
	raw := strings.TrimSpace(address)
	if raw == "" {
		return "", "", 0, "", fmt.Errorf("address is empty")
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "h3://"):
		u, err := url.Parse(raw)
		if err != nil {
			return "", "", 0, "", fmt.Errorf("invalid address: %w", err)
		}
		p := "https"
		if strings.HasPrefix(lower, "h3://") {
			p = "h3"
		}
		host := u.Hostname()
		if host == "" {
			return "", "", 0, "", fmt.Errorf("address host is empty")
		}
		pr := port
		if u.Port() != "" {
			n, convErr := strconv.Atoi(u.Port())
			if convErr != nil {
				return "", "", 0, "", fmt.Errorf("invalid port: %w", convErr)
			}
			pr = n
		}
		pPath := path
		if u.Path != "" && u.Path != "/" {
			pPath = u.Path
		}
		return p, host, pr, pPath, nil
	case strings.HasPrefix(lower, "tls://"):
		return splitSchemeHost("tls", raw[len("tls://"):], port)
	case strings.HasPrefix(lower, "quic://"):
		return splitSchemeHost("quic", raw[len("quic://"):], port)
	case strings.HasPrefix(lower, "tcp://"):
		return splitSchemeHost("tcp", raw[len("tcp://"):], port)
	case strings.HasPrefix(lower, "udp://"):
		return splitSchemeHost("udp", raw[len("udp://"):], port)
	default:
		host, hostPort, splitErr := net.SplitHostPort(raw)
		if splitErr == nil {
			n, convErr := strconv.Atoi(hostPort)
			if convErr != nil {
				return "", "", 0, "", fmt.Errorf("invalid port: %w", convErr)
			}
			if proto == "" {
				proto = "udp"
			}
			return proto, host, n, path, nil
		}
		if proto == "" {
			proto = "udp"
		}
		return proto, raw, port, path, nil
	}
}

func splitSchemeHost(proto, hostport string, port int) (string, string, int, string, error) {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		return "", "", 0, "", fmt.Errorf("address host is empty")
	}
	host, p, err := net.SplitHostPort(hostport)
	if err == nil {
		n, convErr := strconv.Atoi(p)
		if convErr != nil {
			return "", "", 0, "", fmt.Errorf("invalid port: %w", convErr)
		}
		return proto, host, n, "/dns-query", nil
	}
	return proto, hostport, port, "/dns-query", nil
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
