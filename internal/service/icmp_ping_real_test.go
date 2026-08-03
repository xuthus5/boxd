package service

import (
	"context"
	"net"
	"testing"
)

// TestProbeICMPEchoLocalhost 验证真实 ICMP echo 往返（需原始 socket 权限）。
// 非特权环境（如 CI 容器）自动跳过。
func TestProbeICMPEchoLocalhost(t *testing.T) {
	latency, err := probeICMPEcho(context.Background(), "127.0.0.1")
	if err != nil {
		t.Skipf("icmp raw socket unavailable: %v", err)
	}
	if latency <= 0 {
		t.Fatalf("latency = %v", latency)
	}
}

// TestProbeICMPEchoInvalidTarget 验证非法目标在开 socket 前被拒绝。
func TestProbeICMPEchoInvalidTarget(t *testing.T) {
	if _, err := probeICMPEcho(context.Background(), "bad;host"); err == nil {
		t.Fatal("expected error for invalid target")
	}
}

// TestListenICMPIPv4 验证 IPv4 socket 打开（非特权环境跳过）。
func TestListenICMPIPv4(t *testing.T) {
	network, conn, err := listenICMP(net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Skipf("icmp raw socket unavailable: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if network != "ip4" {
		t.Fatalf("network = %q", network)
	}
}

// TestListenICMPIPv6 验证 IPv6 socket 打开（非特权环境跳过）。
func TestListenICMPIPv6(t *testing.T) {
	network, conn, err := listenICMP(net.ParseIP("::1"))
	if err != nil {
		t.Skipf("icmp6 raw socket unavailable: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if network != "ip6" {
		t.Fatalf("network = %q", network)
	}
}

// TestResolvePingHost 验证域名解析与无效域名错误。
func TestResolvePingHost(t *testing.T) {
	ip, err := resolvePingHost(context.Background(), "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if ip == nil {
		t.Fatal("nil ip")
	}
	if _, err := resolvePingHost(context.Background(), "localhost"); err != nil {
		t.Fatal(err)
	}
	if _, err := resolvePingHost(context.Background(), "nonexistent.invalid."); err == nil {
		t.Skip("unexpectedly resolved invalid domain")
	}
}
