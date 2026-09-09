package core

import (
	"runtime"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type sharedConfigCheckers struct {
	http        configReferenceChecker
	certificate configReferenceChecker
	dns         configReferenceChecker
	outbound    configReferenceChecker
	inbound     configReferenceChecker
	endpoints   map[string]string
}

func checkSharedConfigReferences(report *model.ConfigDiagnostics, cfg map[string]any) {
	clients := diagnosticEntries(cfg, "http_clients")
	providers := diagnosticEntries(cfg, "certificate_providers")
	checkDuplicateTags(report, clients)
	checkDuplicateTags(report, providers)
	checkDuplicateTags(report, diagnosticEntries(cfg, "services"))
	checks := sharedConfigCheckers{
		http:        configReferenceChecker{report: report, code: "unknown_http_client_reference", known: tagSet(clients)},
		certificate: configReferenceChecker{report: report, code: "unknown_certificate_provider_reference", known: tagSet(providers)},
		dns:         configReferenceChecker{report: report, code: "unknown_dns_reference", known: tagSet(diagnosticEntriesFromDNS(cfg))},
		outbound:    configReferenceChecker{report: report, code: "unknown_outbound_reference", known: tagSet(append(diagnosticEntries(cfg, "outbounds"), diagnosticEntries(cfg, "endpoints")...))},
		inbound:     configReferenceChecker{report: report, code: "unknown_inbound_reference", known: tagSet(diagnosticEntries(cfg, "inbounds"))},
		endpoints:   diagnosticEntryTypes(diagnosticEntries(cfg, "endpoints")),
	}
	checks.http.check("route.default_http_client", objectValue(cfg["route"])["default_http_client"])
	for _, entry := range diagnosticObjects(cfg["http_clients"], "http_clients") {
		checkRequiredConfigField(report, entry, "tag")
	}
	for _, entry := range diagnosticConfigObjects(cfg) {
		walkDiagnosticObjects(entry, checks.checkObject)
	}
	checkRemoteRuleSetHTTPClients(report, cfg)
}

func (c sharedConfigCheckers) checkObject(entry diagnosticObject) {
	object := entry.object
	c.http.check(entry.path+".http_client", object["http_client"])
	c.certificate.check(entry.path+".certificate_provider", object["certificate_provider"])
	if strings.HasPrefix(entry.path, "inbounds[") && !strings.Contains(entry.path, "].") {
		c.inbound.check(entry.path+".detour", object["detour"])
	} else if additionalDiagnosticDialer(entry) {
		c.dns.checkDomainResolver(entry.path+".domain_resolver", object["domain_resolver"])
		c.outbound.check(entry.path+".detour", object["detour"])
	}
	if strings.HasSuffix(entry.path, ".tls") && object["certificate_provider"] != nil {
		c.checkCertificateConflict(entry)
	}
	if strings.HasPrefix(entry.path, "certificate_providers[") || strings.HasSuffix(entry.path, ".certificate_provider") {
		if stringValue(object["type"]) == "tailscale" {
			typedEndpointChecker{report: c.certificate.report, types: c.endpoints}.check(entry, "tailscale")
		}
	}
}

func (c sharedConfigCheckers) checkCertificateConflict(entry diagnosticObject) {
	if entry.object["acme"] != nil || boolValue(objectValue(entry.object["reality"])["enabled"]) {
		addDiagnostic(c.certificate.report, "certificate_provider_conflict", model.ConfigDiagnosticSeverityError, entry.path+".certificate_provider", "", "")
	}
}

func checkRemoteRuleSetHTTPClients(report *model.ConfigDiagnostics, cfg map[string]any) {
	route := objectValue(cfg["route"])
	clients, _ := cfg["http_clients"].([]any)
	hasDefault := stringValue(route["default_http_client"]) != "" || len(clients) > 0
	for _, entry := range diagnosticObjects(route["rule_set"], "route.rule_set") {
		if stringValue(entry.object["type"]) != "remote" {
			continue
		}
		_, inlineClient := entry.object["http_client"].(map[string]any)
		hasClient := inlineClient || stringValue(entry.object["http_client"]) != ""
		hasDetour := stringValue(entry.object["download_detour"]) != ""
		if hasClient && hasDetour {
			addDiagnostic(report, "http_client_detour_conflict", model.ConfigDiagnosticSeverityError, entry.path+".http_client", "", "")
		}
		if !hasClient && !hasDetour && !hasDefault {
			addDiagnostic(report, "implicit_http_client", model.ConfigDiagnosticSeverityWarning, entry.path+".http_client", "", "")
		}
	}
}

func checkNetworkNamespaces(report *model.ConfigDiagnostics, cfg map[string]any) {
	checkNetworkNamespacesForPlatform(report, cfg, runtime.GOOS)
}

func checkNetworkNamespacesForPlatform(report *model.ConfigDiagnostics, cfg map[string]any, platform string) {
	entries := diagnosticObjects(cfg["network_namespaces"], "network_namespaces")
	checkDuplicateTags(report, diagnosticEntries(cfg, "network_namespaces"))
	for _, entry := range entries {
		checkRequiredConfigField(report, entry, "tag")
		if kind := stringValue(entry.object["type"]); kind == "" || kind == "default" {
			checkRequiredConfigField(report, entry, "path")
		}
		if platform != "linux" && platform != "android" {
			addDiagnostic(report, "network_namespace_unsupported", model.ConfigDiagnosticSeverityError, entry.path, "", "")
		}
	}
}

func checkRequiredConfigField(report *model.ConfigDiagnostics, entry diagnosticObject, key string) {
	if strings.TrimSpace(stringValue(entry.object[key])) == "" {
		addDiagnostic(report, "missing_required_field", model.ConfigDiagnosticSeverityError, entry.path+"."+key, "", "")
	}
}

func checkDNSEndpointReferences(report *model.ConfigDiagnostics, cfg map[string]any) {
	endpoints := diagnosticEntryTypes(diagnosticEntries(cfg, "endpoints"))
	seen := make(map[string]bool)
	for _, entry := range diagnosticObjects(objectValue(cfg["dns"])["servers"], "dns.servers") {
		kind := stringValue(entry.object["type"])
		expected := map[string]string{"tailscale": "tailscale", "openvpn": "openvpn-client", "openconnect": "openconnect"}[kind]
		if expected == "" {
			continue
		}
		typedEndpointChecker{report: report, types: endpoints}.check(entry, expected)
		tag := stringValue(entry.object["endpoint"])
		if (kind == "openvpn" || kind == "openconnect") && tag != "" && seen[tag] {
			addDiagnostic(report, "duplicate_endpoint_dns", model.ConfigDiagnosticSeverityError, entry.path+".endpoint", tag, "")
		}
		seen[tag] = true
	}
}

type typedEndpointChecker struct {
	report *model.ConfigDiagnostics
	types  map[string]string
}

func (c typedEndpointChecker) check(entry diagnosticObject, expected string) {
	tag := stringValue(entry.object["endpoint"])
	if tag == "" {
		checkRequiredConfigField(c.report, entry, "endpoint")
	} else if c.types[tag] != expected {
		addDiagnostic(c.report, "invalid_endpoint_reference", model.ConfigDiagnosticSeverityError, entry.path+".endpoint", tag, expected)
	}
}

func additionalDiagnosticDialer(entry diagnosticObject) bool {
	if strings.Contains(entry.path, "].") {
		return true
	}
	for _, prefix := range []string{"http_clients[", "services[", "certificate_providers["} {
		if strings.HasPrefix(entry.path, prefix) {
			return true
		}
	}
	return false
}
