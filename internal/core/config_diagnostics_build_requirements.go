package core

import "strings"

type configBuildRequirement struct {
	path    string
	feature string
	tag     string
}

// 对照 sing-box 1.14 include/*_stub.go；注册了 JSON options 不代表构造器已编译。
var configTypeBuildTags = map[string]string{
	"inbounds/hysteria": "with_quic", "inbounds/hysteria2": "with_quic", "inbounds/tuic": "with_quic",
	"inbounds/cloudflared": "with_cloudflared",
	"outbounds/hysteria":   "with_quic", "outbounds/hysteria2": "with_quic", "outbounds/tuic": "with_quic",
	"outbounds/naive":     "with_naive_outbound",
	"endpoints/wireguard": "with_wireguard", "endpoints/tailscale": "with_tailscale",
	"endpoints/openvpn-client": "with_openvpn", "endpoints/openvpn-server": "with_openvpn",
	"endpoints/openconnect": "with_openconnect",
	"dns.servers/quic":      "with_quic", "dns.servers/h3": "with_quic", "dns.servers/dhcp": "with_dhcp",
	"dns.servers/tailscale": "with_tailscale", "dns.servers/openvpn": "with_openvpn",
	"dns.servers/openconnect": "with_openconnect",
	"services/derp":           "with_tailscale", "services/hysteria-realm": "with_quic",
	"services/ccm": "with_ccm", "services/ocm": "with_ocm",
	"services/usbip-client": "with_usbip", "services/usbip-server": "with_usbip",
	"certificate_providers/acme": "with_acme", "certificate_providers/tailscale": "with_tailscale",
}

func configBuildRequirements(cfg map[string]any) []configBuildRequirement {
	var requirements []configBuildRequirement
	for _, entry := range diagnosticConfigObjects(cfg) {
		section, _, _ := strings.Cut(entry.path, "[")
		requirements = append(requirements, configTypeBuildRequirements(section, entry)...)
		walkDiagnosticObjects(entry, func(nested diagnosticObject) {
			requirements = append(requirements, nestedBuildRequirements(nested)...)
		})
		if section == "http_clients" {
			requirements = append(requirements, httpEngineBuildRequirements(entry)...)
		}
		transport := objectValue(entry.object["transport"])
		if stringValue(transport["type"]) == "quic" {
			requirements = append(requirements, configBuildRequirement{entry.path + ".transport.type", "quic", "with_quic"})
		}
	}
	return requirements
}

func configTypeBuildRequirements(section string, entry diagnosticObject) []configBuildRequirement {
	feature := stringValue(entry.object["type"])
	var requirements []configBuildRequirement
	if tag := configTypeBuildTags[section+"/"+feature]; tag != "" {
		requirements = append(requirements, configBuildRequirement{entry.path + ".type", feature, tag})
	}
	if section == "endpoints" && (feature == "openvpn-client" || feature == "openvpn-server" || feature == "openconnect") && !boolValue(entry.object["system"]) {
		requirements = append(requirements, configBuildRequirement{entry.path + ".system", "gvisor", "with_gvisor"})
	}
	// Naive 默认 TCP+UDP 可以退回 TCP；仅 UDP 时缺少 QUIC 才是启动错误。
	if section == "inbounds" && feature == "naive" && stringValue(entry.object["network"]) == "udp" {
		requirements = append(requirements, configBuildRequirement{entry.path + ".network", "quic", "with_quic"})
	}
	return requirements
}

func nestedBuildRequirements(entry diagnosticObject) []configBuildRequirement {
	if strings.HasSuffix(entry.path, ".certificate_provider") {
		return configTypeBuildRequirements("certificate_providers", entry)
	}
	if strings.HasSuffix(entry.path, ".http_client") {
		return httpEngineBuildRequirements(entry)
	}
	if !strings.HasSuffix(entry.path, ".tls") || !boolValue(entry.object["enabled"]) {
		return nil
	}
	var requirements []configBuildRequirement
	if boolValue(objectValue(entry.object["utls"])["enabled"]) {
		requirements = append(requirements, configBuildRequirement{entry.path + ".utls.enabled", "utls", "with_utls"})
	}
	if stringValue(entry.object["engine"]) == "apple" {
		requirements = append(requirements, configBuildRequirement{path: entry.path + ".engine", feature: "apple"})
	}
	return requirements
}

func httpEngineBuildRequirements(entry diagnosticObject) []configBuildRequirement {
	if stringValue(entry.object["engine"]) != "apple" {
		return nil
	}
	return []configBuildRequirement{{path: entry.path + ".engine", feature: "apple"}}
}
