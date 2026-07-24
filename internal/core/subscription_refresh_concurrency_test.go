package core

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

func TestSubscriptionRefreshSkipsResultAfterSubscriptionUpdate(t *testing.T) {
	tests := []refreshUpdateRaceCase{
		{
			name:   "url",
			update: SubscriptionParams{URL: "https://example.test/new"},
			assert: func(t *testing.T, got *model.Subscription) {
				t.Helper()
				if got.URL != "https://example.test/new" {
					t.Fatalf("url = %q", got.URL)
				}
			},
		},
		{
			name:   "urltest",
			update: SubscriptionParams{URLTest: &model.URLTestOverrides{Interval: stringPointer("5m")}},
			assert: func(t *testing.T, got *model.Subscription) {
				t.Helper()
				if got.URLTest == nil || got.URLTest.Interval == nil || *got.URLTest.Interval != "5m" {
					t.Fatalf("urltest = %#v", got.URLTest)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { runRefreshUpdateRace(t, test) })
	}
}

type refreshUpdateRaceCase struct {
	name   string
	update SubscriptionParams
	assert func(*testing.T, *model.Subscription)
}

func runRefreshUpdateRace(t *testing.T, test refreshUpdateRaceCase) {
	t.Helper()
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)
	client, started, release := newBlockingSubscriptionClient(t,
		`{"outbounds":[{"tag":"stale-node","type":"direct"}]}`)
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscription, err := manager.Create(SubscriptionParams{
		Name: "race", URL: "https://example.test/old", IntervalMin: 60,
		URLTest: &model.URLTestOverrides{Interval: stringPointer("3m")},
	})
	if err != nil {
		t.Fatal(err)
	}

	refreshDone := make(chan error, 1)
	go func() { refreshDone <- manager.Refresh(subscription.ID) }()
	waitForSubscriptionRefreshSignal(t, started)
	if err := manager.Update(subscription.ID, test.update); err != nil {
		t.Fatal(err)
	}
	release()
	if err := waitForSubscriptionRefreshResult(t, refreshDone); err != nil {
		t.Fatalf("refresh error after superseding update: %v", err)
	}

	got := manager.Get(subscription.ID)
	if got == nil || len(got.Outbounds) != 0 {
		t.Fatalf("stale refresh result was persisted: %#v", got)
	}
	test.assert(t, got)
}

func TestSubscriptionRefreshSkipsStaleErrorAfterSubscriptionUpdate(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	started := make(chan struct{})
	release, releaseRefresh := newSubscriptionRefreshGate(t)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(started)
		select {
		case <-release:
			return nil, errors.New("old source failed")
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscription, err := manager.Create(SubscriptionParams{
		Name: "error-race", URL: "https://example.test/old", IntervalMin: 60,
	})
	if err != nil {
		t.Fatal(err)
	}

	refreshDone := make(chan error, 1)
	go func() { refreshDone <- manager.Refresh(subscription.ID) }()
	waitForSubscriptionRefreshSignal(t, started)
	if err := manager.Update(subscription.ID, SubscriptionParams{URL: "https://example.test/new"}); err != nil {
		t.Fatal(err)
	}
	releaseRefresh()
	if err := waitForSubscriptionRefreshResult(t, refreshDone); err != nil {
		t.Fatalf("refresh error after superseding update: %v", err)
	}

	got := manager.Get(subscription.ID)
	if got == nil || got.Error != "" || got.ErrorCode != "" || got.ErrorAt != nil {
		t.Fatalf("stale refresh error was persisted: %#v", got)
	}
}

func TestSubscriptionRefreshFirstCompletedResultWins(t *testing.T) {
	db, cleanup := setupSubDB(t)
	t.Cleanup(cleanup)

	started := make(chan int, 2)
	releaseFirst, releaseFirstRefresh := newSubscriptionRefreshGate(t)
	releaseSecond, releaseSecondRefresh := newSubscriptionRefreshGate(t)
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		call := int(calls.Add(1))
		started <- call
		if call == 1 {
			<-releaseFirst
			return subscriptionResponse(`{"outbounds":[{"tag":"first","type":"direct"}]}`, req), nil
		}
		<-releaseSecond
		return subscriptionResponse(`{"outbounds":[{"tag":"second","type":"direct"}]}`, req), nil
	})}
	manager := NewSubscriptionManager(db, t.TempDir(), client)
	subscription, err := manager.Create(SubscriptionParams{
		Name: "duplicate", URL: "https://example.test/same", IntervalMin: 60,
	})
	if err != nil {
		t.Fatal(err)
	}

	refreshDone := make(chan error, 2)
	go func() { refreshDone <- manager.Refresh(subscription.ID) }()
	go func() { refreshDone <- manager.Refresh(subscription.ID) }()
	waitForSubscriptionRefreshSignal(t, started)
	waitForSubscriptionRefreshSignal(t, started)
	releaseFirstRefresh()
	if err := waitForSubscriptionRefreshResult(t, refreshDone); err != nil {
		t.Fatalf("first refresh error: %v", err)
	}
	releaseSecondRefresh()
	if err := waitForSubscriptionRefreshResult(t, refreshDone); err != nil {
		t.Fatalf("second refresh error: %v", err)
	}

	got := manager.Get(subscription.ID)
	if got == nil || len(got.Outbounds) != 1 || got.Outbounds[0].Tag != "first" {
		t.Fatalf("final outbounds = %#v", got)
	}
}

func newBlockingSubscriptionClient(t *testing.T, body string) (*http.Client, <-chan struct{}, func()) {
	t.Helper()
	started := make(chan struct{})
	release, releaseRefresh := newSubscriptionRefreshGate(t)
	var startedOnce atomic.Bool
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if startedOnce.CompareAndSwap(false, true) {
			close(started)
		}
		select {
		case <-release:
			return subscriptionResponse(body, req), nil
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	})}
	return client, started, releaseRefresh
}

func newSubscriptionRefreshGate(t *testing.T) (<-chan struct{}, func()) {
	t.Helper()
	release := make(chan struct{})
	var once sync.Once
	releaseRefresh := func() { once.Do(func() { close(release) }) }
	t.Cleanup(releaseRefresh)
	return release, releaseRefresh
}

func subscriptionResponse(body string, req *http.Request) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          io.NopCloser(strings.NewReader(body)),
		Header:        make(http.Header),
		Request:       req,
		ContentLength: int64(len(body)),
	}
}

func waitForSubscriptionRefreshSignal[T any](t *testing.T, signal <-chan T) T {
	t.Helper()
	select {
	case value := <-signal:
		return value
	case <-time.After(time.Second):
		t.Fatal("subscription refresh did not start")
		var zero T
		return zero
	}
}

func waitForSubscriptionRefreshResult(t *testing.T, results <-chan error) error {
	t.Helper()
	select {
	case err := <-results:
		return err
	case <-time.After(time.Second):
		t.Fatal("subscription refresh did not finish")
		return nil
	}
}

func stringPointer(value string) *string { return &value }
