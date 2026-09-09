package core

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
)

func TestRuleSetCacheMatchesSingBox114Encoding(t *testing.T) {
	native := &adapter.SavedBinary{Content: []byte("kernel cache"), LastUpdated: time.Unix(1700000000, 0), LastEtag: "etag", URLHash: ruleSetURLHash("https://example.org/a.srs")}
	payload, err := native.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var decoded savedRuleSetBinary
	if err := decoded.UnmarshalBinary(payload); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.URLHash, native.URLHash) || decoded.LastEtag != native.LastEtag {
		t.Fatalf("decoded = %#v", decoded)
	}
	reencoded, err := decoded.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, reencoded) {
		t.Fatal("boxd cache is not byte-compatible with the 1.14 kernel")
	}
	for cut := range len(payload) {
		if err := decoded.UnmarshalBinary(payload[:cut]); err == nil {
			t.Fatalf("truncated payload at %d accepted", cut)
		}
	}
}

func TestRuleSetCacheEnforcesBoundedMetadata(t *testing.T) {
	for _, saved := range []*savedRuleSetBinary{
		{LastEtag: strings.Repeat("x", maxRuleSetEtagBytes+1)},
		{URLHash: make([]byte, len(ruleSetURLHash("url"))+1)},
	} {
		if _, err := saved.MarshalBinary(); err == nil {
			t.Fatal("oversized cache metadata accepted")
		}
	}
	for _, native := range []*adapter.SavedBinary{
		{LastEtag: strings.Repeat("x", maxRuleSetEtagBytes+1)},
		{URLHash: make([]byte, len(ruleSetURLHash("url"))+1)},
	} {
		payload, err := native.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var decoded savedRuleSetBinary
		if err := decoded.UnmarshalBinary(payload); err == nil {
			t.Fatal("oversized kernel cache metadata accepted")
		}
	}
}

func TestRuleSetUpdaterDoesNotReuseETagAfterURLChanges(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("If-None-Match") != "" {
			t.Error("old URL etag was sent to a new resource")
		}
		w.Header().Set("Etag", "new-etag")
		_, _ = w.Write([]byte("new rules"))
	}))
	t.Cleanup(server.Close)
	url := rulesetTestURL(server, "/new.srs")
	updater := newRuleSet114Updater(t, map[string]any{"route": map[string]any{"rule_set": []any{map[string]any{"type": "remote", "tag": "remote", "url": url}}}})
	updater.client = rulesetTestClient(server)
	if err := updater.saveRemoteCache("remote", &savedRuleSetBinary{Content: []byte("old"), LastEtag: "old-etag", URLHash: ruleSetURLHash("https://old.example.org/a.srs")}); err != nil {
		t.Fatal(err)
	}
	status, err := updater.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if status[0].LastEtag != "" || status[0].LastUpdated != nil {
		t.Fatalf("stale URL cache was displayed: %#v", status[0])
	}
	result, err := updater.Update(t.Context(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdatedCount != 1 || requests.Load() != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRuleSetUpdaterRejects304WithoutMatchingCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotModified) }))
	t.Cleanup(server.Close)
	updater := newRuleSet114Updater(t, map[string]any{"route": map[string]any{"rule_set": []any{map[string]any{"type": "remote", "tag": "remote", "url": rulesetTestURL(server, "/a.srs")}}}})
	updater.client = rulesetTestClient(server)
	result, err := updater.Update(t.Context(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FailedCount != 1 || result.Results[0].ErrorCode != RuleSetErrorHTTP {
		t.Fatalf("result = %#v", result)
	}
}

func TestRuleSetBatchUpdatesSharedLocalFileOnce(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("example.org\n"))
	}))
	t.Cleanup(server.Close)
	path := filepath.Join(t.TempDir(), "shared.json")
	updater := newRuleSet114Updater(t, map[string]any{"route": map[string]any{"rule_set": []any{
		map[string]any{"type": "local", "tag": "a", "path": path}, map[string]any{"type": "local", "tag": "b", "path": path},
	}}})
	updater.installer.client = rulesetTestClient(server)
	updater.installer.sources = []RuleSetSource{{Tag: "a", URL: rulesetTestURL(server, "/domains.txt")}, {Tag: "b", URL: rulesetTestURL(server, "/domains.txt")}}
	result, err := updater.Update(t.Context(), RuleSetUpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdatedCount != 1 || requests.Load() != 1 || !result.Results[1].NotModified {
		t.Fatalf("result = %#v requests = %d", result, requests.Load())
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("updated file metadata = %v %v", info, err)
	}
}

func TestCopyRuleSetCacheReportsMissingSource(t *testing.T) {
	updater := newRuleSet114Updater(t, map[string]any{})
	if err := updater.copyRemoteRuleSetCache("a", "b"); !errors.Is(err, ErrRuleSetCacheDisabled) {
		t.Fatalf("missing db = %v", err)
	}
	if err := updater.saveRemoteCache("other", &savedRuleSetBinary{Content: []byte("data")}); err != nil {
		t.Fatal(err)
	}
	if err := updater.copyRemoteRuleSetCache("a", "b"); !errors.Is(err, ErrRuleSetCacheDisabled) {
		t.Fatalf("missing tag = %v", err)
	}
}
