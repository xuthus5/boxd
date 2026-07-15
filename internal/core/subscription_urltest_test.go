package core

import (
	"testing"

	"github.com/xuthus5/boxui/internal/model"
)

func TestSubscriptionURLTestOverrides(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	disabled := false
	zeroTolerance := uint16(0)
	customURL := "https://example.com/generate_204"
	manager := NewSubscriptionManager(db, t.TempDir())
	created, err := manager.Create(SubscriptionParams{
		Name: "custom", URL: "https://example.com/sub", IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Enabled: &disabled, URL: &customURL, Tolerance: &zeroTolerance},
	})
	if err != nil {
		t.Fatalf("creating subscription: %v", err)
	}
	if created.URLTest == nil || created.URLTest.Enabled == nil || *created.URLTest.Enabled {
		t.Fatalf("created urltest overrides = %#v", created.URLTest)
	}
	if created.URLTest.Tolerance == nil || *created.URLTest.Tolerance != 0 {
		t.Fatalf("created tolerance = %#v", created.URLTest.Tolerance)
	}

	if err := manager.Update(created.ID, SubscriptionParams{URLTest: nil}); err != nil {
		t.Fatalf("clearing overrides: %v", err)
	}
	updated := manager.Get(created.ID)
	if updated == nil || updated.URLTest != nil {
		t.Fatalf("updated urltest overrides = %#v", updated)
	}
}

func TestSubscriptionRejectsInvalidURLTestOverrides(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	invalidInterval := "0s"
	manager := NewSubscriptionManager(db, t.TempDir())
	_, err := manager.Create(SubscriptionParams{
		Name: "invalid", URL: "https://example.com/sub", IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Interval: &invalidInterval},
	})
	if err == nil {
		t.Fatal("expected invalid urltest override error")
	}
}
