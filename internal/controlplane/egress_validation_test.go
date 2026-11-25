package controlplane_test

import (
	"context"
	"testing"

	"github.com/xuthus5/boxd/control"
)

func TestEgressValidationUsesDesiredStateRules(t *testing.T) {
	ctx := context.Background()
	application := newTestApplication(t)
	if _, err := application.Egress().SaveNode(ctx, control.Node{
		ID: "node-1", Name: "Node", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
		Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
	}); err != nil {
		t.Fatalf("SaveNode() error = %v", err)
	}

	diagnostics, err := application.Egress().ValidateGroup(ctx, control.Group{
		Name: "Invalid", Kind: control.GroupKindSelector, Members: []string{"missing"},
	})
	if err != nil {
		t.Fatalf("ValidateGroup() error = %v", err)
	}
	if len(diagnostics) == 0 || diagnostics[0].Code != "missing_node" {
		t.Fatalf("group diagnostics = %+v", diagnostics)
	}

	diagnostics, err = application.Egress().ValidateGroup(ctx, control.Group{
		Name: "Valid", Kind: control.GroupKindSelector, Members: []string{"node-1"},
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("valid group diagnostics = %+v, %v", diagnostics, err)
	}

	if _, err := application.Egress().SaveGroup(ctx, control.Group{
		ID: "group-1", Name: "Group", Kind: control.GroupKindSelector, Members: []string{"node-1"},
	}); err != nil {
		t.Fatalf("SaveGroup() error = %v", err)
	}
	diagnostics, err = application.Egress().ValidateChain(ctx, control.ProxyChain{
		Name: "Loop", Hops: []control.ProxyHop{
			{Kind: control.EgressKindNode, ID: "node-1"},
			{Kind: control.EgressKindNode, ID: "node-1"},
		},
	})
	if err != nil {
		t.Fatalf("ValidateChain() error = %v", err)
	}
	if len(diagnostics) == 0 || diagnostics[0].Code != "duplicate_hop" {
		t.Fatalf("chain diagnostics = %+v", diagnostics)
	}
}
