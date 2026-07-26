package core

import (
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func checkDuplicateTags(report *model.ConfigDiagnostics, entries []diagnosticEntry) {
	seen := make(map[string]diagnosticEntry, len(entries))
	for _, entry := range entries {
		if entry.tag == "" {
			continue
		}
		if _, exists := seen[entry.tag]; exists {
			addDiagnostic(report, "duplicate_tag", model.ConfigDiagnosticSeverityError, entry.path, entry.tag, "")
			continue
		}
		seen[entry.tag] = entry
	}
}

func checkOutboundReferences(report *model.ConfigDiagnostics, cfg map[string]any, outbounds, endpoints, ruleSets []diagnosticEntry) {
	known, allOutbounds := outboundReferenceSets(outbounds, endpoints)
	knownRuleSets := tagSet(ruleSets)
	route := objectValue(cfg["route"])
	if route != nil {
		checkReference(report, "unknown_outbound_reference", "route.final", stringValue(route["final"]), known)
		if rules, ok := route["rules"].([]any); ok {
			for index, item := range rules {
				path := "route.rules[" + strconv.Itoa(index) + "]"
				checkRuleReferences(report, objectValue(item), path, known, knownRuleSets)
			}
		}
	}
	for _, entry := range allOutbounds {
		object := objectAtPath(cfg, entry.path)
		checkReference(report, "unknown_outbound_reference", entry.path+".detour", stringValue(object["detour"]), known)
		for _, reference := range stringValues(object["outbounds"]) {
			checkReference(report, "unknown_outbound_reference", entry.path+".outbounds", reference, known)
		}
	}
}

func outboundReferenceSets(outbounds, endpoints []diagnosticEntry) (map[string]struct{}, []diagnosticEntry) {
	entries := make([]diagnosticEntry, 0, len(outbounds)+len(endpoints))
	entries = append(entries, outbounds...)
	entries = append(entries, endpoints...)
	return tagSet(entries), entries
}

func tagSet(entries []diagnosticEntry) map[string]struct{} {
	known := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.tag != "" {
			known[entry.tag] = struct{}{}
		}
	}
	return known
}

func checkRuleReferences(report *model.ConfigDiagnostics, rule map[string]any, path string, known, knownRuleSets map[string]struct{}) {
	if rule == nil {
		return
	}
	for _, reference := range stringValues(rule["outbound"]) {
		checkReference(report, "unknown_outbound_reference", path+".outbound", reference, known)
	}
	for _, reference := range stringValues(rule["rule_set"]) {
		checkReference(report, "unknown_ruleset_reference", path+".rule_set", reference, knownRuleSets)
	}
	if rules, ok := rule["rules"].([]any); ok {
		for index, item := range rules {
			nestedPath := path + ".rules[" + strconv.Itoa(index) + "]"
			checkRuleReferences(report, objectValue(item), nestedPath, known, knownRuleSets)
		}
	}
}

func checkDNSReferences(report *model.ConfigDiagnostics, cfg map[string]any, servers []diagnosticEntry) {
	known := tagSet(servers)
	dns := objectValue(cfg["dns"])
	if dns == nil {
		return
	}
	checkReference(report, "unknown_dns_reference", "dns.final", stringValue(dns["final"]), known)
	if rules, ok := dns["rules"].([]any); ok {
		for index, item := range rules {
			rule := objectValue(item)
			for _, reference := range stringValues(rule["server"]) {
				path := "dns.rules[" + strconv.Itoa(index) + "].server"
				checkReference(report, "unknown_dns_reference", path, reference, known)
			}
		}
	}
}

func checkReference(report *model.ConfigDiagnostics, code, path, reference string, known map[string]struct{}) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return
	}
	if _, exists := known[reference]; !exists {
		addDiagnostic(report, code, model.ConfigDiagnosticSeverityError, path, reference, "")
	}
}

func checkInsecureTLS(report *model.ConfigDiagnostics, cfg map[string]any) {
	for _, sectionKey := range []string{"inbounds", "outbounds", "endpoints"} {
		items, _ := cfg[sectionKey].([]any)
		for index, item := range items {
			object := objectValue(item)
			tls := objectValue(object["tls"])
			if insecure, _ := tls["insecure"].(bool); insecure {
				path := sectionKey + "[" + strconv.Itoa(index) + "].tls.insecure"
				addDiagnostic(report, "tls_insecure", model.ConfigDiagnosticSeverityWarning, path, stringValue(object["tag"]), "")
			}
		}
	}
}

func setConfigFeatures(report *model.ConfigDiagnostics, cfg map[string]any, inbounds, outbounds, endpoints, ruleSets []diagnosticEntry) {
	entries := make([]diagnosticEntry, 0, len(inbounds)+len(outbounds)+len(endpoints))
	entries = append(entries, inbounds...)
	entries = append(entries, outbounds...)
	entries = append(entries, endpoints...)
	for _, entry := range entries {
		setEntryFeature(&report.Features, entry.typeName)
	}
	for _, entry := range ruleSets {
		if strings.EqualFold(entry.typeName, "remote") {
			report.Features.RemoteRuleSet = true
		}
	}
	dns := objectValue(cfg["dns"])
	if fakeIP := objectValue(dns["fakeip"]); fakeIP != nil {
		report.Features.FakeIP, _ = fakeIP["enabled"].(bool)
	}
	experimental := objectValue(cfg["experimental"])
	if cache := objectValue(experimental["cache_file"]); cache != nil {
		report.Features.CacheFile, _ = cache["enabled"].(bool)
	}
	report.Features.ClashAPI = objectValue(experimental["clash_api"]) != nil
}

func setEntryFeature(features *model.ConfigDiagnosticsFeatures, typeName string) {
	switch strings.ToLower(typeName) {
	case "tun":
		features.TUN = true
	case "selector":
		features.Selector = true
	case "urltest":
		features.URLTest = true
	case "wireguard":
		features.WireGuard = true
	}
}
