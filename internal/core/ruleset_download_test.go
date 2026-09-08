package core

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRaceDownloadContentEmptyBodyRejected(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("")),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	if _, err := installer.raceDownloadContent(context.Background(), "t", "https://raw.githubusercontent.com/a/b/c.txt"); err == nil {
		t.Fatal("empty body must be rejected")
	}
}

func TestUniqueStringsAndFetchErrors(t *testing.T) {
	got := uniqueStrings([]string{"a", "a", "", "b", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got = %#v", got)
	}
	installer := &LoyalsoldierRuleSetInstaller{client: http.DefaultClient, ruleSetDir: t.TempDir()}
	if _, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "x", URL: "://bad"}); err == nil {
		t.Fatal("expected bad url error")
	}
}

func TestRuleSetInstallerRejectsOversizedSource(t *testing.T) {
	installer := &LoyalsoldierRuleSetInstaller{
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: maxRuleSetBodyBytes + 1,
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader("unused")),
				Request:       req,
			}, nil
		})},
		ruleSetDir: t.TempDir(),
	}
	_, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "large", URL: "https://ruleset.test/source.txt"})
	if !errors.Is(err, ErrRuleSetContentTooLarge) {
		t.Fatalf("error = %v, want oversized error", err)
	}
}

func TestJsdelivrMirrorURLs(t *testing.T) {
	rawURL := "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"
	got := jsdelivrMirrorURLs(rawURL)
	want := []string{
		"https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/direct-list.txt",
		"https://fastly.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/direct-list.txt",
	}
	if len(got) != len(want) {
		t.Fatalf("mirrors = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mirror[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if jsdelivrMirrorURLs("https://example.com/a.txt") != nil {
		t.Fatal("non raw.githubusercontent.com URL must have no mirrors")
	}
	if jsdelivrMirrorURLs("https://raw.githubusercontent.com/owner/repo/branch") != nil {
		t.Fatal("URL without file path must have no mirrors")
	}
}

func TestRuleSetSourceURLsOrder(t *testing.T) {
	primary := "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/proxy-list.txt"
	urls := ruleSetSourceURLs(primary)
	wantLen := 3
	if len(urls) != wantLen {
		t.Fatalf("urls = %#v, want 3 entries", urls)
	}
	if urls[0] != primary {
		t.Fatalf("primary = %q, want %q", urls[0], primary)
	}
	if got, want := urls[1], "https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/proxy-list.txt"; got != want {
		t.Fatalf("mirror[1] = %q, want %q", got, want)
	}
	if noMirror := ruleSetSourceURLs("https://example.com/feed"); len(noMirror) != 1 {
		t.Fatalf("non-raw URL should keep single source, got %#v", noMirror)
	}
}

func TestFetchAndConvertUsesPrimaryWhenHealthy(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		calls = append(calls, req.URL.String())
		mu.Unlock()
		if req.URL.Host != "raw.githubusercontent.com" {
			return &http.Response{
				StatusCode:    http.StatusServiceUnavailable,
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: -1,
				Request:       req,
			}, nil
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("# c\nexample.cn\n")),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	rule, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "t", URL: "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"})
	if err != nil {
		t.Fatalf("fetchAndConvert() error = %v", err)
	}
	if len(rule.Rules) != 1 || len(rule.Rules[0].DomainSuffix) != 1 || rule.Rules[0].DomainSuffix[0] != "example.cn" {
		t.Fatalf("rule = %#v", rule.Rules)
	}
	if len(calls) != 3 {
		t.Fatalf("all three sources race, calls = %#v", calls)
	}
	for _, u := range calls {
		if !strings.HasPrefix(u, "https://raw.githubusercontent.com/") &&
			!strings.HasPrefix(u, "https://cdn.jsdelivr.net/") &&
			!strings.HasPrefix(u, "https://fastly.jsdelivr.net/") {
			t.Fatalf("unexpected call %q", u)
		}
	}
}

func TestFetchAndConvertFallsBackToMirror(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var status int
		switch req.URL.Host {
		case "raw.githubusercontent.com":
			status = http.StatusServiceUnavailable
		case "cdn.jsdelivr.net":
			status = http.StatusOK
		case "fastly.jsdelivr.net":
			status = http.StatusServiceUnavailable
		default:
			status = http.StatusNotFound
		}
		mu.Lock()
		counts[req.URL.Host]++
		mu.Unlock()
		body := ""
		if status == http.StatusOK {
			body = "full:exact.example\n"
		}
		return &http.Response{
			StatusCode:    status,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	rule, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "t", URL: "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"})
	if err != nil {
		t.Fatalf("fetchAndConvert() error = %v", err)
	}
	if counts["raw.githubusercontent.com"] != 1 || counts["cdn.jsdelivr.net"] != 1 || counts["fastly.jsdelivr.net"] != 1 {
		t.Fatalf("unexpected request counts: %#v", counts)
	}
	if len(rule.Rules[0].Domain) != 1 || rule.Rules[0].Domain[0] != "exact.example" {
		t.Fatalf("rule = %#v", rule.Rules)
	}
}

func TestFetchAndConvertRacePrefersFastestSource(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		delay := 200 * time.Millisecond
		if req.URL.Host == "cdn.jsdelivr.net" {
			delay = 10 * time.Millisecond
		}
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("fast.example.cn\n")),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	start := time.Now()
	rule, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "t", URL: "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"})
	if err != nil {
		t.Fatalf("fetchAndConvert() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed >= 150*time.Millisecond {
		t.Fatalf("race should finish at the speed of the fastest source, took %v", elapsed)
	}
	if len(rule.Rules[0].DomainSuffix) != 1 || rule.Rules[0].DomainSuffix[0] != "fast.example.cn" {
		t.Fatalf("rule = %#v", rule.Rules)
	}
}

func TestFetchAndConvertAllSourcesFail(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		calls = append(calls, req.URL.String())
		mu.Unlock()
		return &http.Response{
			StatusCode:    http.StatusServiceUnavailable,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader("")),
			ContentLength: -1,
			Request:       req,
		}, nil
	})}
	installer := &LoyalsoldierRuleSetInstaller{client: client, ruleSetDir: t.TempDir()}
	_, err := installer.fetchAndConvert(context.Background(), RuleSetSource{Tag: "t", URL: "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/reject-list.txt"})
	if err == nil {
		t.Fatal("expected error when all sources fail")
	}
	if len(calls) != 3 {
		t.Fatalf("calls = %d, want 3 (primary+cdn+fastly), got %#v", len(calls), calls)
	}
	for _, u := range calls {
		switch {
		case strings.HasPrefix(u, "https://raw.githubusercontent.com/"),
			strings.HasPrefix(u, "https://cdn.jsdelivr.net/"),
			strings.HasPrefix(u, "https://fastly.jsdelivr.net/"):
		default:
			t.Fatalf("unexpected call %q", u)
		}
	}
}
