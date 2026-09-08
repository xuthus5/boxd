package core

import (
	"net"
	"os"
	"runtime"
	"strings"
)

// 网络能力原因码；未知状态与明确禁用分开，供界面说明接管范围。
const (
	NetworkTUNPermission   = "tun_permission_required"
	NetworkTUNDevice       = "tun_device_unavailable"
	NetworkTUNUnknown      = "tun_unknown"
	NetworkTUNUnsupported  = "tun_unsupported"
	NetworkIPv6Disabled    = "ipv6_disabled"
	NetworkIPv6Unknown     = "ipv6_unknown"
	NetworkIPv6Unsupported = "ipv6_unsupported"
)

// NetworkCapabilities 描述当前进程可用的网络能力，不探测公网连接。
type NetworkCapabilities struct {
	Platform      string `json:"platform"`
	Container     bool   `json:"container"`
	TUNAvailable  bool   `json:"tun_available"`
	TUNReason     string `json:"tun_reason"`
	IPv6Available bool   `json:"ipv6_available"`
	IPv6Reason    string `json:"ipv6_reason"`
	DefaultListen string `json:"default_listen"`
}

// DetectNetworkCapabilities 只读取环境与系统状态，不创建 TUN 或改变路由。
func DetectNetworkCapabilities() NetworkCapabilities {
	caps := platformNetworkCapabilities()
	caps.Platform = runtime.GOOS
	caps.Container = detectContainer(os.Getenv, os.Stat)
	caps.DefaultListen = defaultInboundListen(caps.Container)
	return caps
}

func detectContainer(getenv func(string) string, stat func(string) (os.FileInfo, error)) bool {
	value := strings.ToLower(strings.TrimSpace(getenv("BOXD_CONTAINER")))
	if value == "true" || value == "1" {
		return true
	}
	for _, path := range []string{"/run/.containerenv", "/.dockerenv"} {
		if _, err := stat(path); err == nil {
			return true
		}
	}
	return false
}

func defaultInboundListen(container bool) string {
	if container {
		return "0.0.0.0"
	}
	return "127.0.0.1"
}

func probeIPv6Socket() error {
	socket, err := net.ListenPacket("udp6", "[::]:0")
	if err != nil {
		return err
	}
	return socket.Close()
}
