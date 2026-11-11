package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

func TestBuildSupportBundleRedactsSensitiveSources(t *testing.T) {
	collection := supportBundleCollection{
		Sources: supportBundleSources{
			Version:   supportBundleVersion{Version: "dev", KernelVersion: "1.14.0"},
			Service:   &model.ServiceStatus{Running: true, Uptime: "1h", Memory: 8192, Version: "1.14.0", LastError: "dial secret.example", LastErrorCode: "runtime_error"},
			Readiness: supportText("ready"),
			Memory:    &supportBundleMemory{Alloc: 1, Total: 2, Sys: 3, NumGC: 4, HeapInuse: 5, StackInuse: 6, NumGoroutine: 7},
			Diagnostics: &model.ConfigDiagnostics{
				Status: "warning", CheckedAt: time.Now().UTC(), Summary: model.ConfigDiagnosticsSummary{Errors: 1},
				Counts: model.ConfigDiagnosticsCounts{Inbounds: 1}, Features: model.ConfigDiagnosticsFeatures{TUN: true},
				Issues: []model.ConfigDiagnostic{{Code: "dns_rule_invalid", Severity: "error", Detail: "secret detail", Path: "dns.rules"}},
			},
			RuleSets: []model.RuleSetStatusItem{
				{Tag: "private-rules", Type: "remote", Format: "binary", Builtin: true, Updatable: true, LastUpdated: &time.Time{}, URL: "https://rules.example/private"},
			},
			RuleSetAuto: &model.RuleSetAutoUpdate{Enabled: true, Interval: "24h"},
			Subscriptions: []model.Subscription{{
				Name: "Secret subscription", URL: "https://secret.example/sub", Error: "refresh failed",
				ErrorCode: "fetch_failed", Traffic: &model.SubscriptionTraffic{Total: 10},
			}},
			Nodes: []supportBundleNode{{Type: "vless", Source: "import"}, {Type: "vless", Source: "subscription"}},
			NodeHistory: map[string]map[string][]model.LatencyPoint{
				"secret-node": {"tcp": {{Success: true, LatencyMs: 20}, {Success: false, LatencyMs: 40}}},
			},
			ApplyHistory: []model.ConfigApplyEvent{
				{Status: model.ConfigApplyStatusApplied, ErrorCode: "", AppliedAt: time.Now().Add(-time.Minute)},
				{Status: model.ConfigApplyStatusRolledBack, ErrorCode: "restart_failed", AppliedAt: time.Now()},
			},
			Network: []service.InterfaceInfo{{Name: "eth0", IPs: []string{"192.0.2.2", "2001:db8::2"}}},
			Preferences: model.UIPreferences{
				Theme: themeDark, Language: languageChinese, MinimumLogLevel: minimumLogLevel,
			},
		},
		Requests: supportBundleOKRequests(),
	}

	bundle := buildSupportBundle(collection, time.Date(2026, 11, 12, 1, 2, 3, 0, time.UTC))
	if bundle.Product != "boxd" || bundle.FormatVersion != 1 || bundle.Redaction.Strategy != "allowlist" {
		t.Fatalf("bundle header = %+v", bundle)
	}
	if !bundle.Service.Available || bundle.Service.Running == nil || !*bundle.Service.Running {
		t.Fatalf("service summary = %+v", bundle.Service)
	}
	if !bundle.Config.Diagnostics.Available || bundle.Config.Diagnostics.IssueCodes["dns_rule_invalid"] != 1 {
		t.Fatalf("diagnostics summary = %+v", bundle.Config.Diagnostics)
	}
	if bundle.Config.RuleSets.Total != 1 || bundle.Config.RuleSets.Builtin != 1 || bundle.Config.RuleSets.Updatable != 1 {
		t.Fatalf("rule-set summary = %+v", bundle.Config.RuleSets)
	}
	if bundle.Resources.Nodes.Total != 2 || bundle.Resources.Nodes.Types["vless"] != 2 ||
		bundle.Resources.Nodes.Sources["import"] != 1 || bundle.Resources.Nodes.ProbeSamples != 2 {
		t.Fatalf("node summary = %+v", bundle.Resources.Nodes)
	}
	if bundle.Resources.Nodes.AverageLatency == nil || *bundle.Resources.Nodes.AverageLatency != 30 {
		t.Fatalf("average latency = %v", bundle.Resources.Nodes.AverageLatency)
	}
	if bundle.Resources.Subscriptions.Failed != 1 || bundle.Resources.Subscriptions.WithTraffic != 1 {
		t.Fatalf("subscription summary = %+v", bundle.Resources.Subscriptions)
	}
	if bundle.Config.ApplyHistory.FailureCount != 1 || bundle.Network.IPv4 != 1 || bundle.Network.IPv6 != 1 {
		t.Fatalf("history/network summary = %+v / %+v", bundle.Config.ApplyHistory, bundle.Network)
	}

	text, err := formatSupportBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret.example", "Secret subscription", "private-rules", "rules.example", "secret detail", "secret-node", "dns.rules"} {
		if strings.Contains(text, secret) {
			t.Fatalf("support bundle leaked %q: %s", secret, text)
		}
	}
}

func TestCollectSupportBundleRecordsUnavailableSources(t *testing.T) {
	loaders := supportBundleLoaders{
		Version: func() (supportBundleVersion, error) { return supportBundleVersion{Version: "dev"}, nil },
		Service: func() (model.ServiceStatus, error) {
			return model.ServiceStatus{}, service.Errorf(503, "not_ready", "service is not ready")
		},
		Subscriptions: func() ([]model.Subscription, error) { return []model.Subscription{}, nil },
	}
	collection := collectSupportBundle(loaders, model.UIPreferences{})
	if collection.Requests["version"].Status != "ok" {
		t.Fatalf("version state = %+v", collection.Requests["version"])
	}
	if state := collection.Requests["service"]; state.Status != "unavailable" || state.ErrorCode != "not_ready" {
		t.Fatalf("service state = %+v", state)
	}
	if state := collection.Requests["nodes"]; state.Status != "unavailable" || state.ErrorCode != "unavailable" {
		t.Fatalf("missing loader state = %+v", state)
	}
	if collection.Requests["subscriptions"].Status != "ok" || collection.Sources.Subscriptions == nil {
		t.Fatalf("successful empty source = %+v", collection.Sources.Subscriptions)
	}
	if got := countUnavailableSupportBundleSources(collection.Requests); got != 10 {
		t.Fatalf("unavailable count = %d", got)
	}
}

func TestSupportBundleOutputAndFilename(t *testing.T) {
	collection := supportBundleCollection{Sources: supportBundleSources{}, Requests: supportBundleOKRequests()}
	text, err := formatSupportBundle(buildSupportBundle(collection, time.Date(2026, 11, 12, 3, 4, 5, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		t.Fatalf("bundle is not JSON: %v", err)
	}
	if got := supportBundleFilename(time.Date(2026, 11, 12, 3, 4, 5, 0, time.UTC)); got != "boxd-support-bundle-2026-11-12T03-04-05-000Z.json" {
		t.Fatalf("filename = %q", got)
	}
	if supportBundleToken(strings.Repeat("a", 80)) != strings.Repeat("a", 64) {
		t.Fatal("token was not truncated")
	}
	if supportBundleErrorCode(errors.New("secret")) != "unavailable" {
		t.Fatal("unexpected non-domain error code")
	}
}

func TestSettingsShowsSupportBundleExport(t *testing.T) {
	app := newTestApplication(seededModel(themeLight))
	tt := ui.NewTester(app.view, 1440, 900)
	app.router.Push("/settings")
	settleTransitions(tt)
	for _, text := range []string{"Backup and support", "Export support bundle", "allowlisted JSON summary"} {
		if !tt.HasText(text) {
			t.Errorf("settings is missing %q: %v", text, tt.Texts())
		}
	}
}

func supportBundleOKRequests() map[string]supportBundleRequestState {
	requests := make(map[string]supportBundleRequestState, len(supportBundleSourceNames))
	for _, name := range supportBundleSourceNames {
		requests[name] = supportBundleRequestState{Status: "ok"}
	}
	return requests
}
