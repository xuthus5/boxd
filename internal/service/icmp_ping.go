package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// icmpProbeTimeout 单次 ICMP 探测超时。
const icmpProbeTimeout = 3 * time.Second

// ICMPEcho 可注入的 ICMP echo 实现，便于测试替换。
var ICMPEcho = func(ctx context.Context, target string) (float64, error) {
	return probeICMPEcho(ctx, target)
}

// PingCommandOutput 保留 ping 命令回退路径（原始 socket 无权限时使用），可注入便于测试。
var PingCommandOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return commandOutput(ctx, name, args...)
}

// errICMPUnavailable 表示原始 socket 不可用（如非 root），需回退到系统 ping 命令。
var errICMPUnavailable = errors.New("icmp raw socket unavailable")

// ICMPPing 执行一次 ICMP echo 探测，返回毫秒延迟。
// target 为 IP 或域名。原始 socket 不可用时（如非 root），回退到系统 ping 命令。
func ICMPPing(ctx context.Context, target string) (float64, error) {
	latency, err := ICMPEcho(ctx, target)
	if err == nil {
		return latency, nil
	}
	if !errors.Is(err, errICMPUnavailable) {
		return 0, err
	}
	return pingCommandFallback(ctx, target)
}

// pingCommandFallback 用系统 ping 命令测量延迟（兼容非特权环境）。
func pingCommandFallback(ctx context.Context, target string) (float64, error) {
	if err := validatePingTarget(target); err != nil {
		return 0, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, icmpProbeTimeout)
	defer cancel()
	startedAt := time.Now()
	output, err := PingCommandOutput(probeCtx, "ping", "-c", "1", "-W", "3", target)
	latency := time.Since(startedAt).Seconds() * 1000
	if err != nil {
		// 保留 context 取消/超时语义，便于调用方识别。
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return 0, errors.New(message)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if milliseconds, ok := parsePingLatency(line); ok {
			latency = milliseconds
			break
		}
	}
	return latency, nil
}

// probeICMPEcho 用 x/net/icmp 发送单个 echo 请求并测量往返延迟。
func probeICMPEcho(ctx context.Context, target string) (float64, error) {
	if err := validatePingTarget(target); err != nil {
		return 0, err
	}

	ip, err := resolvePingHost(ctx, target)
	if err != nil {
		return 0, err
	}

	network, conn, err := listenICMP(ip)
	if err != nil {
		// 原始 socket 不可用（无 CAP_NET_RAW / 非特权），回退到系统 ping。
		return 0, fmt.Errorf("%w: %v", errICMPUnavailable, err)
	}
	defer func() { _ = conn.Close() }()

	msg := icmp.Message{
		Type: icmpTypeEcho(ip),
		Code: 0,
		Body: &icmp.Echo{
			ID:   time.Now().Nanosecond() & 0xffff,
			Seq:  1,
			Data: []byte("boxd-icmp-probe"),
		},
	}

	wire, err := msg.Marshal(nil)
	if err != nil {
		return 0, fmt.Errorf("marshal icmp message: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(icmpProbeTimeout)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return 0, err
	}

	start := time.Now()
	if _, err := conn.WriteTo(wire, &net.IPAddr{IP: ip}); err != nil {
		return 0, fmt.Errorf("write icmp echo: %w", err)
	}

	reply := make([]byte, 512)
	for {
		n, peer, err := conn.ReadFrom(reply)
		if err != nil {
			return 0, fmt.Errorf("read icmp reply: %w", err)
		}
		elapsed := time.Since(start)
		parsed, err := parseICMPReply(network, reply[:n])
		if err == nil && parsed && peerMatches(peer, ip) {
			return float64(elapsed.Microseconds()) / 1000.0, nil
		}
		if time.Since(start) > icmpProbeTimeout {
			return 0, errors.New("icmp echo timed out")
		}
	}
}

// listenICMP 按 IP 族打开 ICMP socket。
func listenICMP(ip net.IP) (string, *icmp.PacketConn, error) {
	if ip.To4() != nil {
		conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		return "ip4", conn, err
	}
	conn, err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	return "ip6", conn, err
}

// icmpTypeEcho 返回对应 IP 族的 echo 请求类型。
func icmpTypeEcho(ip net.IP) icmp.Type {
	if ip.To4() != nil {
		return ipv4.ICMPTypeEcho
	}
	return ipv6.ICMPTypeEchoRequest
}

// parseICMPReply 解析 ICMP 回复，返回是否为匹配的 echo reply。
func parseICMPReply(network string, data []byte) (bool, error) {
	msg, err := icmp.ParseMessage(parseICMPProtocol(network), data)
	if err != nil {
		return false, err
	}
	switch network {
	case "ip4":
		return msg.Type == ipv4.ICMPTypeEchoReply, nil
	case "ip6":
		return msg.Type == ipv6.ICMPTypeEchoReply, nil
	}
	return false, errors.New("unsupported icmp network")
}

// parseICMPProtocol 返回 icmp 包协议号。
func parseICMPProtocol(network string) int {
	if network == "ip6" {
		return 58 // ICMPv6 protocol number
	}
	return 1 // ICMP protocol number
}

// peerMatches 校验回复来源地址与目标一致。
func peerMatches(peer net.Addr, ip net.IP) bool {
	ipAddr, ok := peer.(*net.IPAddr)
	if !ok {
		return true
	}
	return ipAddr.IP.Equal(ip)
}

// resolvePingHost 解析 IP 或域名。
func resolvePingHost(ctx context.Context, target string) (net.IP, error) {
	if ip := net.ParseIP(target); ip != nil {
		return ip, nil
	}
	resolved, err := net.DefaultResolver.LookupIP(ctx, "ip", target)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", target, err)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("resolve %q: no address", target)
	}
	return resolved[0], nil
}

// validatePingTarget 校验目标格式安全（防注入）。
func validatePingTarget(target string) error {
	if target == "" || len(target) > 253 {
		return errors.New("invalid ping target")
	}
	if strings.ContainsAny(target, ";&|`$(){}\n\r\t ") {
		return errors.New("invalid ping target")
	}
	return nil
}
