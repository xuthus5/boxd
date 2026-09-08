package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	box "github.com/sagernet/sing-box"
)

type sbReloadFixture struct {
	instance   *SBInstance
	boxes      []*fakeBox
	options    []box.Options
	factoryErr error
	startErr   error
}

func newSBReloadFixture(t *testing.T) *sbReloadFixture {
	t.Helper()
	fixture := &sbReloadFixture{
		instance: NewSBInstance(filepath.Join(t.TempDir(), "config.json"), NewLogWriter(5)),
		boxes:    []*fakeBox{},
		options:  []box.Options{},
	}
	writeSBReloadConfig(t, fixture.instance, `{}`)
	withNewBox(t, func(options box.Options) (boxInstance, error) {
		fixture.options = append(fixture.options, options)
		if fixture.factoryErr != nil {
			return nil, fixture.factoryErr
		}
		instance := newFakeBox()
		instance.startErr = fixture.startErr
		fixture.boxes = append(fixture.boxes, instance)
		return instance, nil
	})
	t.Cleanup(func() { _ = fixture.instance.Stop() })
	return fixture
}

func writeSBReloadConfig(t *testing.T, instance *SBInstance, body string) {
	t.Helper()
	if err := os.WriteFile(instance.configPath, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func (f *sbReloadFixture) start(t *testing.T) {
	t.Helper()
	if err := f.instance.Start(); err != nil {
		t.Fatal(err)
	}
}

func TestSBInstanceReloadStopped(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "valid config", body: `{}`},
		{name: "invalid config", body: `{`},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newSBReloadFixture(t)
			writeSBReloadConfig(t, fixture.instance, test.body)
			if err := ReloadConfig(fixture.instance); err != nil {
				t.Fatal(err)
			}
			if fixture.instance.Status().Running || len(fixture.boxes) != 0 {
				t.Fatal("reload of a stopped kernel must not construct or start a box")
			}
		})
	}
}

func TestSBInstanceReloadStoppedPreservesStartError(t *testing.T) {
	instance := NewSBInstance(filepath.Join(t.TempDir(), "missing.json"), NewLogWriter(5))
	if err := instance.Start(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want missing config error, got %v", err)
	}
	before := instance.Status()
	if err := instance.Reload(); err != nil {
		t.Fatalf("reload must not read config for a stopped kernel: %v", err)
	}
	if after := instance.Status(); !reflect.DeepEqual(before, after) {
		t.Fatalf("stopped kernel diagnostics changed: before=%#v after=%#v", before, after)
	}
}

func TestSBInstanceReloadRunningLoadsCurrentConfig(t *testing.T) {
	fixture := newSBReloadFixture(t)
	fixture.start(t)
	previousTracker := fixture.instance.TrafficTracker()
	writeSBReloadConfig(t, fixture.instance, `{"log":{"level":"debug"}}`)
	if err := fixture.instance.Reload(); err != nil {
		t.Fatal(err)
	}
	if !fixture.instance.Status().Running || len(fixture.boxes) != 2 {
		t.Fatalf("expected a running replacement kernel, got %d boxes", len(fixture.boxes))
	}
	if !fixture.boxes[0].closed || fixture.boxes[1].closed {
		t.Fatal("reload must close the previous box and keep its replacement open")
	}
	if !errors.Is(fixture.options[0].Context.Err(), context.Canceled) {
		t.Fatal("reload must cancel the previous kernel context")
	}
	if err := fixture.options[1].Context.Err(); err != nil {
		t.Fatalf("replacement context is canceled: %v", err)
	}
	if fixture.options[1].Log.Level != "debug" {
		t.Fatal("reload did not load the current configuration")
	}
	if tracker := fixture.instance.TrafficTracker(); tracker == nil || tracker == previousTracker {
		t.Fatal("reload must replace the traffic tracker")
	}
}

func TestSBInstanceRestartStartsStoppedKernel(t *testing.T) {
	fixture := newSBReloadFixture(t)
	if err := fixture.instance.Restart(); err != nil {
		t.Fatal(err)
	}
	if !fixture.instance.Status().Running || len(fixture.boxes) != 1 {
		t.Fatal("explicit restart must start a stopped kernel")
	}
}

type configRestartFunc func() error

func (f configRestartFunc) Restart() error { return f() }

type configReloadProbe struct {
	configRestartFunc
	calls int
	err   error
}

func (p *configReloadProbe) Reload() error {
	p.calls++
	return p.err
}

func TestReloadConfigCompatibility(t *testing.T) {
	if err := ReloadConfig(nil); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("restart failed")
	calls := 0
	instance := configRestartFunc(func() error {
		calls++
		return wantErr
	})
	if err := ReloadConfig(instance); !errors.Is(err, wantErr) {
		t.Fatalf("want restart error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("want one legacy restart, got %d", calls)
	}
}

func TestReloadConfigPrefersReload(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "failure", err: errors.New("reload failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			probe := &configReloadProbe{
				err: test.err,
				configRestartFunc: func() error {
					t.Error("reload must not fall back to an explicit restart")
					return nil
				},
			}
			if err := ReloadConfig(probe); !errors.Is(err, test.err) {
				t.Fatalf("want reload error %v, got %v", test.err, err)
			}
			if probe.calls != 1 {
				t.Fatalf("want one reload, got %d", probe.calls)
			}
		})
	}
}
