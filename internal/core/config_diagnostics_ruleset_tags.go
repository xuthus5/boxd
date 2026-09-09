package core

import (
	"slices"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func diagnosticRuleSetEntries(object map[string]any, path string) []diagnosticEntry {
	tags := stringValues(object["tag"])
	if len(tags) == 0 {
		tags = []string{""}
	}
	entries := make([]diagnosticEntry, 0, len(tags))
	_, multipleForm := object["tag"].([]any)
	for index, tag := range tags {
		tagPath := path + ".tag"
		if multipleForm {
			tagPath += "[" + strconv.Itoa(index) + "]"
		}
		entries = append(entries, diagnosticEntry{
			tag: strings.TrimSpace(tag), typeName: stringValue(object["type"]), path: path, tagPath: tagPath,
		})
	}
	return entries
}

func checkRuleSetTagOptions(report *model.ConfigDiagnostics, cfg map[string]any) {
	for _, entry := range diagnosticObjects(objectValue(cfg["route"])["rule_set"], "route.rule_set") {
		tags := stringValues(entry.object["tag"])
		if len(tags) < 2 {
			continue
		}
		switch stringValue(entry.object["type"]) {
		case "", "inline":
			addDiagnostic(report, "invalid_ruleset_tags", model.ConfigDiagnosticSeverityError, entry.path+".tag", "", "inline")
		case "local":
			checkRuleSetTagPlaceholder(report, entry, "path")
		case "remote":
			checkRuleSetTagPlaceholder(report, entry, "url")
			if stringValue(entry.object["initial_path"]) != "" {
				checkRuleSetTagPlaceholder(report, entry, "initial_path")
			}
		}
	}
}

func checkRuleSetTagPlaceholder(report *model.ConfigDiagnostics, entry diagnosticObject, key string) {
	if !strings.Contains(stringValue(entry.object[key]), "{tag}") {
		addDiagnostic(report, "invalid_ruleset_tags", model.ConfigDiagnosticSeverityError, entry.path+"."+key, "", "{tag}")
	}
}

func checkPreferredByReferences(report *model.ConfigDiagnostics, cfg map[string]any) {
	outbounds := append(diagnosticEntries(cfg, "outbounds"), diagnosticEntries(cfg, "endpoints")...)
	routeTypes := diagnosticEntryTypes(outbounds)
	dnsTypes := diagnosticEntryTypes(diagnosticEntriesFromDNS(cfg))
	routeCheck := preferredByChecker{report: report, types: routeTypes, allowed: []string{"bridge", "wireguard", "tailscale", "openvpn-client", "openconnect"}}
	dnsCheck := preferredByChecker{report: report, types: dnsTypes, allowed: []string{"local", "hosts", "mdns", "tailscale", "openvpn", "openconnect"}}
	walkDiagnosticRules(objectValue(cfg["route"])["rules"], "route.rules", func(entry diagnosticObject) {
		routeCheck.check(entry)
	})
	walkDiagnosticRules(objectValue(cfg["dns"])["rules"], "dns.rules", func(entry diagnosticObject) {
		dnsCheck.check(entry)
	})
}

type preferredByChecker struct {
	report  *model.ConfigDiagnostics
	types   map[string]string
	allowed []string
}

func (c preferredByChecker) check(entry diagnosticObject) {
	for _, tag := range stringValues(entry.object["preferred_by"]) {
		if !slices.Contains(c.allowed, c.types[tag]) {
			addDiagnostic(c.report, "invalid_preferred_by_reference", model.ConfigDiagnosticSeverityError, entry.path+".preferred_by", tag, "")
		}
	}
}
