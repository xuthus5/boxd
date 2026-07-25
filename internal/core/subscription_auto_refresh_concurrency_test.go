package core

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubscriptionAutoRefresherRefreshesDueSubscriptionsConcurrently(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	const extraSubscriptions = 2
	total := subscriptionRefreshConcurrency + extraSubscriptions
	started := make(chan struct{}, total)
	release := make(chan struct{})
	var active atomic.Int32
	var maxActive atomic.Int32
	var refreshCalls atomic.Int32
	var syncCalls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		refreshCalls.Add(1)
		current := active.Add(1)
		recordSubscriptionRefreshMax(&maxActive, current)
		started <- struct{}{}
		select {
		case <-release:
			active.Add(-1)
			return subscriptionResponse(`{"outbounds":[{"tag":"auto-node","type":"direct"}]}`, req), nil
		case <-req.Context().Done():
			active.Add(-1)
			return nil, req.Context().Err()
		}
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscriptions := createRefreshAllSubscriptions(t, manager, total)
	auto := newSubscriptionAutoRefresher(manager, func() error {
		syncCalls.Add(1)
		return nil
	}, 60)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan struct{})
	go func() {
		auto.tick(ctx)
		close(done)
	}()
	for range subscriptionRefreshConcurrency {
		waitForSubscriptionRefreshSignal(t, started)
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("automatic refresh did not finish")
	}

	if got := int(maxActive.Load()); got != subscriptionRefreshConcurrency {
		t.Fatalf("max active refreshes = %d, want %d", got, subscriptionRefreshConcurrency)
	}
	if got := int(refreshCalls.Load()); got != total {
		t.Fatalf("refresh calls = %d, want %d", got, total)
	}
	if got := int(syncCalls.Load()); got != 1 {
		t.Fatalf("sync calls = %d, want 1", got)
	}
	for _, subscription := range subscriptions {
		if got := manager.Get(subscription.ID); got == nil || len(got.Outbounds) != 1 {
			t.Fatalf("refreshed subscription = %#v", got)
		}
	}
}

func TestSubscriptionAutoRefresherCancellationSkipsSync(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	const pendingSubscriptions = 2
	total := subscriptionRefreshConcurrency + pendingSubscriptions
	started := make(chan struct{}, total)
	var refreshCalls atomic.Int32
	var syncCalls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		refreshCalls.Add(1)
		started <- struct{}{}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscriptions := createRefreshAllSubscriptions(t, manager, total)
	auto := newSubscriptionAutoRefresher(manager, func() error {
		syncCalls.Add(1)
		return nil
	}, 60)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan struct{})
	go func() {
		auto.tick(ctx)
		close(done)
	}()
	for range subscriptionRefreshConcurrency {
		waitForSubscriptionRefreshSignal(t, started)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("canceled automatic refresh did not finish")
	}

	if got := int(refreshCalls.Load()); got != subscriptionRefreshConcurrency {
		t.Fatalf("refresh calls = %d, want %d", got, subscriptionRefreshConcurrency)
	}
	if got := int(syncCalls.Load()); got != 0 {
		t.Fatalf("sync calls = %d, want zero", got)
	}
	for _, subscription := range subscriptions {
		if got := manager.Get(subscription.ID); got == nil || got.Error != "" {
			t.Fatalf("canceled subscription = %#v", got)
		}
	}
}

func TestSubscriptionAutoRefresherRetainsSyncAfterPartialCancellation(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	const pendingSubscriptions = 2
	total := subscriptionRefreshConcurrency + pendingSubscriptions
	started := make(chan struct{}, total)
	var syncCalls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		started <- struct{}{}
		if req.URL.Path == "/1" {
			return subscriptionResponse(`{"outbounds":[{"tag":"completed","type":"direct"}]}`, req), nil
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscriptions := createRefreshAllSubscriptions(t, manager, total)
	auto := newSubscriptionAutoRefresher(manager, func() error {
		syncCalls.Add(1)
		return nil
	}, 60)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan struct{})
	go func() {
		auto.tick(ctx)
		close(done)
	}()
	waitForSubscriptionRefreshSignal(t, started)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if subscription := manager.Get(subscriptions[0].ID); subscription != nil && len(subscription.Outbounds) == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if subscription := manager.Get(subscriptions[0].ID); subscription == nil || len(subscription.Outbounds) != 1 {
		t.Fatal("successful refresh was not persisted")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("partially canceled automatic refresh did not finish")
	}

	if got := int(syncCalls.Load()); got != 0 {
		t.Fatalf("sync calls = %d, want zero before retry", got)
	}
	if !auto.pendingSync {
		t.Fatal("successful refresh should retain pending sync after cancellation")
	}
	auto.trySync(context.Background())
	if got := int(syncCalls.Load()); got != 1 {
		t.Fatalf("sync calls after retry = %d, want 1", got)
	}
}

func TestSubscriptionAutoRefresherSyncsAfterPartialBatchFailure(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/1" {
			return subscriptionResponse(`{"outbounds":[{"tag":"success","type":"direct"}]}`, req), nil
		}
		return nil, errors.New("source unavailable")
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscriptions := createRefreshAllSubscriptions(t, manager, 2)
	var syncCalls atomic.Int32
	auto := newSubscriptionAutoRefresher(manager, func() error {
		syncCalls.Add(1)
		return nil
	}, 60)
	auto.tick(context.Background())

	if got := int(syncCalls.Load()); got != 1 {
		t.Fatalf("sync calls = %d, want 1", got)
	}
	if got := manager.Get(subscriptions[0].ID); got == nil || len(got.Outbounds) != 1 || got.Error != "" {
		t.Fatalf("successful subscription = %#v", got)
	}
	if got := manager.Get(subscriptions[1].ID); got == nil || got.Error == "" {
		t.Fatalf("failed subscription = %#v", got)
	}
}
