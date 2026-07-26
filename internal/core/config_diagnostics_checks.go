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
			addDiagnostic(report, "duplicate_tag", model.ConfigDiagnosticSeverityError, entry.path+".tag", entry.tag, "")
			continue
		}
		seen[entry.tag] = entry
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
	servers, _ := dns["servers"].([]any)
	for _, item := range servers {
		if strings.EqualFold(stringValue(objectValue(item)["type"]), "fakeip") {
			report.Features.FakeIP = true
			break
		}
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
