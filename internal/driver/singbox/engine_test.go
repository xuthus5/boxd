package singbox

import (
	"context"
	"errors"
	"testing"

	"github.com/xuthus5/boxd/kernel"
)

func TestValidateConfig(t *testing.T) {
	engine := testEngine(t)
	ctx := context.Background()

	valid, err := engine.ValidateConfig(ctx, kernel.ConfigDocument{Format: "json", Body: []byte(`{}`)})
	if err != nil || !valid.Valid {
		t.Fatalf("ValidateConfig(valid) = %+v, %v", valid, err)
	}
	invalid, err := engine.ValidateConfig(ctx, kernel.ConfigDocument{Format: "json", Body: []byte(`{"unknown_field":true}`)})
	if err != nil {
		t.Fatalf("ValidateConfig(invalid) error = %v", err)
	}
	if invalid.Valid || len(invalid.Diagnostics) == 0 {
		t.Fatalf("ValidateConfig(invalid) = %+v", invalid)
	}
}

func TestEngineLifecycleWithEmptyConfiguration(t *testing.T) {
	engine := testEngine(t)
	ctx := context.Background()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	status := engine.Status()
	if status.State != kernel.StateRunning || status.Version == "" || status.StartedAt.IsZero() {
		t.Fatalf("running status = %+v", status)
	}
	if err := engine.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if engine.Status().State != kernel.StateStopped {
		t.Fatalf("state after Stop = %s", engine.Status().State)
	}
}

func TestEngineApplyRestartsRunningInstance(t *testing.T) {
	engine := testEngine(t)
	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	result, err := engine.ApplyConfig(ctx, kernel.ConfigDocument{Format: "json", Body: []byte(`{}`), Revision: "rev-2"}, kernel.ApplyOptions{})
	if err != nil {
		t.Fatalf("ApplyConfig() error = %v", err)
	}
	if result.Revision != "rev-2" || !result.Restarted {
		t.Fatalf("ApplyConfig() = %+v", result)
	}
	if engine.Status().Revision != "rev-2" || engine.Status().State != kernel.StateRunning {
		t.Fatalf("status after apply = %+v", engine.Status())
	}
}

func TestEngineRejectsInvalidApplyAndKeepsRevision(t *testing.T) {
	engine := testEngine(t)
	ctx := context.Background()

	_, err := engine.ApplyConfig(ctx, kernel.ConfigDocument{Format: "json", Body: []byte(`{"unknown_field":true}`), Revision: "rev-bad"}, kernel.ApplyOptions{})
	var kernelError *kernel.Error
	if !errors.As(err, &kernelError) || kernelError.Code != kernel.ErrCodeInvalidConfig {
		t.Fatalf("ApplyConfig() error = %v, want invalid config", err)
	}
	if engine.Status().Revision != "rev-1" {
		t.Fatalf("revision changed to %q", engine.Status().Revision)
	}
}

func testEngine(t *testing.T) *Engine {
	t.Helper()
	factory := NewFactory()
	engine, err := factory.Open(context.Background(), kernel.OpenOptions{
		Name:    "test",
		WorkDir: t.TempDir(),
		Config:  kernel.ConfigDocument{Format: "json", Body: []byte(`{}`), Revision: "rev-1"},
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	return engine.(*Engine)
}
