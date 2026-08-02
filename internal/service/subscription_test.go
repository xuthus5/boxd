package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
)

func newTestSubService(t *testing.T) *SubscriptionService {
	t.Helper()
	db := newTestDB(t)
	configPath := t.TempDir() + "/config.json"
	if err := writeTestJSONFile(configPath, map[string]any{"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}}}); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir(), newSubscriptionTestClient(subscriptionYAMLBody))
	return NewSubscriptionService(subMgr, nodeMgr, configPath, &fakeRestart{})
}

const subscriptionYAMLBody = `proxies:
  - name: node-a
    type: vless
    server: 1.1.1.1
    port: 443
    uuid: 00000000-0000-0000-0000-000000000000
`

func TestSubscriptionCreateGetUpdateDelete(t *testing.T) {
	svc := newTestSubService(t)
	sub, err := svc.Create(context.Background(), SubscriptionInput{Name: "sub", URL: "https://example.com/sub", IntervalMin: 30})
	if err != nil {
		t.Fatal(err)
	}
	if sub.Name != "sub" {
		t.Fatalf("name = %q", sub.Name)
	}
	got, err := svc.Get(context.Background(), sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "sub" {
		t.Fatalf("name = %q", got.Name)
	}
	if err := svc.Update(context.Background(), sub.ID, SubscriptionInput{Name: "renamed", IntervalMin: 60}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), sub.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSubscriptionCreateMissingFields(t *testing.T) {
	svc := newTestSubService(t)
	_, err := svc.Create(context.Background(), SubscriptionInput{})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestSubscriptionCreateInvalidURL(t *testing.T) {
	svc := newTestSubService(t)
	_, err := svc.Create(context.Background(), SubscriptionInput{Name: "x", URL: "file:///etc/passwd"})
	if err == nil {
		t.Fatal("expected error for invalid url")
	}
}

func TestSubscriptionCreateDefaultInterval(t *testing.T) {
	svc := newTestSubService(t)
	sub, err := svc.Create(context.Background(), SubscriptionInput{Name: "x", URL: "https://example.com/sub"})
	if err != nil {
		t.Fatal(err)
	}
	if sub.IntervalMin != 60 {
		t.Fatalf("interval = %d", sub.IntervalMin)
	}
}

func TestSubscriptionGetNotFound(t *testing.T) {
	svc := newTestSubService(t)
	_, err := svc.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestSubscriptionUpdateNotFound(t *testing.T) {
	svc := newTestSubService(t)
	if err := svc.Update(context.Background(), "missing", SubscriptionInput{}); err == nil {
		t.Fatal("expected not found")
	}
}

func TestSubscriptionDeleteNotFound(t *testing.T) {
	svc := newTestSubService(t)
	if err := svc.Delete(context.Background(), "missing"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestSubscriptionList(t *testing.T) {
	svc := newTestSubService(t)
	if _, err := svc.Create(context.Background(), SubscriptionInput{Name: "a", URL: "https://example.com/a"}); err != nil {
		t.Fatal(err)
	}
	subs, err := svc.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 {
		t.Fatalf("subs = %d", len(subs))
	}
}

func TestSubscriptionRefresh(t *testing.T) {
	svc := newTestSubService(t)
	sub, err := svc.Create(context.Background(), SubscriptionInput{Name: "sub", URL: "https://example.com/sub", IntervalMin: 60})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Refresh(context.Background(), sub.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSubscriptionRefreshNotFound(t *testing.T) {
	svc := newTestSubService(t)
	if err := svc.Refresh(context.Background(), "missing"); err == nil {
		t.Fatal("expected refresh error")
	}
}

func TestSubscriptionRefreshAll(t *testing.T) {
	svc := newTestSubService(t)
	if _, err := svc.Create(context.Background(), SubscriptionInput{Name: "a", URL: "https://example.com/a"}); err != nil {
		t.Fatal(err)
	}
	result, err := svc.RefreshAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.SyncFailed {
		t.Fatalf("result = %+v", result)
	}
}

func TestSubscriptionSyncErrorMessage(t *testing.T) {
	if got := subscriptionSyncErrorMessage(nil); got != "subscription refreshed but configuration sync failed" {
		t.Fatalf("got %q", got)
	}
	if got := subscriptionSyncErrorMessage(nil); !strings.Contains(got, "configuration sync failed") {
		t.Fatalf("got %q", got)
	}
}

func TestSubscriptionUpdateInvalidURL(t *testing.T) {
	svc := newTestSubService(t)
	sub, err := svc.Create(context.Background(), SubscriptionInput{Name: "a", URL: "https://example.com/a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(context.Background(), sub.ID, SubscriptionInput{URL: "file:///etc/passwd"}); err == nil {
		t.Fatal("expected invalid url error")
	}
}

func TestSubscriptionSyncConfigNilManagers(t *testing.T) {
	svc := NewSubscriptionService(nil, nil, "", nil)
	if err := svc.syncConfig(); err != nil {
		t.Fatal(err)
	}
}

type subscriptionRoundTripFunc func(*http.Request) (*http.Response, error)

func (f subscriptionRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newSubscriptionTestClient(body string) *http.Client {
	return &http.Client{Transport: subscriptionRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
			Header:        make(http.Header),
			Request:       req,
		}, nil
	})}
}
