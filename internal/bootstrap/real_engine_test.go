package bootstrap_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strconv"
	"testing"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/internal/bootstrap"
	"github.com/xuthus5/boxd/internal/driver/singbox"
	"github.com/xuthus5/boxd/kernel"
)

func TestRealSingBoxStartsThroughControlPlane(t *testing.T) {
	ctx := context.Background()
	app, err := bootstrap.New(ctx, bootstrap.Options{
		DataDir: t.TempDir(),
		Factory: singbox.NewFactory(),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("bootstrap.New() error = %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })

	port := freeTCPPort(t)
	inbounds, err := app.Control.Inbounds().List(ctx)
	if err != nil {
		t.Fatalf("Inbounds().List() error = %v", err)
	}
	mixed := inbounds[0]
	mixed.Listen = control.ListenAddress{Host: "::", Port: port}
	if _, err := app.Control.Inbounds().Save(ctx, mixed); err != nil {
		t.Fatalf("Inbounds().Save() error = %v", err)
	}
	if _, err := app.Control.Routes().SetPresets(ctx, control.RoutePresets{BypassLAN: true}); err != nil {
		t.Fatalf("Routes().SetPresets() error = %v", err)
	}

	if err := app.Control.Kernel().Start(ctx); err != nil {
		t.Fatalf("Kernel().Start() error = %v", err)
	}
	status, err := app.Control.Kernel().Status(ctx)
	if err != nil || status.State != kernel.StateRunning {
		t.Fatalf("Status() = %+v, %v", status, err)
	}
	if err := app.Control.Kernel().Stop(ctx); err != nil {
		t.Fatalf("Kernel().Stop() error = %v", err)
	}
}

func freeTCPPort(t *testing.T) uint16 {
	t.Helper()
	listener, err := net.Listen("tcp", "[::]:0")
	if err != nil {
		t.Fatalf("reserve TCP port: %v", err)
	}
	defer listener.Close()
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}
	return uint16(port)
}
