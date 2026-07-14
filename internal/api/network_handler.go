package api

import (
	"net"
	"net/http"
	"os/exec"
	"strings"
)

type NetworkHandler struct{}

type InterfaceInfo struct {
	Name string   `json:"name"`
	IPs  []string `json:"ips"`
}

var defaultRouteOutput = func() ([]byte, error) {
	return exec.Command("ip", "route", "show", "default").Output()
}

var interfaceAddrs = func(name string) ([]net.Addr, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	return iface.Addrs()
}

func NewNetworkHandler() *NetworkHandler {
	return &NetworkHandler{}
}

func (h *NetworkHandler) GetInterfaces(w http.ResponseWriter, r *http.Request) {
	ips := getDefaultRouteIPs()
	writeJSON(w, http.StatusOK, map[string]any{
		"interfaces": []InterfaceInfo{
			{
				Name: "auto",
				IPs:  append(ips, "0.0.0.0", "::"),
			},
		},
	})
}

func getDefaultRouteIPs() []string {
	out, err := defaultRouteOutput()
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(out))
	if len(fields) < 5 {
		return nil
	}
	ifaceName := fields[4]
	addrs, err := interfaceAddrs(ifaceName)
	if err != nil {
		return nil
	}
	var ips []string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP.String()
		if ip != "::1" {
			ips = append(ips, ip)
		}
	}
	return ips
}
