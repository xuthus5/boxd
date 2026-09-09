package core

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type diagnosticObject struct {
	path   string
	object map[string]any
}

func checkSingBox114Diagnostics(report *model.ConfigDiagnostics, cfg map[string]any) {
	checkDNSResponseRules(report, cfg)
	checkSharedConfigReferences(report, cfg)
	checkNetworkNamespaces(report, cfg)
	checkRuleSetTagOptions(report, cfg)
	checkDNSEndpointReferences(report, cfg)
	checkPreferredByReferences(report, cfg)
	checkNestedRuleActions(report, cfg)
}

// 只遍历已知结构，不能把 headers、凭据字典中的同名键当成配置引用。
func diagnosticConfigObjects(cfg map[string]any) []diagnosticObject {
	var entries []diagnosticObject
	for _, key := range []string{"inbounds", "outbounds", "endpoints", "services", "http_clients", "certificate_providers"} {
		entries = append(entries, diagnosticObjects(cfg[key], key)...)
	}
	entries = append(entries, diagnosticObjects(objectValue(cfg["dns"])["servers"], "dns.servers")...)
	entries = append(entries, diagnosticObjects(objectValue(cfg["route"])["rule_set"], "route.rule_set")...)
	if ntp := objectValue(cfg["ntp"]); ntp != nil {
		entries = append(entries, diagnosticObject{path: "ntp", object: ntp})
	}
	return entries
}

func diagnosticObjects(value any, path string) []diagnosticObject {
	items, _ := value.([]any)
	entries := make([]diagnosticObject, 0, len(items))
	for index, item := range items {
		if object := objectValue(item); object != nil {
			entries = append(entries, diagnosticObject{
				path: path + "[" + strconv.Itoa(index) + "]", object: object,
			})
		}
	}
	return entries
}

func walkDiagnosticObjects(entry diagnosticObject, visit func(diagnosticObject)) {
	visit(entry)
	for _, key := range []string{"tls", "ech", "reality", "handshake", "certificate_provider", "http_client", "masquerade", "realm", "control", "tunnel"} {
		if object := objectValue(entry.object[key]); object != nil {
			walkDiagnosticObjects(diagnosticObject{path: entry.path + "." + key, object: object}, visit)
		}
	}
}

func checkRemovedConfigFields(report *model.ConfigDiagnostics, cfg map[string]any) {
	for _, entry := range diagnosticObjects(cfg["inbounds"], "inbounds") {
		checkRemovedInboundFields(report, entry)
	}
	for _, entry := range diagnosticConfigObjects(cfg) {
		walkDiagnosticObjects(entry, func(item diagnosticObject) {
			checkRemovedTLSFields(report, item)
		})
	}
	for _, section := range []string{"route", "dns"} {
		object := objectValue(cfg[section])
		checkFieldReplacements(report, diagnosticObject{path: section, object: object}, map[string]string{
			"geoip": "route.rule_set", "geosite": "route.rule_set",
		})
		walkDiagnosticRules(object["rules"], section+".rules", func(entry diagnosticObject) {
			checkFieldReplacements(report, entry, map[string]string{
				"geoip": "rule_set", "source_geoip": "rule_set", "geosite": "rule_set",
				"rule_set_ipcidr_match_source": "rule_set_ip_cidr_match_source",
			})
		})
	}
	for _, entry := range diagnosticObjects(cfg["outbounds"], "outbounds") {
		if replacement, removed := removedOutboundReplacement(stringValue(entry.object["type"])); removed {
			addDiagnostic(report, "removed_config_field", model.ConfigDiagnosticSeverityError, entry.path+".type", "", replacement)
		}
		if stringValue(entry.object["type"]) == "direct" {
			checkFieldReplacements(report, entry, map[string]string{
				"override_address": "route.rules[].override_address", "override_port": "route.rules[].override_port",
			})
		}
	}
}

func removedOutboundReplacement(kind string) (string, bool) {
	switch kind {
	case "wireguard":
		return "endpoints[].type=wireguard", true
	case "dns":
		return "route.rules[].action=hijack-dns", true
	case "shadowsocksr":
		return "", true
	default:
		return "", false
	}
}

func checkRemovedInboundFields(report *model.ConfigDiagnostics, entry diagnosticObject) {
	checkFieldReplacements(report, entry, map[string]string{
		"sniff": "route.rules[].action=sniff", "sniff_override_destination": "route.rules[].action=sniff",
		"sniff_timeout": "route.rules[].timeout", "domain_strategy": "route.rules[].action=resolve",
		"udp_disable_domain_unmapping": "route.rules[].udp_disable_domain_unmapping",
		"proxy_protocol":               "", "proxy_protocol_accept_no_header": "",
	})
	if stringValue(entry.object["type"]) != "tun" {
		return
	}
	checkFieldReplacements(report, entry, map[string]string{
		"inet4_address": "address", "inet6_address": "address",
		"inet4_route_address": "route_address", "inet6_route_address": "route_address",
		"inet4_route_exclude_address": "route_exclude_address", "inet6_route_exclude_address": "route_exclude_address",
		"gso": "", "endpoint_independent_nat": "udp_mapping",
	})
}

func checkRemovedTLSFields(report *model.ConfigDiagnostics, entry diagnosticObject) {
	if !strings.HasSuffix(entry.path, ".tls.ech") {
		return
	}
	checkFieldReplacements(report, entry, map[string]string{
		"pq_signature_schemes_enabled": "", "dynamic_record_sizing_disabled": "",
	})
}

func checkFieldReplacements(report *model.ConfigDiagnostics, entry diagnosticObject, replacements map[string]string) {
	for _, key := range slices.Sorted(maps.Keys(replacements)) {
		if _, exists := entry.object[key]; exists {
			addDiagnostic(report, "removed_config_field", model.ConfigDiagnosticSeverityError, entry.path+"."+key, "", replacements[key])
		}
	}
}

func checkDeprecatedConfigFields(report *model.ConfigDiagnostics, cfg map[string]any) {
	checkDeprecatedField(report, diagnosticObject{path: "dns", object: objectValue(cfg["dns"])}, "independent_cache")
	experimental := objectValue(cfg["experimental"])
	checkDeprecatedField(report, diagnosticObject{path: "experimental.cache_file", object: objectValue(experimental["cache_file"])}, "store_rdrc")
	for _, entry := range diagnosticConfigObjects(cfg) {
		walkDiagnosticObjects(entry, func(item diagnosticObject) {
			if strings.HasSuffix(item.path, ".tls") {
				checkDeprecatedField(report, item, "acme")
			}
		})
	}
	for _, entry := range diagnosticObjects(objectValue(cfg["route"])["rule_set"], "route.rule_set") {
		if stringValue(entry.object["type"]) == "remote" {
			checkDeprecatedField(report, entry, "download_detour")
		}
	}
}

func checkDeprecatedField(report *model.ConfigDiagnostics, entry diagnosticObject, key string) {
	if _, exists := entry.object[key]; exists {
		addDiagnostic(report, "deprecated_config_field", model.ConfigDiagnosticSeverityWarning, entry.path+"."+key, "", key)
	}
}

func walkDiagnosticRules(value any, path string, visit func(diagnosticObject)) {
	for _, entry := range diagnosticObjects(value, path) {
		visit(entry)
		walkDiagnosticRules(entry.object["rules"], entry.path+".rules", visit)
	}
}
