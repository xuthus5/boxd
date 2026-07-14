package core

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"go.etcd.io/bbolt"
)

func setupSubDB(t *testing.T) (*bbolt.DB, func()) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/sub_test.db"
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, func() {
		_ = db.Close()
	}
}

func TestNewSubscriptionManager(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	if sm == nil {
		t.Fatal("expected non-nil SubscriptionManager")
	}
}

func TestSubscriptionCreateAndList(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	sub, err := sm.Create("test-sub", "https://example.com/sub", 60)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Name != "test-sub" {
		t.Errorf("expected 'test-sub', got '%s'", sub.Name)
	}
	if sub.ID == "" {
		t.Error("expected non-empty ID")
	}

	subs := sm.List()
	if len(subs) != 1 {
		t.Fatalf("expected 1 sub, got %d", len(subs))
	}
	if subs[0].Name != "test-sub" {
		t.Errorf("expected 'test-sub', got '%s'", subs[0].Name)
	}
}

func TestSubscriptionGet(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	created, err := sm.Create("get-test", "https://example.com/sub", 30)
	if err != nil {
		t.Fatal(err)
	}

	got := sm.Get(created.ID)
	if got == nil {
		t.Fatal("expected non-nil subscription")
	}
	if got.Name != "get-test" {
		t.Errorf("expected 'get-test', got '%s'", got.Name)
	}

	missing := sm.Get("non_existent")
	if missing != nil {
		t.Error("expected nil for missing subscription")
	}
}

func TestSubscriptionUpdate(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	created, err := sm.Create("old-name", "https://old.url", 60)
	if err != nil {
		t.Fatal(err)
	}

	err = sm.Update(created.ID, "new-name", "https://new.url", 30)
	if err != nil {
		t.Fatal(err)
	}

	updated := sm.Get(created.ID)
	if updated.Name != "new-name" {
		t.Errorf("expected 'new-name', got '%s'", updated.Name)
	}
	if updated.URL != "https://new.url" {
		t.Errorf("expected 'https://new.url', got '%s'", updated.URL)
	}
	if updated.IntervalMin != 30 {
		t.Errorf("expected 30, got %d", updated.IntervalMin)
	}
}

func TestSubscriptionUpdatePartial(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	created, err := sm.Create("name", "https://url", 60)
	if err != nil {
		t.Fatal(err)
	}

	if err := sm.Update(created.ID, "", "", 0); err != nil {
		t.Fatal(err)
	}

	updated := sm.Get(created.ID)
	if updated.Name != "name" {
		t.Errorf("name should not change, got '%s'", updated.Name)
	}
	if updated.URL != "https://url" {
		t.Errorf("url should not change, got '%s'", updated.URL)
	}
	if updated.IntervalMin != 60 {
		t.Errorf("interval should not change, got %d", updated.IntervalMin)
	}
}

func TestSubscriptionUpdateNotFound(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	err := sm.Update("non_existent", "x", "x", 10)
	if err == nil {
		t.Fatal("expected error for update of non-existent subscription")
	}
}

func TestSubscriptionDelete(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	created, err := sm.Create("delete-me", "https://url", 60)
	if err != nil {
		t.Fatal(err)
	}

	if err := sm.Delete(created.ID); err != nil {
		t.Fatal(err)
	}

	if sm.Get(created.ID) != nil {
		t.Error("subscription should be nil after delete")
	}
}

func TestSubscriptionDeleteNotFound(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	err := sm.Delete("non_existent")
	if err == nil {
		t.Fatal("expected error for delete of non-existent subscription")
	}
}

func TestSubscriptionEmptyList(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	subs := sm.List()
	if len(subs) != 0 {
		t.Errorf("expected empty list, got %d items", len(subs))
	}
}

func TestSubscriptionSetError(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	created, err := sm.Create("err-test", "https://url", 60)
	if err != nil {
		t.Fatal(err)
	}

	sm.setError(created.ID, "something went wrong")

	updated := sm.Get(created.ID)
	if updated.Error != "something went wrong" {
		t.Errorf("expected 'something went wrong', got '%s'", updated.Error)
	}
}

func TestSubscriptionDB(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	if sm.DB() != db {
		t.Error("DB() should return the same db instance")
	}
}

func TestParseSubscriptionContentJSON(t *testing.T) {
	body := []byte(`{"outbounds":[{"tag":"node1","type":"vless","server":"1.2.3.4","port":443},{"tag":"node2","type":"vmess","server":"5.6.7.8","port":80}]}`)
	result := parseSubscriptionContent(body)
	if len(result) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(result))
	}
	if result[0].Tag != "node1" {
		t.Errorf("expected 'node1', got '%s'", result[0].Tag)
	}
}

func TestParseSubscriptionContentProxyLinks(t *testing.T) {
	body := []byte("vmess://eyJhZGQiOiIxLjIuMy40IiwicG9ydCI6NDQzLCJpZCI6InV1aWQxMjMiLCJhaWQiOjAsIm5ldCI6InRjcCIsInRscyI6IiIsImhvc3QiOiIiLCJwYXRoIjoiIiwicHMiOiJ0ZXN0In0=\ntrojan://pass@example.com:443?security=tls#trojan-test\n")
	result := parseSubscriptionContent(body)
	if len(result) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(result))
	}
}

func TestParseSubscriptionContentEmpty(t *testing.T) {
	result := parseSubscriptionContent([]byte{})
	if result != nil {
		t.Error("expected nil for empty content")
	}
}

func TestParseSubscriptionContentInvalid(t *testing.T) {
	result := parseSubscriptionContent([]byte("invalid content"))
	if result != nil {
		t.Error("expected nil for invalid content")
	}
}

func TestParseSubscriptionContentComments(t *testing.T) {
	body := []byte("# comment line\n// js comment\nvmess://eyJhZGQiOiIxLjIuMy40IiwicG9ydCI6NDQzLCJpZCI6InV1aWQxMjMiLCJhaWQiOjAsIm5ldCI6InRjcCIsInRscyI6IiIsImhvc3QiOiIiLCJwYXRoIjoiIiwicHMiOiJ0ZXN0In0=")
	result := parseSubscriptionContent(body)
	if len(result) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(result))
	}
}

func TestSubscriptionRefreshJSON(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	withSubscriptionHTTPClient(t, `{"outbounds":[{"tag":"json-node","type":"trojan","server":"example.com","port":443}]}`)

	sm := NewSubscriptionManager(db, t.TempDir())
	sub, err := sm.Create("json", "https://example.test/sub", 60)
	if err != nil {
		t.Fatal(err)
	}

	if err := sm.Refresh(sub.ID); err != nil {
		t.Fatal(err)
	}
	got := sm.Get(sub.ID)
	if got == nil || len(got.Outbounds) != 1 {
		t.Fatalf("outbounds = %#v", got)
	}
	if got.Error != "" {
		t.Fatalf("error = %q", got.Error)
	}
	if got.LastUpdated.IsZero() {
		t.Fatal("last updated should be set")
	}
}

func TestSubscriptionRefreshProxyLinks(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	withSubscriptionHTTPClient(t, "trojan://pass@example.com:443#trojan-test\n")

	sm := NewSubscriptionManager(db, t.TempDir())
	sub, err := sm.Create("links", "https://example.test/links", 60)
	if err != nil {
		t.Fatal(err)
	}

	if err := sm.Refresh(sub.ID); err != nil {
		t.Fatal(err)
	}
	got := sm.Get(sub.ID)
	if len(got.Outbounds) != 1 || got.Outbounds[0].Tag != "trojan-test" {
		t.Fatalf("outbounds = %#v", got.Outbounds)
	}
}

func TestSubscriptionRefreshErrors(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	sm := NewSubscriptionManager(db, t.TempDir())
	if err := sm.Refresh("missing"); err == nil {
		t.Fatal("expected missing subscription error")
	}

	sub, err := sm.Create("bad", "://bad-url", 60)
	if err != nil {
		t.Fatal(err)
	}
	if err := sm.Refresh(sub.ID); err == nil {
		t.Fatal("expected bad URL error")
	}
	got := sm.Get(sub.ID)
	if got.Error == "" {
		t.Fatal("refresh error should be stored")
	}
}

func TestSubscriptionRefreshAll(t *testing.T) {
	db, cleanup := setupSubDB(t)
	defer cleanup()

	withSubscriptionHTTPClient(t, `{"outbounds":[{"tag":"ok","type":"direct"}]}`)

	sm := NewSubscriptionManager(db, t.TempDir())
	if _, err := sm.Create("ok", "https://example.test/ok", 60); err != nil {
		t.Fatal(err)
	}
	if errs := sm.RefreshAll(); len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}

	if _, err := sm.Create("bad", "://bad-url", 60); err != nil {
		t.Fatal(err)
	}
	if errs := sm.RefreshAll(); len(errs) != 1 {
		t.Fatalf("errs = %v", errs)
	}
}

func withSubscriptionHTTPClient(t *testing.T, body string) {
	t.Helper()

	previous := subscriptionHTTPClient
	subscriptionHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{},
				Request:    req,
			}, nil
		}),
	}
	t.Cleanup(func() {
		subscriptionHTTPClient = previous
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
