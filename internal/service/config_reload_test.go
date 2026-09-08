package service

import (
	"testing"

	"github.com/xuthus5/boxd/internal/model"
)

type configReloadProbe struct {
	running               bool
	reloads, restarts     int
	reloadErr, restartErr error
}

func (p *configReloadProbe) Reload() error { p.reloads++; return p.reloadErr }

func (p *configReloadProbe) Restart() error { p.restarts++; return p.restartErr }

func (p *configReloadProbe) Status() model.ServiceStatus {
	return model.ServiceStatus{Running: p.running}
}

func TestConfigApplyPreservesStoppedKernel(t *testing.T) {
	svc := newTestService(t)
	writeTestConfig(t, svc.Deps.ConfigPath, map[string]any{})
	probe := &configReloadProbe{}
	cfg := newConfig(svc.Deps.ConfigPath, svc.Deps.DataDir, probe, ConfigInstaller{})
	if _, err := cfg.ApplyConfig(t.Context(), []byte(`{"log":{"level":"warn"}}`), "test"); err != nil {
		t.Fatal(err)
	}
	if probe.reloads != 1 || probe.restarts != 0 {
		t.Fatalf("config apply must preserve stopped state: reload=%d restart=%d", probe.reloads, probe.restarts)
	}
}
