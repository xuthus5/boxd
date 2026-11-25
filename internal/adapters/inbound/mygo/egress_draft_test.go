package mygo

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
)

func TestGroupDraftKeepsOrderedUniqueMembers(t *testing.T) {
	draft := groupDraft{Kind: control.GroupKindSelector}
	draft.addMember("a")
	draft.addMember("b")
	draft.addMember("a")
	if !reflect.DeepEqual(draft.Members, []string{"a", "b"}) {
		t.Fatalf("members = %v", draft.Members)
	}
	if draft.Selected != "a" {
		t.Fatalf("selected = %q, want first member", draft.Selected)
	}
	draft.reorderMembers([]int{0}, 2)
	if !reflect.DeepEqual(draft.Members, []string{"b", "a"}) {
		t.Fatalf("reordered members = %v", draft.Members)
	}
	draft.Selected = "b"
	draft.removeMember("b")
	if draft.Selected != "a" {
		t.Fatalf("selected after removing current = %q, want fallback", draft.Selected)
	}
	draft.addMember("c")
	draft.moveMember(1, -1)
	if !reflect.DeepEqual(draft.Members, []string{"c", "a"}) {
		t.Fatalf("moved members = %v", draft.Members)
	}
}

func TestChainDraftLimitsDuplicatesAndPreservesOrder(t *testing.T) {
	draft := chainDraft{Name: "Relay"}
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		draft.addHop(control.EgressKindNode, id)
	}
	if len(draft.Hops) != maxChainHops {
		t.Fatalf("hops = %d, want %d", len(draft.Hops), maxChainHops)
	}
	if draft.addHop(control.EgressKindNode, "a") {
		t.Fatal("duplicate hop was added")
	}
	draft.reorderHops([]int{0}, 3)
	if got := []string{draft.Hops[0].ID, draft.Hops[1].ID, draft.Hops[2].ID}; !reflect.DeepEqual(got, []string{"b", "c", "a"}) {
		t.Fatalf("reordered hops = %v", got)
	}
	draft.removeHop(1)
	if len(draft.Hops) != 3 || draft.Hops[1].ID != "a" {
		t.Fatalf("hops after remove = %+v", draft.Hops)
	}
}

func TestChainDraftValidationUsesResolvedResources(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	draft := chainDraft{Name: "Relay", Hops: []control.ProxyHop{
		{Kind: control.EgressKindNode, ID: "node-1"},
		{Kind: control.EgressKindGroup, ID: "group-1"},
		{Kind: control.EgressKindNode, ID: "node-1"},
	}}
	foundDuplicate := false
	for _, issue := range draft.issues(controller.snapshot) {
		if strings.Contains(issue, "重复") {
			foundDuplicate = true
		}
	}
	if !foundDuplicate {
		t.Fatalf("issues = %v, want duplicate diagnostic", draft.issues(controller.snapshot))
	}
	draft.Hops = []control.ProxyHop{{Kind: control.EgressKindNode, ID: "missing"}}
	if issues := draft.issues(controller.snapshot); len(issues) != 1 {
		t.Fatalf("missing node issues = %v", issues)
	}
}

func TestEgressEditorsUseResolvedResourcePickers(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.snapshot.Catalog.Nodes = append(controller.snapshot.Catalog.Nodes, control.Node{
		ID: "node-2", Name: "Seoul Edge", Protocol: "trojan", Server: "seoul.example", Port: 8443,
	})
	controller.snapshot.NodesByID["node-2"] = controller.snapshot.Catalog.Nodes[1]
	controller.page = pageEgress
	tester := ui.NewTester(controller.View, 1320, 900)
	if err := tester.Click("编辑策略组 Auto Select"); err != nil {
		t.Fatalf("open group editor: %v", err)
	}
	tester.Frame()
	if !tester.HasText("可用节点") || !tester.HasText("已选成员") || !tester.HasText("Tokyo Edge") {
		t.Fatalf("group editor missing resolved picker; texts=%q", tester.Texts())
	}
	if tester.HasText("成员节点 ID") {
		t.Fatalf("group editor still exposes raw IDs; texts=%q", tester.Texts())
	}
	if err := tester.Click("添加节点 Seoul Edge"); err != nil {
		t.Fatalf("add node: %v", err)
	}
	tester.Frame()
	if len(controller.editor.group.Members) != 2 {
		t.Fatalf("members = %v", controller.editor.group.Members)
	}
	if err := tester.Click("下移节点 Tokyo Edge"); err != nil {
		t.Fatalf("move member down: %v", err)
	}
	tester.Frame()
	if got := controller.editor.group.Members; !reflect.DeepEqual(got, []string{"node-2", "node-1"}) {
		t.Fatalf("members after move = %v", got)
	}
}

func TestChainEditorRendersVisualPathAndResourceGroups(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageEgress
	tester := ui.NewTester(controller.View, 1320, 900)
	if err := tester.Click("编辑链路 Two hop"); err != nil {
		t.Fatalf("open chain editor: %v", err)
	}
	tester.Frame()
	for _, text := range []string{"添加跳点", "链路路径", "本地", "目标网络", "策略组"} {
		if !tester.HasText(text) {
			t.Fatalf("chain editor missing %q; texts=%q", text, tester.Texts())
		}
	}
	if tester.HasText("chain:<链路ID>") {
		t.Fatalf("chain editor still advertises unsupported hops; texts=%q", tester.Texts())
	}
	if err := tester.Click("添加跳点 Auto Select"); err != nil {
		t.Fatalf("add group hop: %v", err)
	}
	tester.Frame()
	if len(controller.editor.chain.Hops) != 2 || controller.editor.chain.Hops[1].Kind != control.EgressKindGroup {
		t.Fatalf("hops = %+v", controller.editor.chain.Hops)
	}
}
