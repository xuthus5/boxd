package core

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuleSetOnlineInstallFailures(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"directory", "download", "write"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			installer := newRuleSetInstallTestInstaller(t)
			switch name {
			case "directory":
				if err := os.WriteFile(installer.RuleSetDir(), []byte("block"), 0600); err != nil {
					t.Fatal(err)
				}
			case "download":
				installer.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
					return nil, errors.New("download unavailable")
				})
			case "write":
				if err := os.MkdirAll(filepath.Join(installer.RuleSetDir(), "geoip-cn.srs"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if entries, err := installer.Install(t.Context()); err == nil || entries != nil {
				t.Fatalf("entries = %v, error = %v, want complete failure", entries, err)
			}
			assertNoRuleSetTemporaryFiles(t, installer.RuleSetDir())
		})
	}
}

func TestRuleSetOnlineInstallCancellationBeforeWrite(t *testing.T) {
	t.Parallel()
	installer := newRuleSetInstallTestInstaller(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	transport := installer.client.Transport
	installer.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cancel()
		return transport.RoundTrip(req)
	})
	if entries, err := installer.Install(ctx); !errors.Is(err, context.Canceled) || entries != nil {
		t.Fatalf("entries = %v, error = %v, want canceled", entries, err)
	}
	if _, err := os.Stat(filepath.Join(installer.RuleSetDir(), "geoip-cn.srs")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled installer wrote a file: %v", err)
	}
}

func TestRuleSetUpdaterCancellationBeforeWrite(t *testing.T) {
	t.Parallel()
	installer := newRuleSetInstallTestInstaller(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	transport := installer.client.Transport
	installer.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cancel()
		return transport.RoundTrip(req)
	})
	updater, path := newCIDRTestUpdater(t, installer)
	before := []byte("keep cached bytes")
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	response, err := updater.Update(ctx, RuleSetUpdateRequest{})
	if err != nil || response.UpdatedCount != 0 || response.FailedCount != 1 {
		t.Fatalf("response = %+v, error = %v, want canceled item", response, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, data) {
		t.Fatalf("canceled updater changed the cache: %q, error = %v", data, err)
	}
}

func TestRuleSetBundledRejectsSourceFormatMismatch(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"binary", "cidr"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
			source := installer.sources[0]
			source.Format = format
			installer.sources = []RuleSetSource{source}
			if entries, err := installer.InstallBundled(t.Context()); err == nil || entries != nil {
				t.Fatalf("entries = %v, error = %v, want incompatible snapshot failure", entries, err)
			}
		})
	}
}

func newRuleSetInstallTestInstaller(t *testing.T) *LoyalsoldierRuleSetInstaller {
	t.Helper()
	return &LoyalsoldierRuleSetInstaller{
		ruleSetDir: filepath.Join(t.TempDir(), "rule-sets"),
		sources: []RuleSetSource{{
			Tag: "geoip-cn", FileName: "geoip-cn.srs", URL: "https://ruleset.test/cn.srs", Format: "binary",
		}},
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("srs")), Request: req}, nil
		})},
	}
}
