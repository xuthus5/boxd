package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

const integrationNodeLink = "vless://00000000-0000-0000-0000-000000000001@edge.example.com:443?security=tls&type=tcp#e2e-node"

func testRuntimeConfig(dir string) runtime.Config {
	return runtime.Config{
		Mode: "embedded", DataDir: filepath.Join(dir, "data"),
		ConfigPath: filepath.Join(dir, "config", "config.json"),
		Username:   "admin", RefreshInterval: 60,
	}
}

func openRuntimeBackend(t *testing.T, cfg runtime.Config) (*runtime.Runtime, *runtimeBackend) {
	t.Helper()
	rt, err := runtime.Open(cfg)
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	return rt, newRuntimeBackend(rt)
}

func closeRuntime(t *testing.T, rt *runtime.Runtime) {
	t.Helper()
	if err := rt.Close(); err != nil {
		t.Fatalf("close runtime: %v", err)
	}
}

func findNode(nodes []nodeView, tag string) *nodeView {
	for i := range nodes {
		if nodes[i].Tag == tag {
			return &nodes[i]
		}
	}
	return nil
}

func findSubscription(subs []model.Subscription, id string) *model.Subscription {
	for i := range subs {
		if subs[i].ID == id {
			return &subs[i]
		}
	}
	return nil
}

func TestRuntimeBackendNodeManagement(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testRuntimeConfig(dir)
	rt, backend := openRuntimeBackend(t, cfg)
	t.Cleanup(func() { closeRuntime(t, rt) })
	ctx := context.Background()

	result, err := backend.ImportNode(ctx, integrationNodeLink)
	if err != nil {
		t.Fatalf("import node: %v", err)
	}
	snap, err := backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	node := findNode(snap.Nodes, result.Tag)
	if node == nil || node.Source != "import" || node.Server != "edge.example.com" {
		t.Fatalf("imported node = %+v", node)
	}
	assertNodeConfig(t, cfg.ConfigPath, result.Tag, true)

	updated := service.NodeInput{
		Tag: result.Tag, Type: result.Type, Server: "updated.example.com",
		Port: result.Port, Config: result.Config,
	}
	if err := backend.UpdateNode(ctx, result.Tag, updated); err != nil {
		t.Fatalf("update node: %v", err)
	}
	snap, err = backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	node = findNode(snap.Nodes, result.Tag)
	if node == nil || node.Server != "updated.example.com" {
		t.Fatalf("updated node = %+v", node)
	}

	if err := backend.DeleteNode(ctx, result.Tag); err != nil {
		t.Fatalf("delete node: %v", err)
	}
	snap, err = backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if findNode(snap.Nodes, result.Tag) != nil {
		t.Fatalf("deleted node still present: %+v", snap.Nodes)
	}
	assertNodeConfig(t, cfg.ConfigPath, result.Tag, false)
}

func assertNodeConfig(t *testing.T, path, tag string, want bool) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Contains(string(body), tag); got != want {
		t.Fatalf("synchronized config contains %q = %v, want %v", tag, got, want)
	}
}

func TestRuntimeBackendSubscriptionManagement(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testRuntimeConfig(dir)
	rt, backend := openRuntimeBackend(t, cfg)
	t.Cleanup(func() { closeRuntime(t, rt) })
	ctx := context.Background()

	created, err := backend.CreateSubscription(ctx, service.SubscriptionInput{
		Name: "E2E subscription", URL: "https://example.com/subscription", IntervalMin: 30,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	snap, err := backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if findSubscription(snap.Subscriptions, created.ID) == nil {
		t.Fatalf("created subscription missing: %+v", snap.Subscriptions)
	}

	if err := backend.UpdateSubscription(ctx, created.ID, service.SubscriptionInput{
		Name: "Updated subscription", URL: "https://example.com/updated", IntervalMin: 45,
	}); err != nil {
		t.Fatalf("update subscription: %v", err)
	}
	snap, err = backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	updated := findSubscription(snap.Subscriptions, created.ID)
	if updated == nil || updated.Name != "Updated subscription" || updated.IntervalMin != 45 {
		t.Fatalf("updated subscription = %+v", updated)
	}

	if err := backend.DeleteSubscription(ctx, created.ID); err != nil {
		t.Fatalf("delete subscription: %v", err)
	}
	snap, err = backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if findSubscription(snap.Subscriptions, created.ID) != nil {
		t.Fatalf("deleted subscription still present: %+v", snap.Subscriptions)
	}
}

func TestRuntimeBackendManagementPersistenceAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testRuntimeConfig(dir)
	ctx := context.Background()

	rt, backend := openRuntimeBackend(t, cfg)
	result, err := backend.ImportNode(ctx, integrationNodeLink)
	if err != nil {
		t.Fatalf("import node: %v", err)
	}
	created, err := backend.CreateSubscription(ctx, service.SubscriptionInput{
		Name: "Persistent subscription", URL: "https://example.com/persistent", IntervalMin: 30,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	closeRuntime(t, rt)

	rt, backend = openRuntimeBackend(t, cfg)
	t.Cleanup(func() { closeRuntime(t, rt) })
	snap, err := backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if findNode(snap.Nodes, result.Tag) == nil {
		t.Fatalf("node did not persist: %+v", snap.Nodes)
	}
	if findSubscription(snap.Subscriptions, created.ID) == nil {
		t.Fatalf("subscription did not persist: %+v", snap.Subscriptions)
	}
}

func TestRuntimeBackendManagementErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testRuntimeConfig(dir)
	rt, backend := openRuntimeBackend(t, cfg)
	t.Cleanup(func() { closeRuntime(t, rt) })
	ctx := context.Background()

	if _, err := backend.ImportNode(ctx, "not-a-proxy-link"); err == nil {
		t.Fatal("invalid proxy link was accepted")
	}
	if err := backend.SaveNode(ctx, service.NodeInput{}); err == nil {
		t.Fatal("empty node was accepted")
	}
	if err := backend.UpdateNode(ctx, "missing", service.NodeInput{Tag: "missing"}); err == nil {
		t.Fatal("missing node update was accepted")
	}
	if err := backend.DeleteNode(ctx, "missing"); err == nil {
		t.Fatal("missing node delete was accepted")
	}
	if _, err := backend.TestNode(ctx, service.TestRequest{}); err == nil {
		t.Fatal("empty test request was accepted")
	}
	if _, err := backend.TestNodes(ctx, service.TestBatchRequest{}); err == nil {
		t.Fatal("empty test batch was accepted")
	}
	if _, err := backend.CreateSubscription(ctx, service.SubscriptionInput{
		Name: "private", URL: "http://127.0.0.1/subscription",
	}); err == nil {
		t.Fatal("private subscription URL was accepted")
	}
	if err := backend.UpdateSubscription(ctx, "missing", service.SubscriptionInput{}); err == nil {
		t.Fatal("missing subscription update was accepted")
	}
	if err := backend.DeleteSubscription(ctx, "missing"); err == nil {
		t.Fatal("missing subscription delete was accepted")
	}
	if err := backend.SelectSubscription(ctx, "missing", "edge"); err == nil {
		t.Fatal("missing subscription selection was accepted")
	}
}
