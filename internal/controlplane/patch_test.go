package controlplane_test

import (
	"context"
	"testing"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/adapters/outbound/nodelink"
	"github.com/xuthus5/boxd/internal/controlplane"
)

func TestAtomicPatchesPreserveUnspecifiedFields(t *testing.T) {
	ctx := context.Background()
	application := newTestApplication(t)

	subscription, err := application.Subscriptions().Save(ctx, control.Subscription{
		Name:            "Primary",
		URL:             "https://example.com/subscription",
		IntervalSeconds: 3600,
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("Subscriptions().Save() error = %v", err)
	}
	renamed := "Renamed"
	subscription, err = application.Subscriptions().Patch(ctx, subscription.ID, control.SubscriptionPatch{Name: &renamed})
	if err != nil {
		t.Fatalf("Subscriptions().Patch() error = %v", err)
	}
	if subscription.Name != renamed || subscription.URL != "https://example.com/subscription" ||
		subscription.IntervalSeconds != 3600 || !subscription.Enabled {
		t.Fatalf("subscription patch changed unspecified fields: %+v", subscription)
	}

	originalDNS, err := application.DNS().Get(ctx)
	if err != nil {
		t.Fatalf("DNS().Get() error = %v", err)
	}
	strategy := "prefer_ipv6"
	patchedDNS, err := application.DNS().Patch(ctx, control.DNSProfilePatch{Strategy: &strategy})
	if err != nil {
		t.Fatalf("DNS().Patch() error = %v", err)
	}
	if patchedDNS.Strategy != strategy || patchedDNS.Cache != originalDNS.Cache ||
		patchedDNS.Domestic != originalDNS.Domestic || patchedDNS.Foreign != originalDNS.Foreign ||
		patchedDNS.Default != originalDNS.Default {
		t.Fatalf("DNS patch changed unspecified fields: before=%+v after=%+v", originalDNS, patchedDNS)
	}

	inbound, err := application.Inbounds().Get(ctx, "mixed")
	if err != nil {
		t.Fatalf("Inbounds().Get() error = %v", err)
	}
	inboundName := "Local Proxy"
	patchedInbound, err := application.Inbounds().Patch(ctx, inbound.ID, control.InboundPatch{Name: &inboundName})
	if err != nil {
		t.Fatalf("Inbounds().Patch() error = %v", err)
	}
	if patchedInbound.Name != inboundName || patchedInbound.Enabled != inbound.Enabled ||
		patchedInbound.Listen != inbound.Listen || patchedInbound.UDP != inbound.UDP || patchedInbound.Kind != inbound.Kind {
		t.Fatalf("inbound patch changed unspecified fields: before=%+v after=%+v", inbound, patchedInbound)
	}

	rule, err := application.Routes().SaveRule(ctx, control.RouteRule{
		Name:   "Example",
		Action: control.RouteActionDirect,
		Match:  control.RouteMatch{DomainSuffixes: []string{".example.com"}},
	})
	if err != nil {
		t.Fatalf("Routes().SaveRule() error = %v", err)
	}
	ruleName := "Example Renamed"
	rule, err = application.Routes().PatchRule(ctx, rule.ID, control.RouteRulePatch{Name: &ruleName})
	if err != nil {
		t.Fatalf("Routes().PatchRule() error = %v", err)
	}
	if rule.Name != ruleName || rule.Action != control.RouteActionDirect || rule.Disabled ||
		len(rule.Match.DomainSuffixes) != 1 || rule.Match.DomainSuffixes[0] != ".example.com" {
		t.Fatalf("route patch changed unspecified fields: %+v", rule)
	}

	if _, err := application.RuleSets().Save(ctx, control.RuleSetDefinition{
		Name: "custom-rules", URL: "https://example.com/rules.srs", Format: control.RuleSetFormatBinary, Enabled: true,
	}); err != nil {
		t.Fatalf("RuleSets().Save() error = %v", err)
	}
	enabled := false
	ruleSet, err := application.RuleSets().Patch(ctx, "custom-rules", control.RuleSetPatch{Enabled: &enabled})
	if err != nil {
		t.Fatalf("RuleSets().Patch() error = %v", err)
	}
	if ruleSet.Enabled || ruleSet.URL != "https://example.com/rules.srs" || ruleSet.Format != control.RuleSetFormatBinary {
		t.Fatalf("rule-set patch changed unspecified fields: %+v", ruleSet)
	}
}

func TestImportLinksIntoExistingGroupAddsMembersWithoutReplacingSelection(t *testing.T) {
	ctx := context.Background()
	application := newApplicationWithDependencies(t, controlplane.Dependencies{NodeCodec: nodelink.New()})

	seed, err := application.Nodes().Import(ctx, []control.Node{{
		ID: "seed", Name: "Seed", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
		Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
	}})
	if err != nil || seed.Added != 1 {
		t.Fatalf("seed Import() = %+v, %v", seed, err)
	}
	if _, err := application.Egress().SaveGroup(ctx, control.Group{
		ID: "manual", Name: "Manual", Kind: control.GroupKindSelector, Members: []string{"seed"}, Selected: "seed",
	}); err != nil {
		t.Fatalf("SaveGroup() error = %v", err)
	}
	result, err := application.Nodes().ImportLinksIntoGroup(ctx, []string{
		"ss://YWVzLTEyOC1nY206c2VjcmV0@127.0.0.2:8388#Imported%20SS",
		"trojan://secret@127.0.0.3:443?sni=example.com#Imported%20Trojan",
	}, "manual")
	if err != nil {
		t.Fatalf("ImportLinksIntoGroup() error = %v", err)
	}
	if result.Added != 2 || len(result.Nodes) != 2 {
		t.Fatalf("ImportLinksIntoGroup() = %+v", result)
	}
	catalog, err := application.Egress().Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if len(catalog.Groups) != 1 || catalog.Groups[0].Selected != "seed" || len(catalog.Groups[0].Members) != 3 {
		t.Fatalf("group after import = %+v", catalog.Groups)
	}
	seen := make(map[string]bool, len(catalog.Groups[0].Members))
	for _, member := range catalog.Groups[0].Members {
		seen[member] = true
	}
	for _, imported := range result.Nodes {
		if !seen[imported.ID] {
			t.Fatalf("imported node %q is not a group member: %+v", imported.ID, catalog.Groups[0].Members)
		}
	}
}

func TestActivityLogReplayAndSubscriptionAreBounded(t *testing.T) {
	service := controlplane.NewActivityService(2)
	for _, message := range []string{"one", "two", "three"} {
		service.Record(context.Background(), control.ActivityLogEntry{Source: "test", Message: message})
	}
	recent := service.Recent(10)
	if len(recent) != 2 || recent[0].Message != "two" || recent[1].Message != "three" {
		t.Fatalf("Recent() = %+v", recent)
	}
	if recent[0].ID != 2 || recent[1].ID != 3 || recent[0].Level != "info" || recent[0].Time.IsZero() {
		t.Fatalf("activity metadata = %+v", recent)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := service.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	service.Record(context.Background(), control.ActivityLogEntry{Source: "test", Message: "live"})
	select {
	case entry := <-stream:
		if entry.Message != "live" {
			t.Fatalf("streamed entry = %+v", entry)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for streamed activity")
	}
	cancel()
	select {
	case _, open := <-stream:
		if open {
			t.Fatal("activity stream remained open after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for activity stream to close")
	}
}
