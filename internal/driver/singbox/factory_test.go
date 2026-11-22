package singbox

import (
	"context"
	"testing"

	"github.com/xuthus5/boxd/kernel"
)

func TestFactoryOpensStoppedEngine(t *testing.T) {
	factory := NewFactory()
	if factory.Kind() != kernel.EngineSingBox {
		t.Fatalf("Kind() = %s, want %s", factory.Kind(), kernel.EngineSingBox)
	}
	engine, err := factory.Open(context.Background(), kernel.OpenOptions{
		Name:    "test",
		WorkDir: t.TempDir(),
		Config:  kernel.ConfigDocument{Format: "json", Body: []byte(`{}`), Revision: "rev-1"},
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if engine.Status().State != kernel.StateStopped || engine.Status().Revision != "rev-1" {
		t.Fatalf("initial status = %+v", engine.Status())
	}
}
