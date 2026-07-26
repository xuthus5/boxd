package core

import (
	"net/netip"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func checkSingBoxMigrationWarnings(
	report *model.ConfigDiagnostics,
	cfg map[string]any,
	dnsServers []diagnosticEntry,
) {
	checkLegacyDNSServers(report, cfg)
	checkLegacyDNSFakeIP(report, cfg)
	checkOutboundDNSRuleItems(report, cfg)
	checkLegacyDomainStrategies(report, cfg)
	checkMissingDomainResolvers(report, cfg, dnsServers)
}

func checkLegacyDNSServers(report *model.ConfigDiagnostics, cfg map[string]any) {
	dns := objectValue(cfg["dns"])
	servers, _ := dns["servers"].([]any)
	for index, item := range servers {
		object := objectValue(item)
		typeName := strings.ToLower(strings.TrimSpace(stringValue(object["type"])))
		if typeName != "" && typeName != "legacy" {
			continue
		}
		path := "dns.servers[" + strconv.Itoa(index) + "]"
		addDiagnostic(
			report,
			"legacy_dns_server",
			model.ConfigDiagnosticSeverityWarning,
			path,
			stringValue(object["tag"]),
			"",
		)
	}
}

func checkLegacyDNSFakeIP(report *model.ConfigDiagnostics, cfg map[string]any) {
	dns := objectValue(cfg["dns"])
	fakeIP := objectValue(dns["fakeip"])
	enabled, _ := fakeIP["enabled"].(bool)
	if enabled {
		addDiagnostic(report, "legacy_dns_fakeip", model.ConfigDiagnosticSeverityWarning, "dns.fakeip", "", "")
	}
}

func checkOutboundDNSRuleItems(report *model.ConfigDiagnostics, cfg map[string]any) {
	dns := objectValue(cfg["dns"])
	checkOutboundDNSRuleItemsAtPath(report, dns["rules"], "dns.rules")
}

func checkOutboundDNSRuleItemsAtPath(report *model.ConfigDiagnostics, value any, path string) {
	rules, _ := value.([]any)
	for index, item := range rules {
		rule := objectValue(item)
		rulePath := path + "[" + strconv.Itoa(index) + "]"
		if outbound := stringValues(rule["outbound"]); len(outbound) > 0 {
			addDiagnostic(
				report,
				"outbound_dns_rule_item",
				model.ConfigDiagnosticSeverityWarning,
				rulePath+".outbound",
				strings.Join(outbound, ", "),
				"",
			)
		}
		checkOutboundDNSRuleItemsAtPath(report, rule["rules"], rulePath+".rules")
	}
}

func checkLegacyDomainStrategies(report *model.ConfigDiagnostics, cfg map[string]any) {
	for _, sectionKey := range []string{"outbounds", "endpoints"} {
		checkLegacyDomainStrategySection(report, cfg[sectionKey], sectionKey)
	}
	dns := objectValue(cfg["dns"])
	checkLegacyDomainStrategySection(report, dns["servers"], "dns.servers")
	route := objectValue(cfg["route"])
	checkLegacyDomainStrategyRules(report, route["rules"], "route.rules")
}

func checkLegacyDomainStrategySection(report *model.ConfigDiagnostics, value any, path string) {
	items, _ := value.([]any)
	for index, item := range items {
		object := objectValue(item)
		strategy := strings.TrimSpace(stringValue(object["domain_strategy"]))
		if strategy == "" {
			continue
		}
		itemPath := path + "[" + strconv.Itoa(index) + "]"
		addDiagnostic(
			report,
			"legacy_domain_strategy",
			model.ConfigDiagnosticSeverityWarning,
			itemPath+".domain_strategy",
			stringValue(object["tag"]),
			"",
		)
	}
}

func checkLegacyDomainStrategyRules(report *model.ConfigDiagnostics, value any, path string) {
	rules, _ := value.([]any)
	for index, item := range rules {
		rule := objectValue(item)
		rulePath := path + "[" + strconv.Itoa(index) + "]"
		isDirect := strings.EqualFold(stringValue(rule["action"]), "direct")
		hasStrategy := strings.TrimSpace(stringValue(rule["domain_strategy"])) != ""
		if isDirect && hasStrategy {
			addDiagnostic(
				report,
				"legacy_domain_strategy",
				model.ConfigDiagnosticSeverityWarning,
				rulePath+".domain_strategy",
				"",
				"",
			)
		}
		checkLegacyDomainStrategyRules(report, rule["rules"], rulePath+".rules")
	}
}

func checkMissingDomainResolvers(
	report *model.ConfigDiagnostics,
	cfg map[string]any,
	dnsServers []diagnosticEntry,
) {
	if len(dnsServers) < 2 {
		return
	}
	route := objectValue(cfg["route"])
	if hasDomainResolver(route["default_domain_resolver"]) {
		return
	}
	for _, sectionKey := range []string{"outbounds", "endpoints"} {
		checkMissingDomainResolversInSection(report, cfg[sectionKey], sectionKey)
	}
}

func checkMissingDomainResolversInSection(report *model.ConfigDiagnostics, value any, sectionKey string) {
	items, _ := value.([]any)
	for index, item := range items {
		object := objectValue(item)
		server := strings.TrimSpace(stringValue(object["server"]))
		hasResolver := hasDomainResolver(object["domain_resolver"])
		hasDetour := strings.TrimSpace(stringValue(object["detour"])) != ""
		if !isDomainName(server) || hasResolver || hasDetour {
			continue
		}
		path := sectionKey + "[" + strconv.Itoa(index) + "].server"
		addDiagnostic(
			report,
			"missing_domain_resolver",
			model.ConfigDiagnosticSeverityWarning,
			path,
			stringValue(object["tag"]),
			"",
		)
	}
}

func hasDomainResolver(value any) bool {
	switch resolver := value.(type) {
	case string:
		return strings.TrimSpace(resolver) != ""
	case map[string]any:
		return strings.TrimSpace(stringValue(resolver["server"])) != ""
	default:
		return false
	}
}

func isDomainName(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	_, err := netip.ParseAddr(value)
	return err != nil
}
