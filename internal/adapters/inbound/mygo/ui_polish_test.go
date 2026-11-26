package mygo

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
)

func TestPageHeadingsRenderOnceForDistinctTitles(t *testing.T) {
	for _, item := range navigation {
		if item.Title == item.Label {
			continue
		}
		controller := newViewTestController(ControllerOptions{})
		controller.page = item.ID
		tester := ui.NewTester(controller.View, 1320, 860)
		if count := countExactText(tester.Texts(), item.Title); count != 1 {
			t.Fatalf("page %s rendered title %q %d times, want 1; texts=%q", item.ID, item.Title, count, tester.Texts())
		}
	}
}

func TestSettingsDraftUsesProductDefaults(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.syncDrafts()
	if got := controller.settingsDraft[control.SettingSubscriptionMaxAttempts]; got != "3" {
		t.Fatalf("max attempts = %q, want 3", got)
	}
	if got := controller.settingsDraft[control.SettingSubscriptionConflictPolicy]; got != control.SubscriptionConflictKeepLocal {
		t.Fatalf("conflict policy = %q", got)
	}
	if got := controller.settingsDraft[control.SettingAutoRefresh]; got != "true" {
		t.Fatalf("auto refresh = %q", got)
	}
}

func TestDNSStrategyUsesFriendlyLabel(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageDNS
	tester := ui.NewTester(controller.View, 1320, 860)
	if !tester.HasText("偏好 IPv4") {
		t.Fatalf("friendly DNS strategy missing; texts=%q", tester.Texts())
	}
	if tester.HasText("prefer_ipv4") {
		t.Fatalf("raw DNS strategy leaked into the UI; texts=%q", tester.Texts())
	}
}

func TestDNSStrategyChoicesMatchControlPlane(t *testing.T) {
	labels, values := dnsStrategyChoices()
	if strings.Join(values, ",") != "auto,prefer_ipv4,prefer_ipv6" {
		t.Fatalf("values = %q", values)
	}
	for i, value := range values {
		if got := dnsStrategyLabel(value); got != labels[i] {
			t.Fatalf("label for %q = %q, want %q", value, got, labels[i])
		}
	}
}

func TestCompactVersionFitsNarrowSidebar(t *testing.T) {
	if got := compactVersion("0.1.0-dev"); got != "v0.1.0" {
		t.Fatalf("compact version = %q", got)
	}
	if got := compactVersion("v1.2.3+build.7"); got != "v1.2.3" {
		t.Fatalf("compact version = %q", got)
	}
}

func TestAboutPageDoesNotRepeatKernelStatus(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageAbout
	tester := ui.NewTester(controller.View, 960, 640)
	if count := countExactText(tester.Texts(), "内核未启动"); count != 1 {
		t.Fatalf("kernel status rendered %d times, want 1; texts=%q", count, tester.Texts())
	}
}

func TestRouteEgressOptionsUseHumanLabels(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	options := strings.Join(egressOptions(controller.snapshot), "\n")
	for _, expected := range []string{"直连", "策略组 · Auto Select", "链路 · Two hop", "节点 · Tokyo Edge"} {
		if !strings.Contains(options, expected) {
			t.Fatalf("options = %q, missing %q", options, expected)
		}
	}
	for _, leaked := range []string{"direct:", "group:", "chain:", "node:", "node-1"} {
		if strings.Contains(options, leaked) {
			t.Fatalf("options leaked internal value %q: %q", leaked, options)
		}
	}
	ref := egressRefFromLabel("策略组 · Auto Select", controller.snapshot)
	if ref.Kind != control.EgressKindGroup || ref.ID != "group-1" {
		t.Fatalf("resolved ref = %+v", ref)
	}
}

func TestRouteDefaultEgressDoesNotExposeInternalValue(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageRoutes
	tester := ui.NewTester(controller.View, 1320, 860)
	if !tester.HasText("直连") {
		t.Fatalf("default egress label missing; texts=%q", tester.Texts())
	}
	if tester.HasText("direct:直连") {
		t.Fatalf("internal egress value rendered; texts=%q", tester.Texts())
	}
}

func TestHeaderPrimaryActionStillOpensEditor(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageSubscriptions
	tester := ui.NewTester(controller.View, 1320, 860)
	if err := tester.Click("添加订阅"); err != nil {
		t.Fatalf("click header action: %v", err)
	}
	if !controller.editorOpen || controller.editor.kind != editorSubscription {
		t.Fatalf("editor state = %+v, open=%v", controller.editor, controller.editorOpen)
	}
}

func countExactText(texts []string, want string) int {
	count := 0
	for _, text := range texts {
		if text == want {
			count++
		}
	}
	return count
}
