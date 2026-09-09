package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestSBInstanceNativeStartupAndShutdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"log":{"disabled":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	instance := NewSBInstance(path, nil)
	t.Cleanup(func() { _ = instance.Stop() })
	for range 20 {
		if err := instance.Start(); err != nil {
			t.Fatal(err)
		}
		if !instance.Status().Running || instance.TrafficTracker() == nil {
			t.Fatal("native runtime was not published after startup")
		}
		if err := instance.Stop(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRealBoxPreservesNativeNetworkMonitors(t *testing.T) {
	instance, err := newRealBox(box.Options{
		Context: include.Context(context.Background()),
		Options: option.Options{Log: &option.LogOptions{Disabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = instance.Close() })
	network := instance.box.Network()
	if network.NetworkMonitor() == nil || network.InterfaceMonitor() == nil {
		t.Fatal("resolved and bridge require the native network monitor")
	}
	if err := instance.Start(); err != nil {
		t.Fatal(err)
	}
}

func TestRealBoxRejectsMissingRegistries(t *testing.T) {
	if _, err := newRealBox(box.Options{}); err == nil {
		t.Fatal("expected missing registry error")
	}
}
