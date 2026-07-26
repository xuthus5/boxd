package core

import (
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type configReferenceChecker struct {
	report *model.ConfigDiagnostics
	code   string
	known  map[string]struct{}
}

type routeReferenceCheckers struct {
	outbound configReferenceChecker
	ruleSet  configReferenceChecker
}

type dnsReferenceCheckers struct {
	dns     configReferenceChecker
	ruleSet configReferenceChecker
}

type outboundReferenceEntries struct {
	outbounds []diagnosticEntry
	endpoints []diagnosticEntry
	ruleSets  []diagnosticEntry
}

func checkOutboundReferences(
	report *model.ConfigDiagnostics,
	cfg map[string]any,
	entries outboundReferenceEntries,
) {
	known, allOutbounds := outboundReferenceSets(entries.outbounds, entries.endpoints)
	outboundChecker := configReferenceChecker{
		report: report,
		code:   "unknown_outbound_reference",
		known:  known,
	}
	ruleSetChecker := configReferenceChecker{
		report: report,
		code:   "unknown_ruleset_reference",
		known:  tagSet(entries.ruleSets),
	}
	checkRouteOutboundReferences(cfg, outboundChecker, ruleSetChecker)
	for _, entry := range allOutbounds {
		object := objectAtPath(cfg, entry.path)
		outboundChecker.check(entry.path+".detour", object["detour"])
		outboundChecker.checkList(entry.path+".outbounds", object["outbounds"])
		outboundChecker.check(entry.path+".default", object["default"])
		outboundChecker.checkEmptyGroup(entry.path, object)
	}
	outboundChecker.checkDNSDetours(objectValue(cfg["dns"]))
	outboundChecker.checkNTPDetour(objectValue(cfg["ntp"]))
	outboundChecker.checkExperimentalDetour(objectValue(cfg["experimental"]))
}

func checkDNSReferences(report *model.ConfigDiagnostics, cfg map[string]any, servers []diagnosticEntry) {
	dnsChecker := configReferenceChecker{
		report: report,
		code:   "unknown_dns_reference",
		known:  tagSet(servers),
	}
	ruleSetChecker := configReferenceChecker{
		report: report,
		code:   "unknown_ruleset_reference",
		known:  tagSet(diagnosticEntriesFromRoute(cfg, "rule_set")),
	}
	checks := dnsReferenceCheckers{dns: dnsChecker, ruleSet: ruleSetChecker}
	checks.checkDNSSection(objectValue(cfg["dns"]))
	checks.checkRouteSection(objectValue(cfg["route"]))
	for _, section := range []string{"outbounds", "endpoints"} {
		checks.checkEntryResolvers(cfg[section], section)
	}
	checks.checkNTPResolver(objectValue(cfg["ntp"]))
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

func (c configReferenceChecker) check(path string, value any) {
	reference := strings.TrimSpace(stringValue(value))
	if reference == "" {
		return
	}
	if _, exists := c.known[reference]; !exists {
		addDiagnostic(c.report, c.code, model.ConfigDiagnosticSeverityError, path, reference, "")
	}
}

func (c configReferenceChecker) checkList(path string, value any) {
	if _, ok := value.(string); ok {
		c.check(path, value)
		return
	}
	items, _ := value.([]any)
	for index, item := range items {
		c.check(path+"["+strconv.Itoa(index)+"]", item)
	}
}

func (c configReferenceChecker) checkDomainResolver(path string, value any) {
	switch resolver := value.(type) {
	case string:
		c.check(path, resolver)
	case map[string]any:
		c.check(path+".server", resolver["server"])
	}
}

func (c configReferenceChecker) checkEmptyGroup(path string, object map[string]any) {
	typeName := strings.TrimSpace(stringValue(object["type"]))
	if typeName != "selector" && typeName != "urltest" {
		return
	}
	members, _ := object["outbounds"].([]any)
	if len(members) == 0 {
		addDiagnostic(c.report, "empty_group", model.ConfigDiagnosticSeverityWarning, path+".outbounds", typeName, "")
	}
}

func checkRouteOutboundReferences(
	cfg map[string]any,
	outboundChecker configReferenceChecker,
	ruleSetChecker configReferenceChecker,
) {
	route := objectValue(cfg["route"])
	if route == nil {
		return
	}
	outboundChecker.check("route.final", route["final"])
	routeReferenceCheckers{outbound: outboundChecker, ruleSet: ruleSetChecker}.
		checkRules(route["rules"], "route.rules")
	checkRouteDownloadDetours(route, outboundChecker)
}

func (c routeReferenceCheckers) checkRules(value any, path string) {
	rules, _ := value.([]any)
	for index, item := range rules {
		rule := objectValue(item)
		rulePath := path + "[" + strconv.Itoa(index) + "]"
		c.outbound.check(rulePath+".outbound", rule["outbound"])
		c.ruleSet.checkList(rulePath+".rule_set", rule["rule_set"])
		c.checkRules(rule["rules"], rulePath+".rules")
	}
}

func checkRouteDownloadDetours(route map[string]any, outboundChecker configReferenceChecker) {
	ruleSets, _ := route["rule_set"].([]any)
	seen := make(map[string]struct{}, len(ruleSets))
	for index, item := range ruleSets {
		ruleSet := objectValue(item)
		if ruleSet == nil {
			continue
		}
		path := "route.rule_set[" + strconv.Itoa(index) + "]"
		tag := strings.TrimSpace(stringValue(ruleSet["tag"]))
		if tag == "" {
			addDiagnostic(outboundChecker.report, "missing_tag", model.ConfigDiagnosticSeverityError, path+".tag", "", "")
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		outboundChecker.check(path+".download_detour", ruleSet["download_detour"])
	}
	for _, section := range []string{"geoip", "geosite"} {
		options := objectValue(route[section])
		outboundChecker.check("route."+section+".download_detour", options["download_detour"])
	}
}

func (c configReferenceChecker) checkDNSDetours(dns map[string]any) {
	servers, _ := dns["servers"].([]any)
	for index, item := range servers {
		server := objectValue(item)
		path := "dns.servers[" + strconv.Itoa(index) + "].detour"
		c.check(path, server["detour"])
	}
}

func (c configReferenceChecker) checkNTPDetour(ntp map[string]any) {
	enabled, _ := ntp["enabled"].(bool)
	if enabled {
		c.check("ntp.detour", ntp["detour"])
	}
}

func (c configReferenceChecker) checkExperimentalDetour(experimental map[string]any) {
	clashAPI := objectValue(experimental["clash_api"])
	c.check("experimental.clash_api.external_ui_download_detour", clashAPI["external_ui_download_detour"])
}

func (c dnsReferenceCheckers) checkDNSSection(dns map[string]any) {
	servers, _ := dns["servers"].([]any)
	for index, item := range servers {
		server := objectValue(item)
		path := "dns.servers[" + strconv.Itoa(index) + "]"
		c.dns.checkDomainResolver(path+".domain_resolver", server["domain_resolver"])
		c.dns.check(path+".address_resolver", server["address_resolver"])
	}
	c.dns.check("dns.final", dns["final"])
	c.checkDNSRules(dns["rules"], "dns.rules")
}

func (c dnsReferenceCheckers) checkRouteSection(route map[string]any) {
	c.dns.checkDomainResolver("route.default_domain_resolver", route["default_domain_resolver"])
	c.checkRouteRules(route["rules"], "route.rules")
}

func (c dnsReferenceCheckers) checkDNSRules(value any, path string) {
	rules, _ := value.([]any)
	for index, item := range rules {
		rule := objectValue(item)
		rulePath := path + "[" + strconv.Itoa(index) + "]"
		c.dns.check(rulePath+".server", rule["server"])
		c.ruleSet.checkList(rulePath+".rule_set", rule["rule_set"])
		c.checkDNSRules(rule["rules"], rulePath+".rules")
	}
}

func (c dnsReferenceCheckers) checkRouteRules(value any, path string) {
	rules, _ := value.([]any)
	for index, item := range rules {
		rule := objectValue(item)
		rulePath := path + "[" + strconv.Itoa(index) + "]"
		c.dns.check(rulePath+".server", rule["server"])
		c.checkRouteRules(rule["rules"], rulePath+".rules")
	}
}

func (c dnsReferenceCheckers) checkEntryResolvers(value any, section string) {
	items, _ := value.([]any)
	for index, item := range items {
		entry := objectValue(item)
		path := section + "[" + strconv.Itoa(index) + "].domain_resolver"
		c.dns.checkDomainResolver(path, entry["domain_resolver"])
	}
}

func (c dnsReferenceCheckers) checkNTPResolver(ntp map[string]any) {
	enabled, _ := ntp["enabled"].(bool)
	if enabled {
		c.dns.checkDomainResolver("ntp.domain_resolver", ntp["domain_resolver"])
	}
}
