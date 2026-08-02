package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestApplyInstalledConfigInvalidRuntime(t *testing.T) {
	dir := t.TempDir()
	cfg := newConfig(dir, nil, ConfigInstaller{})
	_, err := cfg.applyInstalledConfig(context.Background(), map[string]any{
		"outbounds": []any{
			map[string]any{"type": "selector", "tag": "a", "outbounds": []any{"b"}},
			map[string]any{"type": "urltest", "tag": "b", "outbounds": []any{"a"}},
		},
	}, "test", []map[string]any{{}})
	if err == nil {
		t.Fatal("expected invalid runtime error")
	}
	var invalid *ErrInvalidRuntime
	if !errors.As(err, &invalid) {
		t.Fatalf("type = %T", err)
	}
}

func TestApplyInstalledConfigEncodeError(t *testing.T) {
	cfg := newConfig(t.TempDir(), nil, ConfigInstaller{})
	_, err := cfg.applyInstalledConfig(context.Background(), map[string]any{
		"bad": func() {},
	}, "test", nil)
	if err == nil {
		t.Fatal("expected encode error")
	}
}

func TestApplyInstalledConfigWriteError(t *testing.T) {
	cfg := newConfig(t.TempDir(), nil, ConfigInstaller{})
	_, err := cfg.applyInstalledConfig(context.Background(), map[string]any{
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
	}, "test", nil)
	if err == nil {
		t.Fatal("expected write error for missing parent dir")
	}
}

func TestInstallDefaultRouteRulesWithMetadata(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	configPath := dir + "/config.json"
	if err := writeTestJSONFile(configPath, map[string]any{"outbounds": []any{}}); err != nil {
		t.Fatal(err)
	}
	cfg := newConfig(configPath, nil, ConfigInstaller{
		RouteInstaller: core.NewDefaultRouteInstaller(),
		ApplyHistory:   core.NewConfigApplyHistoryManager(db),
		RouteMetadata:  core.NewRouteRuleMetadataManager(db),
	})
	result, err := cfg.InstallDefaultRouteRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestRestartFailureMessage(t *testing.T) {
	if got := restartFailureMessage(errors.New("boom")); got != "restart failed after config save: boom" {
		t.Fatalf("got %q", got)
	}
	if got := restartFailureMessage(nil); got != "restart failed after config save" {
		t.Fatalf("got %q", got)
	}
	if got := restartFailureMessage(errors.New("  ")); got != "restart failed after config save" {
		t.Fatalf("got %q", got)
	}
}

func TestAtomicWriteFileErrorPaths(t *testing.T) {
	dir := t.TempDir()
	if err := atomicWriteFile(dir, []byte("x")); err == nil {
		t.Fatal("expected error writing to directory path")
	}
	if err := atomicWriteFile(dir+"/nonexistent/config.json", []byte("x")); err != nil {
		t.Fatal(err)
	}
}

func TestProbeDNSServerEmptyResponse(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	dnsUDPExchange = func(_ context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		return nil, nil
	}
	result := probeDNSServer(context.Background(), DNSProbeRequest{Type: "udp", Server: "1.1.1.1"})
	if result.Success {
		t.Fatal("expected failure for nil response")
	}
	if result.ErrorCode != ProbeErrorEmpty {
		t.Fatalf("code = %q", result.ErrorCode)
	}
}

func TestProbeDNSServerBadRcode(t *testing.T) {
	original := dnsUDPExchange
	t.Cleanup(func() { dnsUDPExchange = original })
	dnsUDPExchange = func(_ context.Context, _ *dns.Msg, _ string, _ time.Duration) (*dns.Msg, error) {
		response := new(dns.Msg)
		response.Rcode = dns.RcodeServerFailure
		return response, nil
	}
	result := probeDNSServer(context.Background(), DNSProbeRequest{Type: "udp", Server: "1.1.1.1"})
	if result.Success {
		t.Fatal("expected failure for servfail")
	}
	if result.ErrorCode != ProbeErrorDNSRcode {
		t.Fatalf("code = %q", result.ErrorCode)
	}
}

func TestExchangeDNSHTTP3UnavailableConfig(t *testing.T) {
	message := new(dns.Msg)
	message.SetQuestion("example.com.", dns.TypeA)
	original := newDNSProbeTLSConfig
	t.Cleanup(func() { newDNSProbeTLSConfig = original })
	newDNSProbeTLSConfig = func(string) *tls.Config { return nil }
	if _, err := exchangeDNSHTTP3(context.Background(), message, "dns.example", 443, "/dns-query", time.Second); err == nil {
		t.Fatal("expected error when tls config unavailable")
	}
}

func TestValidateDNSProbeDomain(t *testing.T) {
	if err := validateDNSProbeDomain(""); err != nil {
		t.Fatal(err)
	}
	if err := validateDNSProbeDomain("example.com"); err != nil {
		t.Fatal(err)
	}
	if err := validateDNSProbeDomain(strings.Repeat("a", 300)); err == nil {
		t.Fatal("expected too long error")
	}
	if err := validateDNSProbeDomain("bad/domain"); err == nil {
		t.Fatal("expected slash error")
	}
	if err := validateDNSProbeDomain("bad domain"); err == nil {
		t.Fatal("expected space error")
	}
	if err := validateDNSProbeDomain("bad\x00domain"); err == nil {
		t.Fatal("expected control error")
	}
}

func TestExistingOutbounds(t *testing.T) {
	if got := existingOutbounds(map[string]any{}); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if got := existingOutbounds(map[string]any{"outbounds": []any{map[string]any{"tag": "a"}}}); len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestManagedURLTestTags(t *testing.T) {
	subs := []model.Subscription{{Name: "s1"}}
	tags := managedURLTestTags([]string{"old"}, subs)
	if !tags["old"] || !tags["s1"] {
		t.Fatalf("tags = %v", tags)
	}
}

func TestBuildManagedOutboundPreserveRaw(t *testing.T) {
	outbound := model.Outbound{
		Tag: "n", Type: "vless", Server: "1.1.1.1", Port: 443,
		Raw: map[string]any{"uuid": "xxx", "flow": "xtls-rprx-vision"},
	}
	entry, err := buildManagedOutbound(nil, outbound)
	if err != nil {
		t.Fatal(err)
	}
	if entry["uuid"] != "xxx" {
		t.Fatalf("uuid = %v", entry["uuid"])
	}
	if entry["routing_mark"] != 128 {
		t.Fatalf("routing_mark = %v", entry["routing_mark"])
	}
	existing := map[string]any{"tag": "n", "type": "vless"}
	entry2, err := buildManagedOutbound(existing, outbound)
	if err != nil {
		t.Fatal(err)
	}
	if entry2["tag"] != "n" {
		t.Fatalf("tag = %v", entry2["tag"])
	}
}

func TestFailedDNSProbeResult(t *testing.T) {
	base := DNSProbeResult{Tag: "t"}
	result := failedDNSProbeResult(base, "", errors.New("boom"))
	if result.Success {
		t.Fatal("expected failure")
	}
	if result.Error != "boom" {
		t.Fatalf("error = %q", result.Error)
	}
	result = failedDNSProbeResult(base, "", nil)
	if result.Error != "probe failed" {
		t.Fatalf("error = %q", result.Error)
	}
}

func TestIsProbeNetworkError(t *testing.T) {
	if isProbeNetworkError(errors.New("plain")) {
		t.Fatal("unexpected")
	}
	if !isProbeNetworkError(&net.OpError{Op: "dial"}) {
		t.Fatal("expected OpError to be network error")
	}
	if !isProbeNetworkError(&net.DNSError{Err: "no such host"}) {
		t.Fatal("expected DNSError to be network error")
	}
	if !isProbeNetworkError(syscall.ECONNREFUSED) {
		t.Fatal("expected ECONNREFUSED to be network error")
	}
	if !isProbeNetworkError(syscall.EHOSTUNREACH) {
		t.Fatal("expected EHOSTUNREACH to be network error")
	}
	if !isProbeNetworkError(syscall.ENETUNREACH) {
		t.Fatal("expected ENETUNREACH to be network error")
	}
}

func TestRestoreWithGroupFailure(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	previous := []byte(`{"outbounds":[]}`)
	if err := os.WriteFile(configPath, previous, 0600); err != nil {
		t.Fatal(err)
	}
	snapshot := outboundSyncSnapshot{
		configPath: configPath,
		config:     previous,
		groups:     []string{"g"},
		settings:   core.NewSettingsManager(db),
	}
	_ = db.Close()
	err := snapshot.restore()
	if err == nil {
		t.Fatal("expected group restore failure")
	}
	if !strings.Contains(err.Error(), "restoring previous managed groups") {
		t.Fatalf("err = %v", err)
	}
}

func writeTestJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func TestReadDNSMessageBodyNil(t *testing.T) {
	if _, err := readDNSMessageBody(nil); err == nil {
		t.Fatal("expected nil reader error")
	}
}
