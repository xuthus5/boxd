package bootstrap_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/bootstrap"
	"github.com/xuthus5/boxd/internal/kernel/fake"
	"github.com/xuthus5/boxd/kernel"
)

func TestApplicationAssemblyAndPersistence(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	app, err := bootstrap.New(ctx, bootstrap.Options{DataDir: dataDir, Factory: fake.NewFactory(), Logger: logger})
	if err != nil {
		t.Fatalf("bootstrap.New() error = %v", err)
	}
	if status, _ := app.Control.Kernel().Status(ctx); status.State != kernel.StateStopped {
		t.Fatalf("initial state = %s", status.State)
	}
	if inbounds, err := app.Control.Inbounds().List(ctx); err != nil || len(inbounds) != 2 {
		t.Fatalf("default inbounds = %+v, %v", inbounds, err)
	}
	if app.Control.Subscriptions() == nil || app.Control.Nodes() == nil || app.Control.Policies() == nil ||
		app.Control.RuleSets() == nil || app.Control.Security() == nil {
		t.Fatal("full application facade returned a nil module")
	}
	if ruleSets, err := app.Control.RuleSets().List(ctx); err != nil || len(ruleSets) != 3 {
		t.Fatalf("default rule sets = %+v, %v", ruleSets, err)
	}
	if current, err := app.Control.Config().Read(ctx); err != nil || current.Revision == "" {
		t.Fatalf("initial config = %+v, %v", current, err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "boxd.sqlite")); err != nil {
		t.Fatalf("database file: %v", err)
	}

	if _, err := app.Control.Settings().Set(ctx, "theme", "green"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	document := kernel.ConfigDocument{Format: "json", Body: []byte(`{"mixed":true}`), Revision: "rev-1"}
	if _, err := app.Control.Config().Apply(ctx, document, kernel.ApplyOptions{Source: "test"}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if err := app.Control.Kernel().Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if status, _ := app.Control.Kernel().Status(ctx); status.State != kernel.StateRunning || status.Revision != "rev-1" {
		t.Fatalf("running status = %+v", status)
	}
	if err := app.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}

	reopened, err := bootstrap.New(ctx, bootstrap.Options{DataDir: dataDir, Factory: fake.NewFactory(), Logger: logger})
	if err != nil {
		t.Fatalf("reopen bootstrap.New() error = %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	current, err := reopened.Control.Config().Read(ctx)
	if err != nil || current.Revision != "rev-1" || string(current.Body) != string(document.Body) {
		t.Fatalf("persisted config = %+v, %v", current, err)
	}
	setting, found, err := reopened.Control.Settings().Get(ctx, "theme")
	if err != nil || !found || setting.Value != "green" {
		t.Fatalf("persisted setting = %+v, found %v, err %v", setting, found, err)
	}
}
