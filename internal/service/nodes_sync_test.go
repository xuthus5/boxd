package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestSyncOutboundsToConfigEmpty(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	if err := SyncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}
}

func TestSyncOutboundsToConfigMissingFile(t *testing.T) {
	db := newTestDB(t)
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	err := SyncOutboundsToConfig(nodeMgr, subMgr, filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestSyncOutboundsToConfigInvalidJSON(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	if err := SyncOutboundsToConfig(nodeMgr, subMgr, configPath); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSyncOutboundsToConfigWithSubscriptionsAndURLTest(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	settings := core.NewSettingsManager(db)
	if err := settings.SetURLTestDefaults(model.URLTestDefaults{
		Enabled: true, URL: "https://cp.cloudflare.com/", Interval: "3m", Tolerance: 50,
	}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	sub, err := subMgr.Create(core.SubscriptionParams{
		Name:    "sub-a",
		URL:     "https://example.com/sub",
		URLTest: &model.URLTestOverrides{Enabled: &enabled},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub.Outbounds = []model.Outbound{
		{Tag: "sub-node", Type: "vmess", Server: "2.2.2.2", Port: 443},
	}
	subData, err := json.Marshal(sub)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("subscriptions")).Put([]byte(sub.ID), subData)
	}); err != nil {
		t.Fatal(err)
	}
	_ = settings
	if err := SyncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(string(data), "sub-a") || !containsSubstring(string(data), "sub-node") {
		t.Fatalf("subscription group missing: %s", data)
	}
}

func TestSyncOutboundsAndRestartFailureRollback(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	previous := []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if err := os.WriteFile(configPath, previous, 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	if err := nodeMgr.Add(model.Outbound{Tag: "desired", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	instance := &fakeRestart{errs: []error{errors.New("restart failed"), nil}}
	err := SyncOutboundsAndRestart(nodeMgr, subMgr, configPath, instance)
	if err == nil {
		t.Fatal("expected restart failure")
	}
	after, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(previous) {
		t.Fatalf("config not rolled back\nwant: %s\ngot: %s", previous, after)
	}
}

func TestSyncOutboundsAndRestartCaptureFailure(t *testing.T) {
	db := newTestDB(t)
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	err := SyncOutboundsAndRestart(nodeMgr, subMgr, filepath.Join(t.TempDir(), "missing.json"), &fakeRestart{})
	if err == nil {
		t.Fatal("expected capture failure")
	}
}

func TestSyncOutboundsAndRestartSyncFailureRestore(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	previous := []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if err := os.WriteFile(configPath, previous, 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	_ = db.Close()
	err := SyncOutboundsAndRestart(nodeMgr, subMgr, configPath, &fakeRestart{})
	if err == nil {
		t.Fatal("expected sync failure")
	}
}

func TestOutboundConfigChanged(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot := outboundSyncSnapshot{config: []byte(`{"outbounds":[]}`)}
	changed, err := outboundConfigChanged(snapshot, configPath)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("should be unchanged")
	}
	if _, err := outboundConfigChanged(snapshot, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected read error")
	}
}

func TestRestartAdapterNil(t *testing.T) {
	var adapter restartAdapter
	if err := adapter.Restart(); err != nil {
		t.Fatal(err)
	}
}

func TestIsErrInvalidRuntime(t *testing.T) {
	if IsErrInvalidRuntime(errors.New("x")) {
		t.Fatal("unexpected")
	}
	if !IsErrInvalidRuntime(&ErrInvalidRuntime{}) {
		t.Fatal("expected true")
	}
}

func TestDomainError(t *testing.T) {
	de := Errorf(400, "invalid_request", "bad %s", "input")
	if de.Error() != "bad input" {
		t.Fatalf("error = %q", de.Error())
	}
	if de.Status != 400 || de.Code != "invalid_request" {
		t.Fatalf("de = %+v", de)
	}
}

func TestRuntimeConfigErrorMessage(t *testing.T) {
	msg := runtimeConfigErrorMessage(errors.New("invalid sing-box config: outbounds[0]: duplicate tag \"x\""))
	if !containsSubstring(msg, "duplicate tag") {
		t.Fatalf("msg = %q", msg)
	}
	msg = runtimeConfigErrorMessage(errors.New("invalid sing-box config: line one\nfield.more invalid"))
	if !containsSubstring(msg, "line one") {
		t.Fatalf("msg = %q", msg)
	}
	if got := fallbackRuntimeDiagnosticDetail(""); got != "invalid sing-box config" {
		t.Fatalf("got = %q", got)
	}
	if got := singleLineRuntimeConfigError([]string{"only one"}); got != "only one" {
		t.Fatalf("got = %q", got)
	}
}

func TestManagedNodeTypesAndProxy(t *testing.T) {
	if !isProxyLikeOutboundType("vless") {
		t.Fatal("vless should be proxy-like")
	}
	if isProxyLikeOutboundType("direct") {
		t.Fatal("direct should not be proxy-like")
	}
	if !managedNodeTypes["vmess"] {
		t.Fatal("vmess should be managed")
	}
	if _, ok := managedNodeTypes["ssh"]; ok {
		t.Fatal("ssh should not be managed")
	}
	if !isProxySelectorCandidate("vless") {
		t.Fatal("vless should be candidate")
	}
	if isProxySelectorCandidate("selector") {
		t.Fatal("selector should not be candidate")
	}
}

func TestAppendManagedOutboundsRawError(t *testing.T) {
	_, err := buildManagedOutbound(nil, model.Outbound{
		Tag: "bad", Type: "vless", Server: "1.1.1.1", Port: 443,
		Raw: map[string]any{"weird": func() {}},
	})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestSubscriptionGroupBuilderEmpty(t *testing.T) {
	builder := subscriptionGroupBuilder{}
	outbounds, groupTags := builder.append([]any{}, nil)
	if len(outbounds) != 0 || len(groupTags) != 0 {
		t.Fatalf("outbounds=%v groupTags=%v", outbounds, groupTags)
	}
}

func TestUpsertProxySelectorExisting(t *testing.T) {
	outbounds := []any{
		map[string]any{"type": "selector", "tag": "proxy", "outbounds": []any{}},
	}
	result := upsertProxySelector(outbounds, []string{"g"}, []string{"n"})
	entry, _ := result[0].(map[string]any)
	if entry["default"] != "g" {
		t.Fatalf("default = %v", entry["default"])
	}
}

func TestUpsertProxySelectorEmptyMembers(t *testing.T) {
	result := upsertProxySelector([]any{}, nil, nil)
	if len(result) != 0 {
		t.Fatalf("result = %v", result)
	}
}

func TestReadSyncConfigAndCommit(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	previous := []byte(`{"outbounds":[]}`)
	if err := os.WriteFile(configPath, previous, 0600); err != nil {
		t.Fatal(err)
	}
	config, raw, err := readSyncConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("raw empty")
	}
	_ = config

	settings := core.NewSettingsManager(db)
	commit := syncCommit{path: configPath, previous: previous, groups: settings}
	config["outbounds"] = []any{map[string]any{"type": "direct", "tag": "direct"}}
	if err := commit.write(config, []string{"sub-a"}); err != nil {
		t.Fatal(err)
	}
	groups, err := settings.URLTestManagedGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0] != "sub-a" {
		t.Fatalf("groups = %v", groups)
	}
}

func TestSyncCommitGroupWriteFailure(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	previous := []byte(`{"outbounds":[]}`)
	if err := os.WriteFile(configPath, previous, 0600); err != nil {
		t.Fatal(err)
	}
	settings := core.NewSettingsManager(db)
	_ = db.Close()
	commit := syncCommit{path: configPath, previous: previous, groups: settings}
	err := commit.write(map[string]any{"outbounds": []any{}}, nil)
	if err == nil {
		t.Fatal("expected group write failure")
	}
	after, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(previous) {
		t.Fatalf("config not restored: %s", after)
	}
}

func TestCaptureOutboundSyncSnapshot(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	snapshot, err := captureOutboundSyncSnapshot(subMgr, configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.restore(); err != nil {
		t.Fatal(err)
	}
	if _, err := captureOutboundSyncSnapshot(subMgr, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected capture error")
	}
}

func TestJSONEncodeRoundtrip(t *testing.T) {
	var v any
	if err := json.Unmarshal([]byte(`{"a":1}`), &v); err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(map[string]any); !ok {
		t.Fatalf("type = %T", v)
	}
}
