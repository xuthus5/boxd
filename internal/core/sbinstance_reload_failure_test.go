package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSBInstanceReloadStopError(t *testing.T) {
	fixture := newSBReloadFixture(t)
	fixture.start(t)
	wantErr := errors.New("close failed")
	fixture.boxes[0].closeErr = wantErr
	if err := fixture.instance.Reload(); !errors.Is(err, wantErr) {
		t.Fatalf("want close error, got %v", err)
	}
	assertSBReloadStopped(t, fixture)
	if len(fixture.boxes) != 1 {
		t.Fatal("reload must not start a replacement after a close failure")
	}
	if err := fixture.instance.Restart(); err != nil {
		t.Fatalf("explicit restart must recover the stopped kernel: %v", err)
	}
	if !fixture.instance.Status().Running {
		t.Fatal("kernel did not recover after a failed reload")
	}
}

func TestSBInstanceRestartStopErrorDoesNotStartReplacement(t *testing.T) {
	fixture := newSBReloadFixture(t)
	fixture.start(t)
	wantErr := errors.New("close failed")
	fixture.boxes[0].closeErr = wantErr
	if err := fixture.instance.Restart(); !errors.Is(err, wantErr) {
		t.Fatalf("want close error, got %v", err)
	}
	assertSBReloadStopped(t, fixture)
	if len(fixture.boxes) != 1 {
		t.Fatal("restart must not ignore a close failure")
	}
}

func TestSBInstanceReloadFileErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		missing bool
	}{
		{name: "missing file", missing: true},
		{name: "invalid json"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newSBReloadFixture(t)
			fixture.start(t)
			path := fixture.instance.configPath
			writeSBReloadConfig(t, fixture.instance, `{`)
			if test.missing {
				fixture.instance.configPath = filepath.Join(t.TempDir(), "missing.json")
			}
			err := fixture.instance.Reload()
			if err == nil {
				t.Fatal("expected config load failure")
			}
			if test.missing && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("want missing file error, got %v", err)
			}
			assertSBReloadStopped(t, fixture)
			assertSBReloadDiagnostic(t, fixture.instance, err)
			fixture.instance.configPath = path
			writeSBReloadConfig(t, fixture.instance, `{}`)
			assertSBReloadRecovery(t, fixture)
		})
	}
}

func TestSBInstanceReloadConstructionErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		factory bool
	}{
		{name: "factory", factory: true},
		{name: "start"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newSBReloadFixture(t)
			fixture.start(t)
			wantErr := errors.New(test.name + " failed")
			if test.factory {
				fixture.factoryErr = wantErr
			} else {
				fixture.startErr = wantErr
			}
			if err := fixture.instance.Reload(); !errors.Is(err, wantErr) {
				t.Fatalf("want %v, got %v", wantErr, err)
			}
			assertSBReloadStopped(t, fixture)
			assertSBReloadDiagnostic(t, fixture.instance, wantErr)
			if !errors.Is(fixture.options[1].Context.Err(), context.Canceled) {
				t.Fatal("failed replacement must cancel its context")
			}
			if !test.factory && !fixture.boxes[1].closed {
				t.Fatal("failed replacement must be closed")
			}
			fixture.factoryErr, fixture.startErr = nil, nil
			assertSBReloadRecovery(t, fixture)
		})
	}
}

func assertSBReloadStopped(t *testing.T, fixture *sbReloadFixture) {
	t.Helper()
	if fixture.instance.Status().Running || fixture.instance.TrafficTracker() != nil {
		t.Fatal("failed reload must clear the running state and traffic tracker")
	}
	if !fixture.boxes[0].closed || !errors.Is(fixture.options[0].Context.Err(), context.Canceled) {
		t.Fatal("previous kernel resources were not released")
	}
}

func assertSBReloadDiagnostic(t *testing.T, instance *SBInstance, wantErr error) {
	t.Helper()
	status := instance.Status()
	if status.LastError != wantErr.Error() || status.LastErrorAt == nil {
		t.Fatalf("reload failure diagnostics = %#v", status)
	}
}

func assertSBReloadRecovery(t *testing.T, fixture *sbReloadFixture) {
	t.Helper()
	if err := fixture.instance.Restart(); err != nil {
		t.Fatalf("explicit restart after rollback failed: %v", err)
	}
	status := fixture.instance.Status()
	if !status.Running || status.LastError != "" || status.LastErrorAt != nil {
		t.Fatalf("kernel did not recover cleanly: %#v", status)
	}
}
