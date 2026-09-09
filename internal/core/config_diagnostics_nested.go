package core

import (
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/internal/model"
)

func checkNestedRuleActions(report *model.ConfigDiagnostics, cfg map[string]any) {
	checks := map[string][]reflect.Type{
		"route": {reflect.TypeFor[option.RuleAction](), reflect.TypeFor[option.RouteActionOptions]()},
		"dns":   {reflect.TypeFor[option.DNSRuleAction](), reflect.TypeFor[option.DNSRouteActionOptions](), reflect.TypeFor[option.DNSEvaluateActionOptions]()},
	}
	for _, section := range []string{"route", "dns"} {
		keys := make(map[string]struct{})
		for _, kind := range checks[section] {
			collectDiagnosticJSONKeys(keys, kind)
		}
		for _, entry := range diagnosticObjects(objectValue(cfg[section])["rules"], section+".rules") {
			walkDiagnosticRules(entry.object["rules"], entry.path+".rules", func(child diagnosticObject) {
				checkNestedActionKeys(report, child, keys)
			})
		}
	}
}

func checkNestedActionKeys(report *model.ConfigDiagnostics, entry diagnosticObject, keys map[string]struct{}) {
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		if _, exists := entry.object[key]; exists {
			addDiagnostic(report, "nested_rule_action_unsupported", model.ConfigDiagnosticSeverityError, entry.path+"."+key, "", "")
		}
	}
}

func collectDiagnosticJSONKeys(keys map[string]struct{}, kind reflect.Type) {
	for kind.Kind() == reflect.Pointer {
		kind = kind.Elem()
	}
	if kind.Kind() != reflect.Struct {
		return
	}
	for index := range kind.NumField() {
		field := kind.Field(index)
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if field.Anonymous && name == "" {
			collectDiagnosticJSONKeys(keys, field.Type)
		} else if name != "" {
			keys[name] = struct{}{}
		}
	}
}

func containsEmptyRuleSetTag(tags []string) bool {
	return slices.ContainsFunc(tags, func(tag string) bool { return strings.TrimSpace(tag) == "" })
}
