package service

import (
	"context"
	"net"
	"sort"
	"strings"
)

// InterfaceInfo 网卡信息。
type InterfaceInfo struct {
	Name string   `json:"name"`
	IPs  []string `json:"ips"`
}

// Network 提供网卡相关用例逻辑。
type Network struct{}

// 可注入钩子，便于单测覆盖而不依赖真实网卡。
var (
	listInterfaces = net.Interfaces
	interfaceAddrs = func(iface net.Interface) ([]net.Addr, error) {
		return iface.Addrs()
	}
)

// ListInterfaces 列出非回环网卡及其非回环 IP。
func (s *Network) ListInterfaces(_ context.Context) ([]InterfaceInfo, error) {
	ifaces, err := listInterfaces()
	if err != nil {
		return nil, Errorf(500, "internal_error", "failed to list network interfaces")
	}

	result := make([]InterfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		name := strings.TrimSpace(iface.Name)
		if name == "" || name == "lo" {
			continue
		}
		addrs, err := interfaceAddrs(iface)
		if err != nil {
			continue
		}
		ips := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}
			ip := ipNet.IP
			if ip.IsLoopback() {
				continue
			}
			ips = append(ips, ip.String())
		}
		result = append(result, InterfaceInfo{Name: name, IPs: ips})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}
