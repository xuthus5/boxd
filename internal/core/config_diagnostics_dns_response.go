package core

import (
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type dnsResponseInspector struct {
	report          *model.ConfigDiagnostics
	servers         map[string]string
	definitions     map[string]string
	references      map[string]bool
	definitionOrder []string
	anonymous       string
	anonymousRead   bool
	raceSeen        bool
}

func checkDNSResponseRules(report *model.ConfigDiagnostics, cfg map[string]any) {
	dns := objectValue(cfg["dns"])
	checkDNSRuleModes(report, cfg)
	inspector := dnsResponseInspector{
		report: report, servers: diagnosticEntryTypes(diagnosticEntriesFromDNS(cfg)),
		definitions: make(map[string]string), references: make(map[string]bool),
	}
	for _, entry := range diagnosticObjects(dns["rules"], "dns.rules") {
		inspector.checkRule(entry)
	}
	for _, tag := range inspector.definitionOrder {
		if !inspector.references[tag] {
			addDiagnostic(report, "unused_dns_evaluate_tag", model.ConfigDiagnosticSeverityWarning, inspector.definitions[tag]+".tag", tag, "")
		}
	}
}

func (i *dnsResponseInspector) checkRule(entry diagnosticObject) {
	action := stringValue(entry.object["action"])
	i.checkAction(entry)
	if boolValue(entry.object["speculative"]) && !i.raceSeen {
		addDiagnostic(i.report, "dns_speculative_without_race", model.ConfigDiagnosticSeverityWarning, entry.path+".speculative", "", "")
	}
	i.raceSeen = i.raceSeen || boolValue(entry.object["race"])
	i.checkResponseUse(entry)
	if action != "evaluate" {
		return
	}
	tag := stringValue(entry.object["tag"])
	if tag == "" {
		if i.anonymous != "" && !i.anonymousRead {
			addDiagnostic(i.report, "dns_evaluate_overwritten", model.ConfigDiagnosticSeverityWarning, i.anonymous+".action", "", "")
		}
		i.anonymous, i.anonymousRead = entry.path, false
		return
	}
	if _, exists := i.definitions[tag]; exists {
		addDiagnostic(i.report, "duplicate_dns_evaluate_tag", model.ConfigDiagnosticSeverityError, entry.path+".tag", tag, "")
		return
	}
	i.definitions[tag] = entry.path
	i.definitionOrder = append(i.definitionOrder, tag)
}

func (i *dnsResponseInspector) checkAction(entry diagnosticObject) {
	object := entry.object
	action := stringValue(object["action"])
	server := stringValue(object["server"])
	if (action == "" || action == "route" || action == "evaluate") && server == "" {
		addDiagnostic(i.report, "dns_missing_server", model.ConfigDiagnosticSeverityError, entry.path+".server", "", "")
	}
	if action == "evaluate" && i.servers[server] == "fakeip" {
		addDiagnostic(i.report, "dns_evaluate_fakeip", model.ConfigDiagnosticSeverityError, entry.path+".server", server, "")
	}
	if boolValue(object["remove_client_subnet"]) && object["client_subnet"] != nil {
		addDiagnostic(i.report, "dns_client_subnet_conflict", model.ConfigDiagnosticSeverityError, entry.path+".remove_client_subnet", "", "")
	}
	i.checkRaceAction(entry)
}

func (i *dnsResponseInspector) checkRaceAction(entry diagnosticObject) {
	object := entry.object
	action := stringValue(object["action"])
	if !boolValue(object["race"]) {
		return
	}
	if !dnsRuleMatchesResponse(object) {
		addDiagnostic(i.report, "dns_race_requires_response", model.ConfigDiagnosticSeverityError, entry.path+".race", "", "")
	}
	switch action {
	case "", "route", "respond", "reject", "predefined":
	default:
		addDiagnostic(i.report, "dns_race_invalid_action", model.ConfigDiagnosticSeverityError, entry.path+".race", "", "")
	}
	if boolValue(object["speculative"]) {
		addDiagnostic(i.report, "dns_race_speculative_conflict", model.ConfigDiagnosticSeverityError, entry.path+".speculative", "", "")
	}
}

func (i *dnsResponseInspector) checkResponseUse(entry diagnosticObject) {
	if stringValue(entry.object["type"]) != "logical" {
		i.checkDefaultResponseUse(entry)
		return
	}
	for _, child := range diagnosticObjects(entry.object["rules"], entry.path+".rules") {
		i.checkResponseUse(child)
	}
	if stringValue(entry.object["action"]) == "respond" {
		if dnsRuleHasTaggedResponse(entry.object) {
			addDiagnostic(i.report, "dns_logical_respond_tag", model.ConfigDiagnosticSeverityError, entry.path+".action", "", "")
		}
		i.requireAnonymous(entry.path + ".action")
	}
}

func (i *dnsResponseInspector) checkDefaultResponseUse(entry diagnosticObject) {
	response := entry.object["match_response"]
	if tag := stringValue(response); tag != "" {
		if _, exists := i.definitions[tag]; !exists {
			addDiagnostic(i.report, "unknown_dns_evaluate_reference", model.ConfigDiagnosticSeverityError, entry.path+".match_response", tag, "")
		} else {
			i.references[tag] = true
		}
		return
	}
	if boolValue(response) {
		i.requireAnonymous(entry.path + ".match_response")
	} else if stringValue(entry.object["action"]) == "respond" {
		i.requireAnonymous(entry.path + ".action")
	}
}

func (i *dnsResponseInspector) requireAnonymous(path string) {
	if i.anonymous == "" {
		addDiagnostic(i.report, "dns_evaluate_required", model.ConfigDiagnosticSeverityError, path, "", "")
		return
	}
	i.anonymousRead = true
}

func dnsRuleMatchesResponse(rule map[string]any) bool {
	if boolValue(rule["match_response"]) || stringValue(rule["match_response"]) != "" {
		return true
	}
	for _, entry := range diagnosticObjects(rule["rules"], "") {
		if dnsRuleMatchesResponse(entry.object) {
			return true
		}
	}
	return false
}

func dnsRuleHasTaggedResponse(rule map[string]any) bool {
	if stringValue(rule["match_response"]) != "" {
		return true
	}
	for _, entry := range diagnosticObjects(rule["rules"], "") {
		if dnsRuleHasTaggedResponse(entry.object) {
			return true
		}
	}
	return false
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func diagnosticEntryTypes(entries []diagnosticEntry) map[string]string {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		result[entry.tag] = strings.ToLower(entry.typeName)
	}
	return result
}
