package core

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

func TestSubscriptionRefreshSchedule(t *testing.T) {
	now := time.Date(2026, time.July, 25, 8, 0, 0, 0, time.UTC)
	errorAt := now.Add(-5 * time.Minute)
	tests := []struct {
		name     string
		sub      model.Subscription
		fallback int
		due      bool
	}{
		{name: "never attempted", sub: model.Subscription{IntervalMin: 60}, fallback: 60, due: true},
		{name: "successful refresh is not due", sub: model.Subscription{IntervalMin: 60, LastUpdated: now.Add(-10 * time.Minute)}, fallback: 60, due: false},
		{name: "successful refresh is due", sub: model.Subscription{IntervalMin: 60, LastUpdated: now.Add(-61 * time.Minute)}, fallback: 60, due: true},
		{name: "recent failure delays retry", sub: model.Subscription{IntervalMin: 60, LastUpdated: now.Add(-2 * time.Hour), ErrorAt: &errorAt}, fallback: 60, due: false},
		{name: "subscription interval overrides fallback", sub: model.Subscription{IntervalMin: 5, LastUpdated: now.Add(-6 * time.Minute)}, fallback: 60, due: true},
		{name: "fallback interval", sub: model.Subscription{LastUpdated: now.Add(-61 * time.Minute)}, fallback: 60, due: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interval := subscriptionRefreshInterval(test.sub, test.fallback)
			got := subscriptionRefreshDue(test.sub, now, interval)
			if got != test.due {
				t.Fatalf("due = %v, want %v (interval=%v)", got, test.due, interval)
			}
		})
	}
	if got := subscriptionRefreshInterval(model.Subscription{}, 0); got != defaultSubscriptionRefreshInterval {
		t.Fatalf("default interval = %v", got)
	}
	if got := subscriptionRefreshInterval(model.Subscription{IntervalMin: -1}, -1); got != defaultSubscriptionRefreshInterval {
		t.Fatalf("invalid interval = %v", got)
	}
	if !subscriptionRefreshDue(model.Subscription{}, now, 0) {
		t.Fatal("zero interval should still mark an unattempted subscription due")
	}
}

func TestSubscriptionAutoRefresherLifecycleDefaults(t *testing.T) {
	auto := NewSubscriptionAutoRefresher(nil, nil, 0)
	if auto.fallbackMinutes != 60 {
		t.Fatalf("fallback minutes = %d", auto.fallbackMinutes)
	}
	auto.scanInterval = time.Millisecond
	auto.Start()
	auto.Start()
	time.Sleep(5 * time.Millisecond)
	auto.Stop()
	auto.Stop()
	var nilAuto *SubscriptionAutoRefresher
	nilAuto.Start()
	nilAuto.Stop()
}

func TestSubscriptionAutoRefresherHandlesListAndContextErrors(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	manager := NewSubscriptionManager(db, t.TempDir())
	auto := newSubscriptionAutoRefresher(manager, nil, 60)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	auto.tick(ctx)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	auto.tick(context.Background())
}

func TestSubscriptionAutoRefresherTickRefreshesDueSubscriptions(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	var refreshCalls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		refreshCalls.Add(1)
		return subscriptionResponse(`{"outbounds":[{"tag":"due-node","type":"direct"}]}`, req), nil
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	due, err := manager.Create(SubscriptionParams{Name: "due", URL: "https://example.test/due", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	insertLegacySubscription(t, manager, model.Subscription{
		ID: "not-due", Name: "not-due", URL: "https://example.test/not-due", IntervalMin: 60,
		LastUpdated: now.Add(-time.Minute),
	})
	var syncCalls atomic.Int32
	auto := newSubscriptionAutoRefresher(manager, func() error {
		syncCalls.Add(1)
		return nil
	}, 60)
	auto.now = func() time.Time { return now }
	auto.tick(context.Background())

	updated := manager.Get(due.ID)
	if updated == nil || len(updated.Outbounds) != 1 || updated.Outbounds[0].Tag != "due-node" {
		t.Fatalf("due subscription = %#v", updated)
	}
	untouched := manager.Get("not-due")
	if untouched == nil || len(untouched.Outbounds) != 0 {
		t.Fatalf("not-due subscription = %#v", untouched)
	}
	if refreshCalls.Load() != 1 || syncCalls.Load() != 1 {
		t.Fatalf("refresh calls = %d, sync calls = %d", refreshCalls.Load(), syncCalls.Load())
	}
}

func TestSubscriptionAutoRefresherRetriesSyncWithoutRefreshingAgain(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	manager := NewSubscriptionManager(db, t.TempDir(), newSubscriptionTestClientForCore(
		`{"outbounds":[{"tag":"node","type":"direct"}]}`,
	))
	if _, err := manager.Create(SubscriptionParams{Name: "sync", URL: "https://example.test/sync", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	var syncCalls atomic.Int32
	auto := newSubscriptionAutoRefresher(manager, func() error {
		if syncCalls.Add(1) == 1 {
			return errors.New("sync unavailable")
		}
		return nil
	}, 60)
	auto.now = time.Now
	auto.tick(context.Background())
	auto.tick(context.Background())
	if syncCalls.Load() != 2 {
		t.Fatalf("sync calls = %d, want 2", syncCalls.Load())
	}
}

func TestSubscriptionAutoRefresherBacksOffAfterFailure(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	var refreshCalls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		refreshCalls.Add(1)
		return nil, errors.New("source unavailable")
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	if _, err := manager.Create(SubscriptionParams{Name: "failed", URL: "https://example.test/failed", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	auto := newSubscriptionAutoRefresher(manager, nil, 60)
	now := time.Now().UTC()
	auto.now = func() time.Time { return now }
	auto.tick(context.Background())
	auto.tick(context.Background())
	if refreshCalls.Load() != 1 {
		t.Fatalf("refresh calls = %d, want one attempt during interval", refreshCalls.Load())
	}
	if got := manager.Get("1"); got == nil || got.Error == "" || got.ErrorAt == nil {
		t.Fatalf("failure state = %#v", got)
	}
}

func TestSubscriptionAutoRefresherStopCancelsRefresh(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	started := make(chan struct{})
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	if _, err := manager.Create(SubscriptionParams{Name: "cancel", URL: "https://example.test/cancel", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	auto := newSubscriptionAutoRefresher(manager, nil, 60)
	auto.Start()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("automatic refresh did not start")
	}
	stopDone := make(chan struct{})
	go func() {
		auto.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("automatic refresher did not stop")
	}
	if got := manager.Get("1"); got == nil || got.Error != "" {
		t.Fatalf("cancellation should not persist an error: %#v", got)
	}
	auto.Stop()
}

func TestSubscriptionRefreshContextCancellation(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	started := make(chan struct{})
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscription, err := manager.Create(SubscriptionParams{Name: "cancel-manual", URL: "https://example.test/cancel", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	refreshDone := make(chan error, 1)
	go func() { refreshDone <- manager.RefreshContext(ctx, subscription.ID) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("manual refresh did not start")
	}
	cancel()
	select {
	case refreshErr := <-refreshDone:
		if !errors.Is(refreshErr, context.Canceled) {
			t.Fatalf("refresh error = %v", refreshErr)
		}
	case <-time.After(time.Second):
		t.Fatal("manual refresh did not cancel")
	}
	if got := manager.Get(subscription.ID); got == nil || got.Error != "" {
		t.Fatalf("cancellation should not persist an error: %#v", got)
	}
	canceled, cancelAgain := context.WithCancel(context.Background())
	cancelAgain()
	if err := manager.RefreshContext(canceled, subscription.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled refresh error = %v", err)
	}
}

func TestSubscriptionRefreshAllContextCancellation(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	manager := NewSubscriptionManager(db, t.TempDir())
	if failures := manager.RefreshAllContext(context.Background()); len(failures) != 0 {
		t.Fatalf("empty refresh failures = %#v", failures)
	}
	if _, err := manager.Create(SubscriptionParams{Name: "cancel-all", URL: "https://example.test/sub", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if failures := manager.RefreshAllContext(ctx); len(failures) != 0 {
		t.Fatalf("canceled refresh failures = %#v", failures)
	}
}

func newSubscriptionTestClientForCore(body string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return subscriptionResponse(body, req), nil
	})}
}
