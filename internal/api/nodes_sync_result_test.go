package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

func TestNodeSyncWithoutKernelReportsNoRestart(t *testing.T) {
	nodes, subs, _, configPath := newAPIManagers(t)
	handler := NewNodesHandler(nodes, subs, configPath)
	recorder := httptest.NewRecorder()
	handler.SyncToConfig(recorder, httptest.NewRequest(http.MethodPost, "/api/nodes/sync-config", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"restarted":false`) {
		t.Fatalf("sync without a kernel must report no restart, got %s", recorder.Body.String())
	}
}

func TestKernelReloaded(t *testing.T) {
	startedAt := time.Now().UTC()
	restartedAt := startedAt.Add(time.Second)
	running := model.ServiceStatus{Running: true, StartedAt: &startedAt}
	restarted := model.ServiceStatus{Running: true, StartedAt: &restartedAt}
	for _, test := range []struct {
		name          string
		before, after model.ServiceStatus
		want          bool
	}{
		{name: "stopped"},
		{name: "stopped before reload", after: restarted},
		{name: "stopped after reload", before: running},
		{name: "same instance", before: running, after: running},
		{name: "new instance", before: running, after: restarted, want: true},
		{name: "legacy status", before: model.ServiceStatus{Running: true}, after: restarted, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := kernelReloaded(test.before, test.after); got != test.want {
				t.Fatalf("want reloaded=%v, got %v", test.want, got)
			}
		})
	}
}
