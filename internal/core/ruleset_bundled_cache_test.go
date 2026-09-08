package core

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestRuleSetValidateCacheData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		body   string
		format string
		valid  bool
	}{
		{name: "valid source", body: `{"version":4,"rules":[{"domain_suffix":["example.com"]}]}`, valid: true},
		{name: "invalid json", body: "broken"},
		{name: "unknown field", body: `{"version":4,"rules":[{"domain_sufix":["example.com"]}]}`},
		{name: "missing rules", body: `{"version":4}`},
		{name: "empty match", body: `{"version":4,"rules":[{}]}`},
		{name: "invalid regex", body: `{"version":4,"rules":[{"domain_regex":["["]}]}`},
		{name: "invalid binary", body: "SRS", format: "binary"},
		{name: "invalid cidr cache", body: "not srs", format: "cidr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRuleSetData(t.Context(), []byte(tt.body), tt.format)
			if (err == nil) != tt.valid {
				t.Fatalf("validation error = %v, want valid = %v", err, tt.valid)
			}
		})
	}
	var empty bytes.Buffer
	if err := srs.Write(&empty, option.PlainRuleSet{}, C.RuleSetVersionCurrent); err != nil {
		t.Fatal(err)
	}
	if err := validateRuleSetData(t.Context(), empty.Bytes(), "binary"); err == nil {
		t.Fatal("empty binary cache must be rejected")
	}
}

func TestRuleSetCacheState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "valid", body: `{"version":4,"rules":[{"domain":["keep.example"]}]}`, want: true},
		{name: "corrupt", body: "broken"},
		{name: "oversized", body: strings.Repeat("x", maxRuleSetBodyBytes+1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "cache.json")
			if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
				t.Fatal(err)
			}
			valid, err := cachedRuleSetValid(t.Context(), path, "source")
			if err != nil || valid != tt.want {
				t.Fatalf("cache valid = %v, error = %v, want valid = %v", valid, err, tt.want)
			}
		})
	}
	valid, err := cachedRuleSetValid(t.Context(), filepath.Join(t.TempDir(), "missing"), "source")
	if err != nil || valid {
		t.Fatalf("missing cache valid = %v, error = %v", valid, err)
	}
}

func TestRuleSetCacheFilesystemErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if valid, err := cachedRuleSetValid(t.Context(), dir, "source"); err == nil || valid {
		t.Fatalf("directory cache valid = %v, error = %v", valid, err)
	}
	path := filepath.Join(dir, "target")
	content := []byte(`{"version":4,"rules":[{"domain":["keep.example"]}]}`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if valid, err := cachedRuleSetValid(t.Context(), link, "source"); err == nil || valid {
		t.Fatalf("symlink cache valid = %v, error = %v", valid, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(content, after) {
		t.Fatalf("symlink target changed: %q, error = %v", after, err)
	}
}

func TestRuleSetOnlineInstallCanceled(t *testing.T) {
	t.Parallel()
	installer := NewLoyalsoldierRuleSetInstaller(t.TempDir())
	installer.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("canceled install made an HTTP request")
		return nil, errors.New("unexpected HTTP request")
	})}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if entries, err := installer.Install(ctx); !errors.Is(err, context.Canceled) || entries != nil {
		t.Fatalf("entries = %v, error = %v, want canceled", entries, err)
	}
}
