package api

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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

func (p *nodeSyncReloadProbe) Status() model.ServiceStatus {
	return model.ServiceStatus{Running: p.running}
}

func TestNodeSyncUnchangedSkipsReload(t *testing.T) {
	nodes, subs, _, configPath := newAPIManagers(t)
	if err := syncOutboundsToConfig(nodes, subs, configPath); err != nil {
		t.Fatal(err)
	}
	probe := &nodeSyncReloadProbe{running: true}
	handler := NewNodesHandler(nodes, subs, configPath, probe)
	recorder := httptest.NewRecorder()
	handler.SyncToConfig(recorder, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"restarted":false`) {
		t.Fatalf("unchanged config must report no restart, got %s", recorder.Body.String())
	}
	if !probe.running || probe.reloadCalls != 0 || probe.restartCalls != 0 {
		t.Fatalf("unchanged config must leave the running kernel untouched: %#v", probe)
	}
}

func TestNodeSyncReloadFailureRestoresRunningKernel(t *testing.T) {
	nodes, subs, _, configPath := newAPIManagers(t)
	previous, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("reload failed")
	probe := &nodeSyncReloadProbe{running: true, reloadErr: wantErr}
	err = syncOutboundsAndRestart(nodes, subs, configPath, probe)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "previous configuration restored") {
		t.Fatalf("want recovered reload error, got %v", err)
	}
	if !probe.running || probe.reloadCalls != 1 || probe.restartCalls != 1 {
		t.Fatalf("rollback must explicitly restart the stopped kernel: %#v", probe)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(previous, after) {
		t.Fatalf("previous config was not restored: %s", after)
	}
}

func TestNodeSyncReloadFailureReportsRecoveryFailure(t *testing.T) {
	nodes, subs, _, configPath := newAPIManagers(t)
	reloadErr := errors.New("reload failed")
	restartErr := errors.New("recovery failed")
	probe := &nodeSyncReloadProbe{running: true, reloadErr: reloadErr, restartErr: restartErr}
	err := syncOutboundsAndRestart(nodes, subs, configPath, probe)
	if !errors.Is(err, reloadErr) || !errors.Is(err, restartErr) {
		t.Fatalf("want both reload and recovery errors, got %v", err)
	}
	if probe.running || probe.reloadCalls != 1 || probe.restartCalls != 1 {
		t.Fatalf("failed recovery state = %#v", probe)
	}
}

func TestNodeSyncPreservesKernelState(t *testing.T) {
	for _, running := range []bool{false, true} {
		name := "stopped"
		if running {
			name = "running"
		}
		t.Run(name, func(t *testing.T) {
			nodes, subs, _, configPath := newAPIManagers(t)
			if err := nodes.Add(model.Outbound{
				Tag: "desired-node", Type: "vless", Server: "example.test", Port: 443,
			}); err != nil {
				t.Fatal(err)
			}
			probe := &nodeSyncReloadProbe{running: running}
			handler := NewNodesHandler(nodes, subs, configPath, probe)
			recorder := httptest.NewRecorder()
			handler.SyncToConfig(recorder, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if probe.running != running || probe.reloadCalls != 1 || probe.restartCalls != 0 {
				t.Fatalf("want running=%v and one reload without explicit restart; got %#v", running, probe)
			}
			if !running && !strings.Contains(recorder.Body.String(), `"restarted":false`) {
				t.Fatalf("stopped kernel must report no restart, got %s", recorder.Body.String())
			}
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), `"desired-node"`) {
				t.Fatalf("synchronized node missing from config: %s", data)
			}
		})
	}
}

func TestNodeSyncWaitsForConfigSnapshotLock(t *testing.T) {
	const blockedWindow = 20 * time.Millisecond
	const completionTimeout = time.Second
	nodes, subs, _, configPath := newAPIManagers(t)
	unlock := sync.OnceFunc(core.LockConfig(configPath))
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
		result <- syncOutboundsAndRestart(nodes, subs, configPath, probe)
	})
	<-started
	select {
	case err := <-result:
		t.Fatalf("sync bypassed the held config lock: %v", err)
	case <-time.After(blockedWindow):
	}
	updated := []byte(`{"outbounds":[],"log":{"level":"debug"}}`)
	if err := os.WriteFile(configPath, updated, 0600); err != nil {
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
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updated, after) {
		t.Fatalf("rollback restored a stale config snapshot: %s", after)
	}
}
