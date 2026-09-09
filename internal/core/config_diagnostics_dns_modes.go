package core

import (
	"github.com/xuthus5/boxd/internal/model"
)

type diagnosticRuleSetShape struct {
	ip    bool
	other bool
	query bool
}

func checkDNSRuleModes(report *model.ConfigDiagnostics, cfg map[string]any) {
	rules := objectValue(cfg["dns"])["rules"]
	shapes := diagnosticRuleSetShapes(cfg)
	modern := false
	walkDiagnosticRules(rules, "dns.rules", func(entry diagnosticObject) {
		modern = modern || dnsRuleNeedsModernMode(entry.object, shapes)
	})
	walkDiagnosticRules(rules, "dns.rules", func(entry diagnosticObject) {
		checkDNSRuleModeFields(report, entry, modern)
		if modern && !dnsRuleMatchesResponse(entry.object) {
			checkDNSResponseRuleSets(report, entry, shapes)
		}
	})
}

func dnsRuleNeedsModernMode(rule map[string]any, shapes map[string]diagnosticRuleSetShape) bool {
	switch stringValue(rule["action"]) {
	case "evaluate", "respond":
		return true
	}
	for _, field := range []string{"match_response", "response_rcode", "response_answer", "response_ns", "response_extra", "ip_version", "query_type", "race", "speculative", "disable_optimistic_cache"} {
		if diagnosticDNSFieldActive(rule, field) {
			return true
		}
	}
	for _, tag := range stringValues(rule["rule_set"]) {
		if shapes[tag].query {
			return true
		}
	}
	return false
}

func checkDNSRuleModeFields(report *model.ConfigDiagnostics, entry diagnosticObject, modern bool) {
	for _, key := range []string{"strategy", "rule_set_ip_cidr_accept_empty"} {
		if !diagnosticDNSFieldActive(entry.object, key) {
			continue
		}
		if modern {
			addDiagnostic(report, "dns_legacy_mode_conflict", model.ConfigDiagnosticSeverityError, entry.path+"."+key, "", key)
		} else {
			checkDeprecatedField(report, entry, key)
		}
	}
	if dnsRuleMatchesResponse(entry.object) {
		return
	}
	for _, key := range []string{"ip_cidr", "ip_is_private", "ip_accept_any", "response_rcode", "response_answer", "response_ns", "response_extra"} {
		if !diagnosticDNSFieldActive(entry.object, key) {
			continue
		}
		if modern {
			addDiagnostic(report, "dns_response_requires_match", model.ConfigDiagnosticSeverityError, entry.path+"."+key, "", "")
		} else {
			checkDeprecatedField(report, entry, key)
		}
	}
}

func checkDNSResponseRuleSets(report *model.ConfigDiagnostics, entry diagnosticObject, shapes map[string]diagnosticRuleSetShape) {
	for _, tag := range stringValues(entry.object["rule_set"]) {
		shape := shapes[tag]
		if shape.ip && !shape.other {
			addDiagnostic(report, "dns_response_requires_match", model.ConfigDiagnosticSeverityError, entry.path+".rule_set", tag, "")
		}
	}
}

func diagnosticRuleSetShapes(cfg map[string]any) map[string]diagnosticRuleSetShape {
	shapes := make(map[string]diagnosticRuleSetShape)
	for _, entry := range diagnosticObjects(objectValue(cfg["route"])["rule_set"], "route.rule_set") {
		if kind := stringValue(entry.object["type"]); kind == "local" || kind == "remote" {
			addDiagnosticFileRuleSetShapes(shapes, entry)
			continue
		}
		shape := diagnosticRuleSetShape{}
		walkDiagnosticRules(entry.object["rules"], entry.path+".rules", func(rule diagnosticObject) {
			shape.ip = shape.ip || diagnosticFieldActive(rule.object["ip_cidr"])
			shape.query = shape.query || diagnosticFieldActive(rule.object["query_type"])
			shape.other = shape.other || diagnosticRuleSetHasNonIP(rule.object)
		})
		for _, tag := range stringValues(entry.object["tag"]) {
			shapes[tag] = shape
		}
	}
	return shapes
}

func diagnosticRuleSetHasNonIP(rule map[string]any) bool {
	for key, value := range rule {
		switch key {
		case "type", "mode", "rules", "ip_cidr", "invert":
			continue
		}
		if diagnosticFieldActive(value) {
			return true
		}
	}
	return false
}

func diagnosticFieldActive(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	case float64:
		return typed != 0
	default:
		return true
	}
}

func diagnosticDNSFieldActive(rule map[string]any, key string) bool {
	if key == "response_rcode" {
		return rule[key] != nil
	}
	if key == "strategy" && stringValue(rule[key]) == "as_is" {
		return false
	}
	return diagnosticFieldActive(rule[key])
}
