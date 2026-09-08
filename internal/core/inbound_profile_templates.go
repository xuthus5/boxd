package core

import (
	"bytes"
	"encoding/json"
	"runtime"
)

const (
	defaultMixedPort = 1080
	defaultTUNMTU    = 9000
	defaultTUNIPv4   = "172.19.0.1/30"
	defaultTUNIPv6   = "fdfe:dcba:9876::1/126"
)

func mixedInboundFor(caps NetworkCapabilities) map[string]any {
	listen := caps.DefaultListen
	if listen == "" {
		listen = defaultInboundListen(caps.Container)
	}
	return map[string]any{"type": "mixed", "tag": "mixed-in", "listen": listen, "listen_port": defaultMixedPort}
}

func tunInboundTemplate() map[string]any {
	return tunInboundFor(NetworkCapabilities{Platform: runtime.GOOS}, true)
}

func tunInboundFor(caps NetworkCapabilities, ipv6 bool) map[string]any {
	stack := "gvisor"
	if caps.Platform == "linux" {
		stack = "system"
	}
	addresses := []string{defaultTUNIPv4}
	if ipv6 {
		addresses = append(addresses, defaultTUNIPv6)
	}
	return map[string]any{
		"type": "tun", "tag": "tun-in", "interface_name": "boxd0", "address": addresses,
		"mtu": defaultTUNMTU, "auto_route": true, "strict_route": true, "stack": stack,
	}
}

func configureProfileMixed(byTag map[string]map[string]any, order *[]string, profile inboundProfile) error {
	existing, exists := byTag["mixed-in"]
	if exists && profile.options.Mode != "preserve" && existing["type"] != "mixed" {
		return &InboundProfileError{Code: InboundProfileConflict, Message: "mixed-in belongs to another inbound type; rename it before selecting a profile"}
	}
	if exists && profile.options.Mode != "preserve" && managedMixedInbound(existing) {
		byTag["mixed-in"] = mixedInboundFor(profile.caps)
		return nil
	}
	ensureInbound(byTag, order, "mixed-in", mixedInboundFor(profile.caps))
	return nil
}

func configureProfileTUN(byTag map[string]map[string]any, order *[]string, profile inboundProfile) error {
	if profile.options.Mode == "preserve" {
		return nil
	}
	for tag, inbound := range byTag {
		if inbound["type"] == "tun" && (tag != "tun-in" || !managedTUNInbound(inbound)) {
			return &InboundProfileError{Code: InboundProfileConflict, Message: "custom TUN inbound exists; edit it explicitly before changing the inbound profile"}
		}
	}
	existing, exists := byTag["tun-in"]
	if exists && !managedTUNInbound(existing) {
		return &InboundProfileError{Code: InboundProfileConflict, Message: "tun-in contains custom settings; preserve or edit it explicitly"}
	}
	if profile.options.Mode == "proxy" {
		delete(byTag, "tun-in")
		return nil
	}
	if !exists {
		*order = append(*order, "tun-in")
	}
	byTag["tun-in"] = tunInboundFor(profile.caps, profile.ipv6)
	return nil
}

func managedMixedInbound(inbound map[string]any) bool {
	for _, listen := range []string{"127.0.0.1", "0.0.0.0", "::"} {
		if equalInboundJSON(inbound, mixedInboundFor(NetworkCapabilities{DefaultListen: listen})) {
			return true
		}
	}
	return false
}

func managedTUNInbound(inbound map[string]any) bool {
	for _, platform := range []string{"linux", "windows"} {
		for _, ipv6 := range []bool{true, false} {
			if equalInboundJSON(inbound, tunInboundFor(NetworkCapabilities{Platform: platform}, ipv6)) {
				return true
			}
		}
	}
	return false
}

func equalInboundJSON(left, right map[string]any) bool {
	leftJSON, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightJSON, err := json.Marshal(right)
	return err == nil && bytes.Equal(leftJSON, rightJSON)
}
