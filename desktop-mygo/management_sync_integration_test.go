package main

import (
	"context"
	"os"
	"testing"

	"github.com/xuthus5/boxd/internal/service"
)

func TestRuntimeBackendSubscriptionCreateSyncRollback(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testRuntimeConfig(dir)
	rt, backend := openRuntimeBackend(t, cfg)
	t.Cleanup(func() { closeRuntime(t, rt) })
	ctx := context.Background()
	original, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.ConfigPath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.CreateSubscription(ctx, service.SubscriptionInput{
		Name: "Rollback create", URL: "https://example.com/rollback-create",
	}); err == nil {
		t.Fatal("create with unsynchronizable config was accepted")
	}
	if err := os.WriteFile(cfg.ConfigPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Subscriptions) != 0 {
		t.Fatalf("failed create was not rolled back: %+v", snap.Subscriptions)
	}
}

func TestRuntimeBackendSubscriptionUpdateDeleteSyncRollback(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := testRuntimeConfig(dir)
	rt, backend := openRuntimeBackend(t, cfg)
	t.Cleanup(func() { closeRuntime(t, rt) })
	ctx := context.Background()
	original, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	created, err := backend.CreateSubscription(ctx, service.SubscriptionInput{
		Name: "Stable subscription", URL: "https://example.com/stable", IntervalMin: 30,
	})
	if err != nil {
		t.Fatalf("create stable subscription: %v", err)
	}
	if err := os.WriteFile(cfg.ConfigPath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := backend.UpdateSubscription(ctx, created.ID, service.SubscriptionInput{
		Name: "Broken update", URL: "https://example.com/broken",
	}); err == nil {
		t.Fatal("update with unsynchronizable config was accepted")
	}
	if err := os.WriteFile(cfg.ConfigPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := backend.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stable := findSubscription(snap.Subscriptions, created.ID)
	if stable == nil || stable.Name != "Stable subscription" {
		t.Fatalf("failed update was not rolled back: %+v", stable)
	}
	if err := os.WriteFile(cfg.ConfigPath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := backend.DeleteSubscription(ctx, created.ID); err == nil {
		t.Fatal("delete with unsynchronizable config was accepted")
	}
}
