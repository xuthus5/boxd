package api

import (
	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type nodeSyncReloadTracker struct {
	restartableInstance
	restarted bool
}

func (h *NodesHandler) syncWithResult() (bool, error) {
	if h.instance == nil {
		return false, h.sync()
	}
	tracker := &nodeSyncReloadTracker{restartableInstance: h.instance}
	err := syncOutboundsAndRestart(h.nodeManager, h.subManager, h.configPath, tracker)
	return tracker.restarted, err
}

func (t *nodeSyncReloadTracker) Reload() error {
	provider, reportsStatus := t.restartableInstance.(interface{ Status() model.ServiceStatus })
	var before model.ServiceStatus
	if reportsStatus {
		before = provider.Status()
	}
	if err := core.ReloadConfig(t.restartableInstance); err != nil {
		return err
	}
	t.restarted = true
	if reportsStatus {
		t.restarted = kernelReloaded(before, provider.Status())
	}
	return nil
}

func kernelReloaded(before, after model.ServiceStatus) bool {
	if !before.Running || !after.Running {
		return false
	}
	if before.StartedAt == nil || after.StartedAt == nil {
		return true
	}
	return !before.StartedAt.Equal(*after.StartedAt)
}
