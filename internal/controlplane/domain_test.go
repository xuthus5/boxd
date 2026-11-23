package controlplane_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/controlplane"
	"github.com/xuthus5/boxd/internal/kernel/fake"
	"github.com/xuthus5/boxd/internal/kernel/manager"
	"github.com/xuthus5/boxd/internal/kernel/memstore"
	"github.com/xuthus5/boxd/kernel"
)

func TestDefaultProfileAndManagedMutations(t *testing.T) {
	ctx := context.Background()
	application := newTestApplication(t)

	inbounds, err := application.Inbounds().List(ctx)
	if err != nil {
		t.Fatalf("Inbounds().List() error = %v", err)
	}
	if len(inbounds) != 2 {
		t.Fatalf("inbounds = %+v, want mixed and tun", inbounds)
	}
	mixed := inbounds[0]
	if mixed.Kind != control.InboundKindMixed || !mixed.Enabled || mixed.Listen.Host != "::" || mixed.Listen.Port != 1080 || !mixed.UDP || mixed.Auth != nil {
		t.Fatalf("default mixed inbound = %+v", mixed)
	}

	if _, err := application.Egress().SaveNode(ctx, control.Node{
		ID:       "hk-01",
		Name:     "Hong Kong",
		Protocol: "shadowsocks",
		Server:   "127.0.0.1",
		Port:     8388,
		Options:  map[string]any{"method": "aes-128-gcm", "password": "secret"},
	}); err != nil {
		t.Fatalf("SaveNode() error = %v", err)
	}
	if _, err := application.Egress().SaveGroup(ctx, control.Group{
		ID:      "smart",
		Name:    "Smart",
		Kind:    control.GroupKindSelector,
		Members: []string{"hk-01"},
	}); err != nil {
		t.Fatalf("SaveGroup() error = %v", err)
	}
	if _, err := application.Egress().SaveChain(ctx, control.ProxyChain{
		ID:   "relay",
		Name: "Relay",
		Hops: []control.ProxyHop{
			{Kind: control.EgressKindNode, ID: "hk-01"},
			{Kind: control.EgressKindGroup, ID: "smart"},
		},
	}); err != nil {
		t.Fatalf("SaveChain() error = %v", err)
	}
	if _, err := application.Routes().SetDefaultEgress(ctx, control.EgressRef{Kind: control.EgressKindChain, ID: "relay"}); err != nil {
		t.Fatalf("SetDefaultEgress() error = %v", err)
	}
	if _, err := application.Routes().SaveRule(ctx, control.RouteRule{
		ID:     "media",
		Name:   "Media",
		Action: control.RouteActionEgress,
		Egress: control.EgressRef{Kind: control.EgressKindGroup, ID: "smart"},
		Match:  control.RouteMatch{DomainSuffixes: []string{".netflix.com"}},
	}); err != nil {
		t.Fatalf("SaveRule() error = %v", err)
	}
	catalog, err := application.Egress().Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if len(catalog.Nodes) != 1 || len(catalog.Groups) != 1 || len(catalog.Chains) != 1 {
		t.Fatalf("catalog = %+v", catalog)
	}

	dnsProfile, err := application.DNS().Get(ctx)
	if err != nil {
		t.Fatalf("DNS().Get() error = %v", err)
	}
	dnsProfile.Strategy = "prefer_ipv6"
	if _, err := application.DNS().Update(ctx, dnsProfile); err != nil {
		t.Fatalf("DNS().Update() error = %v", err)
	}

	if _, err := application.Config().Reconcile(ctx, kernel.ApplyOptions{Source: "test"}); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if traffic, err := application.Observability().Traffic(ctx); err != nil || traffic.UploadTotal != 0 {
		t.Fatalf("Traffic() = %+v, %v", traffic, err)
	}
	capabilities, err := application.Setup().Capabilities(ctx)
	if err != nil || capabilities.Kind != kernel.EngineSingBox {
		t.Fatalf("Capabilities() = %+v, %v", capabilities, err)
	}
}

func TestProfileValidationRejectsInvalidTopology(t *testing.T) {
	ctx := context.Background()
	application := newTestApplication(t)

	if _, err := application.Egress().SaveGroup(ctx, control.Group{ID: "empty", Name: "Empty", Kind: control.GroupKindSelector}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("empty group error = %v, want invalid config", err)
	}
	if _, err := application.Egress().SaveNode(ctx, control.Node{
		ID: "hk", Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
		Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
	}); err != nil {
		t.Fatalf("SaveNode() error = %v", err)
	}
	if _, err := application.Egress().SaveChain(ctx, control.ProxyChain{
		ID: "direct-hop", Name: "Bad", Hops: []control.ProxyHop{{Kind: control.EgressKindDirect}},
	}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("direct hop error = %v, want invalid config", err)
	}
	for _, id := range []string{"n1", "n2", "n3", "n4", "n5"} {
		if _, err := application.Egress().SaveNode(ctx, control.Node{
			ID: id, Name: id, Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
			Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
		}); err != nil {
			t.Fatalf("SaveNode(%s) error = %v", id, err)
		}
	}
	if _, err := application.Egress().SaveChain(ctx, control.ProxyChain{
		ID: "long", Name: "Long", Hops: []control.ProxyHop{
			{Kind: control.EgressKindNode, ID: "n1"},
			{Kind: control.EgressKindNode, ID: "n2"},
			{Kind: control.EgressKindNode, ID: "n3"},
			{Kind: control.EgressKindNode, ID: "n4"},
			{Kind: control.EgressKindNode, ID: "n5"},
		},
	}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("long chain error = %v, want invalid config", err)
	}
	if _, err := application.Routes().SetMode(ctx, control.RouteModeGlobal); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("global mode without egress error = %v, want invalid config", err)
	}
	profile, err := application.DNS().Get(ctx)
	if err != nil {
		t.Fatalf("DNS().Get() error = %v", err)
	}
	profile.Strategy = "ipv9"
	if _, err := application.DNS().Update(ctx, profile); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("invalid DNS error = %v, want invalid config", err)
	}
}

func TestChainRejectsUDPDisabledNodeWhenMixedUsesUDP(t *testing.T) {
	ctx := context.Background()
	application := newTestApplication(t)
	udpDisabled := false
	if _, err := application.Egress().SaveNode(ctx, control.Node{
		ID: "hk", Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
		Options:     map[string]any{"method": "aes-128-gcm", "password": "secret"},
		SupportsUDP: &udpDisabled,
	}); err != nil {
		t.Fatalf("SaveNode() error = %v", err)
	}
	if _, err := application.Egress().SaveChain(ctx, control.ProxyChain{
		ID: "relay", Name: "Relay", Hops: []control.ProxyHop{{Kind: control.EgressKindNode, ID: "hk"}},
	}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("UDP chain error = %v, want invalid config", err)
	}
}

func TestSelectGroupMemberPersistsSelection(t *testing.T) {
	ctx := context.Background()
	application := newTestApplication(t)
	for _, id := range []string{"hk", "jp"} {
		if _, err := application.Egress().SaveNode(ctx, control.Node{
			ID: id, Name: id, Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
			Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
		}); err != nil {
			t.Fatalf("SaveNode(%s) error = %v", id, err)
		}
	}
	if _, err := application.Egress().SaveGroup(ctx, control.Group{
		ID: "smart", Name: "Smart", Kind: control.GroupKindSelector, Members: []string{"hk", "jp"},
	}); err != nil {
		t.Fatalf("SaveGroup() error = %v", err)
	}
	if err := application.Egress().SelectGroupMember(ctx, "smart", "jp"); err != nil {
		t.Fatalf("SelectGroupMember() error = %v", err)
	}
	catalog, err := application.Egress().Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if len(catalog.Groups) != 1 || catalog.Groups[0].Selected != "jp" {
		t.Fatalf("persisted selection = %+v", catalog.Groups)
	}
	if err := application.Egress().SelectGroupMember(ctx, "smart", "missing"); !hasCode(err, kernel.ErrCodeNotFound) {
		t.Fatalf("SelectGroupMember(missing) error = %v, want not found", err)
	}
}

func TestProfileRollsBackWhenConfigCompilationFails(t *testing.T) {
	ctx := context.Background()
	store := &memstore.ConfigStore{}
	kernelManager, err := manager.New(ctx, manager.Options{Factory: fake.NewFactory(), Store: store})
	if err != nil {
		t.Fatalf("manager.New() error = %v", err)
	}
	t.Cleanup(func() { _ = kernelManager.Close() })
	compiler := &switchCompiler{}
	profileService := controlplane.NewProfileService(kernelManager, newMemoryProfiles(), compiler)
	application := controlplane.New(kernelManager, store, newMemorySettings(), profileService)
	if _, err := application.Setup().EnsureDefaults(ctx); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}

	compiler.fail.Store(true)
	if _, err := application.Egress().SaveNode(ctx, control.Node{
		ID: "hk", Name: "HK", Protocol: "shadowsocks", Server: "127.0.0.1", Port: 8388,
		Options: map[string]any{"method": "aes-128-gcm", "password": "secret"},
	}); !hasCode(err, kernel.ErrCodeInvalidConfig) {
		t.Fatalf("SaveNode() error = %v, want invalid config", err)
	}
	catalog, err := application.Egress().Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if len(catalog.Nodes) != 0 {
		t.Fatalf("catalog nodes = %+v, want rollback to empty", catalog.Nodes)
	}
}

type switchCompiler struct {
	fail atomic.Bool
}

func (c *switchCompiler) Compile(context.Context, control.DesiredState) (kernel.ConfigDocument, error) {
	if c.fail.Load() {
		return kernel.ConfigDocument{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "test.compile", "forced compile failure")
	}
	return kernel.ConfigDocument{Format: "json", Body: []byte(`{}`), Revision: "profile-rev"}, nil
}

func newTestApplication(t *testing.T) control.Application {
	t.Helper()
	ctx := context.Background()
	store := &memstore.ConfigStore{}
	kernelManager, err := manager.New(ctx, manager.Options{Factory: fake.NewFactory(), Store: store})
	if err != nil {
		t.Fatalf("manager.New() error = %v", err)
	}
	t.Cleanup(func() { _ = kernelManager.Close() })
	profileService := controlplane.NewProfileService(kernelManager, newMemoryProfiles(), staticCompiler{})
	application := controlplane.New(kernelManager, store, newMemorySettings(), profileService)
	if _, err := application.Setup().EnsureDefaults(ctx); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	return application
}

func hasCode(err error, code kernel.ErrorCode) bool {
	var kernelError *kernel.Error
	return errors.As(err, &kernelError) && kernelError.Code == code
}
