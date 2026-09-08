//go:build integration && linux

package core

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"golang.org/x/sys/unix"
)

const (
	tunSmokeUplink    = "boxd-smoke0"
	tunSmokeUnknown   = "boxd-startup-probe.invalid."
	tunSmokeLogBuffer = 256
	tunSmokeTimeout   = time.Second
	tunSmokePacketMax = 65536
)

// 仅运行于 unshare --net 创建的独立命名空间，并显式设置 BOXD_TUN_SMOKE=1。
// 编译时使用 integration 与 Makefile 中全部生产 build tags。
func TestInitialConfigTUNNamespaceSmoke(t *testing.T) {
	requireTUNSmokeNamespace(t)
	setupTUNSmokeNetwork(t)
	packetFD := captureTUNSmokeUplink(t)
	verifyTUNSmokeCapture(t, packetFD)
	path := initializeTUNSmokeConfig(t)
	logs := NewLogWriter(tunSmokeLogBuffer)
	instance := NewSBInstance(path, logs)
	if err := instance.Start(); err != nil {
		t.Fatalf("start real default config: %v; logs=%v", err, logs.Recent())
	}
	t.Cleanup(func() {
		if err := instance.Stop(); err != nil {
			t.Errorf("stop kernel: %v", err)
		}
	})
	assertTUNSmokeRuntime(t, instance)
	for _, mode := range []string{"Rule", "Global"} {
		if _, err := instance.SetClashMode(mode); err != nil {
			t.Fatal(err)
		}
		for _, network := range []string{"udp", "tcp"} {
			t.Run(mode+"/"+network, func(t *testing.T) { exchangeTUNSmokeDNS(t, network) })
		}
	}
	assertTUNSmokeBlockedDNSLog(t, logs)
	assertTUNSmokeNoUplinkTraffic(t, packetFD)
	if err := instance.Stop(); err != nil {
		t.Fatal(err)
	}
	if _, err := net.InterfaceByName("boxd0"); err == nil {
		t.Fatal("stopping the kernel left its TUN device behind")
	}
}

func requireTUNSmokeNamespace(t *testing.T) {
	t.Helper()
	if os.Getenv("BOXD_TUN_SMOKE") != "1" {
		t.Skip("set BOXD_TUN_SMOKE=1 inside an isolated network namespace")
	}
	current, err := os.Readlink("/proc/self/ns/net")
	if err != nil {
		t.Fatal(err)
	}
	initial, err := os.Readlink("/proc/1/ns/net")
	if err != nil {
		t.Fatal(err)
	}
	if current == initial {
		t.Fatal("refusing to create TUN or change routes in the host network namespace")
	}
	if !bootstrapTUNAvailable() {
		t.Fatal("isolated smoke requires CAP_NET_ADMIN and /dev/net/tun")
	}
	t.Logf("isolated namespace=%s, host namespace=%s", current, initial)
}

func setupTUNSmokeNetwork(t *testing.T) {
	t.Helper()
	commands := [][]string{
		{"link", "set", "lo", "up"},
		{"link", "add", tunSmokeUplink, "type", "dummy"},
		{"address", "add", "192.0.2.1/24", "dev", tunSmokeUplink},
		{"-6", "address", "add", "2001:db8::1/64", "dev", tunSmokeUplink, "nodad"},
		{"link", "set", tunSmokeUplink, "up"},
		{"route", "add", "default", "dev", tunSmokeUplink},
		{"-6", "route", "add", "default", "dev", tunSmokeUplink},
	}
	for _, args := range commands {
		if output, err := exec.CommandContext(t.Context(), "ip", args...).CombinedOutput(); err != nil {
			t.Fatalf("ip %v: %v: %s", args, err, output)
		}
	}
}

func initializeTUNSmokeConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	path, dataDir := filepath.Join(root, "etc", "config.json"), filepath.Join(root, "data")
	if created, err := EnsureDefaultConfig(t.Context(), path, dataDir); err != nil || !created {
		t.Fatalf("initialize defaults: created=%v error=%v", created, err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatal(err)
	}
	route := cfg["route"].(map[string]any)
	if route["auto_detect_interface"] != true || route["final"] != "proxy" {
		t.Fatalf("invalid initial TUN route: %v", route)
	}
	entries := route["rule_set"].([]any)
	if len(entries) != 4 {
		t.Fatalf("rule sets = %d, want four complete local snapshots", len(entries))
	}
	for _, value := range entries {
		entry := value.(map[string]any)
		file := entry["path"].(string)
		if entry["type"] != "local" || filepath.Dir(file) != filepath.Join(dataDir, "rule-sets") {
			t.Fatalf("invalid local snapshot path: %v", entry)
		}
		if info, err := os.Stat(file); err != nil || info.Size() == 0 || info.Mode().Perm() != 0600 {
			t.Fatalf("invalid snapshot %s: info=%v error=%v", file, info, err)
		}
	}
	cache := cfg["experimental"].(map[string]any)["cache_file"].(map[string]any)
	if cache["path"] != filepath.Join(dataDir, "cache.db") || cache["enabled"] != true {
		t.Fatalf("invalid persistent selection cache: %v", cache)
	}
	return path
}

func assertTUNSmokeRuntime(t *testing.T, instance *SBInstance) {
	t.Helper()
	device, err := net.InterfaceByName("boxd0")
	if err != nil || device.Flags&net.FlagUp == 0 {
		t.Fatalf("TUN device unavailable: %v %v", device, err)
	}
	addresses, err := device.Addrs()
	if err != nil {
		t.Fatalf("TUN addresses=%v error=%v", addresses, err)
	}
	actual := make(map[string]bool)
	for _, address := range addresses {
		actual[address.String()] = true
	}
	if !actual["172.19.0.1/30"] || !actual["fdfe:dcba:9876::1/126"] {
		t.Fatalf("TUN is missing configured addresses: %v", addresses)
	}
	mode, err := instance.ClashMode()
	if err != nil || !strings.EqualFold(mode.Mode, "Rule") {
		t.Fatalf("initial mode=%v error=%v", mode, err)
	}
	groups := instance.OutboundGroups()
	if len(groups) != 1 || groups[0].Tag != "proxy" || groups[0].Now != "block" {
		t.Fatalf("initial selector must block without nodes: %v", groups)
	}
	t.Logf("kernel running with TUN=%s addresses=%v mode=%s selected=%s", device.Name, addresses, mode.Mode, groups[0].Now)
}

func exchangeTUNSmokeDNS(t *testing.T, network string) {
	t.Helper()
	client := mdns.Client{Net: network, Timeout: tunSmokeTimeout}
	for _, server := range []string{"198.51.100.53:53", "[2001:db8:1::53]:53"} {
		message := new(mdns.Msg)
		message.SetQuestion("doubleclick.net.", mdns.TypeA)
		response, _, err := client.ExchangeContext(t.Context(), message, server)
		if err != nil || response.Rcode != mdns.RcodeSuccess || len(response.Answer) != 0 {
			t.Fatalf("%s DNS advertisement hijack via %s: response=%v error=%v", network, server, response, err)
		}
		message.SetQuestion(tunSmokeUnknown, mdns.TypeA)
		response, _, err = client.ExchangeContext(t.Context(), message, server)
		if err == nil || response != nil {
			t.Fatalf("%s DNS unexpectedly resolved without a proxy: response=%v error=%v", network, response, err)
		}
		t.Logf("%s %s: advertisement answered locally; unknown blocked: %v", network, server, err)
	}
}

func assertTUNSmokeBlockedDNSLog(t *testing.T, logs *LogWriter) {
	t.Helper()
	for _, entry := range logs.Recent() {
		if strings.Contains(entry.Message, tunSmokeUnknown) && strings.Contains(entry.Message, "exchange failed") {
			return
		}
	}
	t.Fatalf("unknown DNS requests never reached the actual DNS router: %v", logs.Recent())
}

func captureTUNSmokeUplink(t *testing.T) int {
	t.Helper()
	device, err := net.InterfaceByName(tunSmokeUplink)
	if err != nil {
		t.Fatal(err)
	}
	var protocolBytes [2]byte
	binary.BigEndian.PutUint16(protocolBytes[:], unix.ETH_P_ALL)
	protocol := binary.NativeEndian.Uint16(protocolBytes[:])
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC, int(protocol))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := unix.Close(fd); err != nil {
			t.Errorf("close packet capture: %v", err)
		}
	})
	if err := unix.Bind(fd, &unix.SockaddrLinklayer{Ifindex: device.Index, Protocol: protocol}); err != nil {
		t.Fatal(err)
	}
	return fd
}

func assertTUNSmokeNoUplinkTraffic(t *testing.T, fd int) {
	t.Helper()
	packet := make([]byte, tunSmokePacketMax)
	for {
		n, _, err := unix.Recvfrom(fd, packet, unix.MSG_DONTWAIT)
		if errors.Is(err, unix.EAGAIN) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if tunSmokeTransportPacket(packet[:n]) {
			t.Fatalf("unexpected TCP/UDP egress on isolated uplink: %x", packet[:n])
		}
	}
	t.Log("uplink TCP/UDP packets=0; no plaintext DNS or direct DoH fallback escaped")
}

func verifyTUNSmokeCapture(t *testing.T, fd int) {
	t.Helper()
	conn, err := net.Dial("udp", "198.51.100.53:53")
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := conn.Write([]byte("capture-control"))
	if err := errors.Join(writeErr, conn.Close()); err != nil {
		t.Fatal(err)
	}
	ready, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, int(tunSmokeTimeout/time.Millisecond))
	if err != nil || ready == 0 {
		t.Fatalf("capture did not observe the UDP53 control packet: ready=%d error=%v", ready, err)
	}
	packet, observed := make([]byte, tunSmokePacketMax), false
	for {
		n, _, err := unix.Recvfrom(fd, packet, unix.MSG_DONTWAIT)
		if errors.Is(err, unix.EAGAIN) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		observed = observed || tunSmokeTransportPacket(packet[:n])
	}
	if !observed {
		t.Fatal("capture cannot detect TCP/UDP traffic")
	}
	t.Log("capture verified with an isolated UDP53 control packet before kernel startup")
}

func tunSmokeTransportPacket(packet []byte) bool {
	const ipv4Header, ipv6Header = 20, 40
	if len(packet) < ipv4Header {
		return false
	}
	if packet[0]>>4 == 4 {
		return packet[9] == unix.IPPROTO_TCP || packet[9] == unix.IPPROTO_UDP
	}
	if packet[0]>>4 == 6 && len(packet) >= ipv6Header {
		return packet[6] == unix.IPPROTO_TCP || packet[6] == unix.IPPROTO_UDP
	}
	return false
}
