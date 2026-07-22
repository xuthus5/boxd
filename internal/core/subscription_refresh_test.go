package core

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSubscriptionRefreshJSON(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	withSubscriptionHTTPClient(t, `{"outbounds":[{"tag":"json-node","type":"trojan","server":"example.com","port":443}]}`)
	manager := NewSubscriptionManager(db, t.TempDir())
	subscription, err := manager.Create(SubscriptionParams{Name: "json", URL: "https://example.test/sub", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Refresh(subscription.ID); err != nil {
		t.Fatal(err)
	}
	got := manager.Get(subscription.ID)
	if got == nil || len(got.Outbounds) != 1 || got.Error != "" || got.LastUpdated.IsZero() {
		t.Fatalf("refreshed subscription = %#v", got)
	}
}

func TestSubscriptionRefreshProxyLinks(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	withSubscriptionHTTPClient(t, "trojan://pass@example.com:443#trojan-test\n")
	manager := NewSubscriptionManager(db, t.TempDir())
	subscription, err := manager.Create(SubscriptionParams{Name: "links", URL: "https://example.test/links", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Refresh(subscription.ID); err != nil {
		t.Fatal(err)
	}
	got := manager.Get(subscription.ID)
	if len(got.Outbounds) != 1 || got.Outbounds[0].Tag != "trojan-test" {
		t.Fatalf("outbounds = %#v", got.Outbounds)
	}
}

func TestSubscriptionRefreshErrors(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	manager := NewSubscriptionManager(db, t.TempDir())
	if err := manager.Refresh("missing"); err == nil {
		t.Fatal("expected missing subscription error")
	}
	subscription, err := manager.Create(SubscriptionParams{Name: "bad", URL: "://bad-url", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Refresh(subscription.ID); err == nil {
		t.Fatal("expected bad URL error")
	}
	if got := manager.Get(subscription.ID); got.Error == "" {
		t.Fatal("refresh error should be stored")
	}
}

func TestSubscriptionRefreshAll(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	withSubscriptionHTTPClient(t, `{"outbounds":[{"tag":"ok","type":"direct"}]}`)
	manager := NewSubscriptionManager(db, t.TempDir())
	if _, err := manager.Create(SubscriptionParams{Name: "ok", URL: "https://example.test/ok", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	if errs := manager.RefreshAll(); len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if _, err := manager.Create(SubscriptionParams{Name: "bad", URL: "://bad-url", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	if errs := manager.RefreshAll(); len(errs) != 1 {
		t.Fatalf("errs = %v", errs)
	}
}

func TestSubscriptionRefreshUserinfo(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	withSubscriptionHTTPClientHeader(t,
		`{"outbounds":[{"tag":"node","type":"direct"}]}`,
		http.Header{"Subscription-Userinfo": []string{"upload=1; download=2; total=10; expire=1719859200"}},
	)
	manager := NewSubscriptionManager(db, t.TempDir())
	subscription, err := manager.Create(SubscriptionParams{Name: "traffic", URL: "https://example.test/traffic", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Refresh(subscription.ID); err != nil {
		t.Fatal(err)
	}
	got := manager.Get(subscription.ID)
	if got == nil || got.Traffic == nil {
		t.Fatalf("expected traffic, got %#v", got)
	}
	if got.Traffic.Upload != 1 || got.Traffic.Download != 2 || got.Traffic.Total != 10 {
		t.Fatalf("traffic = %#v", got.Traffic)
	}
	if got.Traffic.Expire == nil {
		t.Fatal("expected expire")
	}
}

func withSubscriptionHTTPClient(t *testing.T, body string) {
	t.Helper()
	withSubscriptionHTTPClientHeader(t, body, http.Header{})
}

func withSubscriptionHTTPClientHeader(t *testing.T, body string, header http.Header) {
	t.Helper()
	previous := subscriptionHTTPClient
	if header == nil {
		header = http.Header{}
	}
	subscriptionHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     header.Clone(),
			Request:    req,
		}, nil
	})}
	t.Cleanup(func() { subscriptionHTTPClient = previous })
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
