package service

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type nodeSyncReloadProbe struct {
	running      bool
	reloadCalls  int
	restartCalls int
	reloadErr    error
	restartErr   error
}

func (p *nodeSyncReloadProbe) Reload() error {
	p.reloadCalls++
	if !p.running {
		return nil
	}
	p.running = p.reloadErr == nil
	return p.reloadErr
}

func (p *nodeSyncReloadProbe) Restart() error {
	p.restartCalls++
	p.running = p.restartErr == nil
	return p.restartErr
}

type nodeSyncReloadFixture struct {
	nodes    *core.NodeManager
	subs     *core.SubscriptionManager
	settings *core.SettingsManager
	path     string
	previous []byte
}

func newNodeSyncReloadFixture(t *testing.T) nodeSyncReloadFixture {
	t.Helper()
	db := newTestDB(t)
	fixture := nodeSyncReloadFixture{
		nodes:    core.NewNodeManager(db),
		subs:     core.NewSubscriptionManager(db, t.TempDir()),
		settings: core.NewSettingsManager(db),
		path:     filepath.Join(t.TempDir(), "config.json"),
		previous: []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`),
	}
	if err := os.WriteFile(fixture.path, fixture.previous, 0600); err != nil {
		t.Fatal(err)
	}
	if err := fixture.nodes.Add(model.Outbound{
		Tag: "desired-node", Type: "vless", Server: "example.test", Port: 443,
	}); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestSyncOutboundsPreservesKernelState(t *testing.T) {
	for _, running := range []bool{false, true} {
		name := "stopped"
		if running {
			name = "running"
		}
		t.Run(name, func(t *testing.T) {
			fixture := newNodeSyncReloadFixture(t)
			probe := &nodeSyncReloadProbe{running: running}
			if err := SyncOutboundsAndRestart(fixture.nodes, fixture.subs, fixture.path, probe); err != nil {
				t.Fatal(err)
			}
			if probe.running != running || probe.reloadCalls != 1 || probe.restartCalls != 0 {
				t.Fatalf("want running=%v and one reload without explicit restart; got %#v", running, probe)
			}
			data, err := os.ReadFile(fixture.path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), `"desired-node"`) {
				t.Fatalf("synchronized node missing from config: %s", data)
			}
		})
	}
}

func TestSyncOutboundsUnchangedSkipsReload(t *testing.T) {
	fixture := newNodeSyncReloadFixture(t)
	if err := fixture.nodes.Delete("desired-node"); err != nil {
		t.Fatal(err)
	}
	if err := SyncOutboundsToConfig(fixture.nodes, fixture.subs, fixture.path); err != nil {
		t.Fatal(err)
	}
	probe := &nodeSyncReloadProbe{running: true}
	if err := SyncOutboundsAndRestart(fixture.nodes, fixture.subs, fixture.path, probe); err != nil {
		t.Fatal(err)
	}
	if !probe.running || probe.reloadCalls != 0 || probe.restartCalls != 0 {
		t.Fatalf("unchanged config must leave the running kernel untouched: %#v", probe)
	}
}

func TestSyncOutboundsWithoutKernelSavesConfig(t *testing.T) {
	fixture := newNodeSyncReloadFixture(t)
	if err := SyncOutboundsAndRestart(fixture.nodes, fixture.subs, fixture.path, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"desired-node"`) {
		t.Fatalf("synchronized node missing from config: %s", data)
	}
}

func TestSyncOutboundsSyncErrorPreservesKernelAndConfig(t *testing.T) {
	fixture := newNodeSyncReloadFixture(t)
	previous := []byte(`{`)
	if err := os.WriteFile(fixture.path, previous, 0600); err != nil {
		t.Fatal(err)
	}
	probe := &nodeSyncReloadProbe{running: true}
	if err := SyncOutboundsAndRestart(fixture.nodes, fixture.subs, fixture.path, probe); err == nil {
		t.Fatal("expected a synchronization error for invalid JSON")
	}
	if !probe.running || probe.reloadCalls != 0 || probe.restartCalls != 0 {
		t.Fatalf("failed synchronization must leave the running kernel untouched: %#v", probe)
	}
	after, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(previous, after) {
		t.Fatalf("failed synchronization changed config: %s", after)
	}
}

func TestSyncOutboundsReloadFailureRestoresRunningKernel(t *testing.T) {
	fixture := newNodeSyncReloadFixture(t)
	groups := []string{"previous-group"}
	if err := fixture.settings.SetURLTestManagedGroups(groups); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("reload failed")
	probe := &nodeSyncReloadProbe{running: true, reloadErr: wantErr}
	err := SyncOutboundsAndRestart(fixture.nodes, fixture.subs, fixture.path, probe)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "previous configuration restored") {
		t.Fatalf("want recovered reload error, got %v", err)
	}
	if !probe.running || probe.reloadCalls != 1 || probe.restartCalls != 1 {
		t.Fatalf("rollback must explicitly restart the stopped kernel: %#v", probe)
	}
	after, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixture.previous, after) {
		t.Fatalf("config was not restored: %s", after)
	}
	afterGroups, err := fixture.settings.URLTestManagedGroups()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(groups, afterGroups) {
		t.Fatalf("want managed groups %v, got %v", groups, afterGroups)
	}
	if fixture.nodes.Get("desired-node") == nil {
		t.Fatal("desired node must remain available for retry")
	}
}

func TestSyncOutboundsReloadFailureReportsRecoveryFailure(t *testing.T) {
	fixture := newNodeSyncReloadFixture(t)
	reloadErr := errors.New("reload failed")
	restartErr := errors.New("recovery failed")
	probe := &nodeSyncReloadProbe{running: true, reloadErr: reloadErr, restartErr: restartErr}
	err := SyncOutboundsAndRestart(fixture.nodes, fixture.subs, fixture.path, probe)
	if !errors.Is(err, reloadErr) || !errors.Is(err, restartErr) {
		t.Fatalf("want both reload and recovery errors, got %v", err)
	}
	if probe.running || probe.reloadCalls != 1 || probe.restartCalls != 1 {
		t.Fatalf("failed recovery state = %#v", probe)
	}
}

func TestOutboundReloadAttemptsRecoveryAfterRestoreFailure(t *testing.T) {
	fixture := newNodeSyncReloadFixture(t)
	snapshot, err := captureOutboundSyncSnapshot(fixture.subs, fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.configPath = filepath.Join(fixture.path, "config.json")
	if err := fixture.subs.DB().Close(); err != nil {
		t.Fatal(err)
	}
	reloadErr := errors.New("reload failed")
	probe := &nodeSyncReloadProbe{running: true, reloadErr: reloadErr}
	err = snapshot.reload(probe)
	if !errors.Is(err, reloadErr) || !strings.Contains(err.Error(), "restoring previous outbound configuration") {
		t.Fatalf("want reload and config restoration errors, got %v", err)
	}
	if !strings.Contains(err.Error(), "restoring previous managed groups") {
		t.Fatalf("want managed group restoration error, got %v", err)
	}
	if probe.restartCalls != 1 || !probe.running {
		t.Fatalf("restore error must not suppress explicit recovery: %#v", probe)
	}
}

func TestSyncOutboundsWaitsForConfigSnapshotLock(t *testing.T) {
	const blockedWindow = 20 * time.Millisecond
	const completionTimeout = time.Second
	fixture := newNodeSyncReloadFixture(t)
	unlock := sync.OnceFunc(core.LockConfig(fixture.path))
	var workers sync.WaitGroup
	t.Cleanup(func() {
		unlock()
		workers.Wait()
	})
	reloadErr := errors.New("reload failed")
	probe := &nodeSyncReloadProbe{running: true, reloadErr: reloadErr}
	started := make(chan struct{})
	result := make(chan error, 1)
	workers.Go(func() {
		close(started)
		result <- SyncOutboundsAndRestart(fixture.nodes, fixture.subs, fixture.path, probe)
	})
	<-started
	select {
	case err := <-result:
		t.Fatalf("sync bypassed the held config lock: %v", err)
	case <-time.After(blockedWindow):
	}
	updated := []byte(`{"outbounds":[],"log":{"level":"debug"}}`)
	if err := os.WriteFile(fixture.path, updated, 0600); err != nil {
		t.Fatal(err)
	}
	unlock()
	select {
	case err := <-result:
		if !errors.Is(err, reloadErr) {
			t.Fatalf("want reload failure after releasing the lock, got %v", err)
		}
	case <-time.After(completionTimeout):
		t.Fatal("sync did not finish after releasing the config lock")
	}
	after, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updated, after) {
		t.Fatalf("rollback restored a stale config snapshot: %s", after)
	}
}
