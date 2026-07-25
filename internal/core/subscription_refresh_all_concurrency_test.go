package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

func TestSubscriptionRefreshAllContextReportsListFailure(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	manager := NewSubscriptionManager(db, t.TempDir())
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	failures := manager.RefreshAllContext(context.Background())
	if len(failures) != 1 || failures[0].Code != SubRefreshUnknown {
		t.Fatalf("failures = %#v", failures)
	}
}

func TestSubscriptionRefreshAllContextLimitsConcurrency(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	const extraSubscriptions = 2
	total := subscriptionRefreshConcurrency + extraSubscriptions
	started := make(chan struct{}, total)
	release := make(chan struct{})
	var active atomic.Int32
	var maxActive atomic.Int32
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		current := active.Add(1)
		recordSubscriptionRefreshMax(&maxActive, current)
		started <- struct{}{}
		select {
		case <-release:
			active.Add(-1)
			return subscriptionResponse(`{"outbounds":[{"tag":"node","type":"direct"}]}`, req), nil
		case <-req.Context().Done():
			active.Add(-1)
			return nil, req.Context().Err()
		}
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	createRefreshAllSubscriptions(t, manager, total)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan []SubscriptionRefreshFailure, 1)
	go func() { done <- manager.RefreshAllContext(ctx) }()
	for range subscriptionRefreshConcurrency {
		waitForSubscriptionRefreshSignal(t, started)
	}
	close(release)

	if failures := waitForSubscriptionRefreshSignal(t, done); len(failures) != 0 {
		t.Fatalf("failures = %#v", failures)
	}
	if got := int(maxActive.Load()); got != subscriptionRefreshConcurrency {
		t.Fatalf("max active refreshes = %d, want %d", got, subscriptionRefreshConcurrency)
	}
	if got := int(calls.Load()); got != total {
		t.Fatalf("refresh calls = %d, want %d", got, total)
	}
}

func TestSubscriptionRefreshAllContextPreservesFailureOrder(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	const total = 3
	releases := make([]chan struct{}, total)
	started := make(chan int, total)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		index, err := subscriptionRefreshRequestIndex(req, total)
		if err != nil {
			return nil, err
		}
		started <- index
		select {
		case <-releases[index]:
			return nil, fmt.Errorf("source %d failed", index+1)
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscriptions := createRefreshAllSubscriptions(t, manager, total)
	for index := range releases {
		releases[index] = make(chan struct{})
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan []SubscriptionRefreshFailure, 1)
	go func() { done <- manager.RefreshAllContext(ctx) }()
	for range total {
		waitForSubscriptionRefreshSignal(t, started)
	}
	for index := total - 1; index >= 0; index-- {
		close(releases[index])
		waitForSubscriptionRefreshError(t, manager, subscriptions[index].ID)
	}

	failures := waitForSubscriptionRefreshSignal(t, done)
	if len(failures) != total {
		t.Fatalf("failures = %#v", failures)
	}
	for index, failure := range failures {
		if failure.ID != subscriptions[index].ID {
			t.Fatalf("failure %d id = %q, want %q", index, failure.ID, subscriptions[index].ID)
		}
	}
}

func TestSubscriptionRefreshAllContextKeepsCompletedFailuresOnCancellation(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	total := subscriptionRefreshConcurrency + 1
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/1" {
			return nil, errors.New("completed failure")
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscriptions := createRefreshAllSubscriptions(t, manager, total)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan []SubscriptionRefreshFailure, 1)
	go func() { done <- manager.RefreshAllContext(ctx) }()
	waitForSubscriptionRefreshError(t, manager, subscriptions[0].ID)
	cancel()

	failures := waitForSubscriptionRefreshSignal(t, done)
	if len(failures) != 1 || failures[0].ID != subscriptions[0].ID {
		t.Fatalf("failures = %#v", failures)
	}
	for _, subscription := range subscriptions[1:] {
		if got := manager.Get(subscription.ID); got == nil || got.Error != "" {
			t.Fatalf("canceled subscription = %#v", got)
		}
	}
}

func TestSubscriptionRefreshAllContextStopsDispatchOnCancellation(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	const pendingSubscriptions = 2
	total := subscriptionRefreshConcurrency + pendingSubscriptions
	started := make(chan struct{}, total)
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		started <- struct{}{}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscriptions := createRefreshAllSubscriptions(t, manager, total)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan []SubscriptionRefreshFailure, 1)
	go func() { done <- manager.RefreshAllContext(ctx) }()
	for range subscriptionRefreshConcurrency {
		waitForSubscriptionRefreshSignal(t, started)
	}
	cancel()

	if failures := waitForSubscriptionRefreshSignal(t, done); len(failures) != 0 {
		t.Fatalf("failures = %#v", failures)
	}
	if got := int(calls.Load()); got != subscriptionRefreshConcurrency {
		t.Fatalf("refresh calls = %d, want %d", got, subscriptionRefreshConcurrency)
	}
	for _, subscription := range subscriptions {
		if got := manager.Get(subscription.ID); got == nil || got.Error != "" {
			t.Fatalf("canceled subscription = %#v", got)
		}
	}
}

func createRefreshAllSubscriptions(
	t *testing.T,
	manager *SubscriptionManager,
	total int,
) []*model.Subscription {
	t.Helper()
	subscriptions := make([]*model.Subscription, 0, total)
	for index := 1; index <= total; index++ {
		subscription, err := manager.Create(SubscriptionParams{
			Name:        fmt.Sprintf("subscription-%d", index),
			URL:         fmt.Sprintf("https://example.test/%d", index),
			IntervalMin: 60,
		})
		if err != nil {
			t.Fatal(err)
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions
}

func subscriptionRefreshRequestIndex(req *http.Request, total int) (int, error) {
	var index int
	if _, err := fmt.Sscanf(req.URL.Path, "/%d", &index); err != nil {
		return 0, fmt.Errorf("parse request path %q: %w", req.URL.Path, err)
	}
	index--
	if index < 0 || index >= total {
		return 0, fmt.Errorf("request path %q is out of range", req.URL.Path)
	}
	return index, nil
}

func waitForSubscriptionRefreshError(t *testing.T, manager *SubscriptionManager, id string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if subscription := manager.Get(id); subscription != nil && subscription.Error != "" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("subscription %s refresh error was not persisted", id)
}

func recordSubscriptionRefreshMax(target *atomic.Int32, value int32) {
	for {
		current := target.Load()
		if value <= current || target.CompareAndSwap(current, value) {
			return
		}
	}
}
