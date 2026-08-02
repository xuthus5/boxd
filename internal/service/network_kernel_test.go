package service

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestNetworkListInterfaces(t *testing.T) {
	oldList := listInterfaces
	oldAddrs := interfaceAddrs
	t.Cleanup(func() {
		listInterfaces = oldList
		interfaceAddrs = oldAddrs
	})

	listInterfaces = func() ([]net.Interface, error) {
		return []net.Interface{
			{Name: "eth0"},
			{Name: "lo"},
		}, nil
	}
	interfaceAddrs = func(iface net.Interface) ([]net.Addr, error) {
		return []net.Addr{
			&net.IPNet{IP: net.ParseIP("192.168.1.10"), Mask: net.CIDRMask(24, 32)},
			&net.IPAddr{IP: net.ParseIP("127.0.0.1")},
		}, nil
	}

	svc := &ServiceSet{network: &Network{}}
	result, err := svc.Network().ListInterfaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "eth0" {
		t.Fatalf("result = %+v", result)
	}
	if len(result[0].IPs) != 1 || result[0].IPs[0] != "192.168.1.10" {
		t.Fatalf("ips = %v", result[0].IPs)
	}
}

func TestNetworkListInterfacesError(t *testing.T) {
	oldList := listInterfaces
	t.Cleanup(func() { listInterfaces = oldList })
	listInterfaces = func() ([]net.Interface, error) {
		return nil, errors.New("boom")
	}
	svc := &ServiceSet{network: &Network{}}
	_, err := svc.Network().ListInterfaces(context.Background())
	var de *DomainError
	if !errors.As(err, &de) {
		t.Fatalf("expected DomainError, got %v", err)
	}
	if de.Code != "internal_error" {
		t.Fatalf("code = %q", de.Code)
	}
}

func TestKernelVersionMemoryGC(t *testing.T) {
	svc := &ServiceSet{kernel: &Kernel{version: "test-version"}}
	ver := svc.Kernel().KernelVersion(context.Background())
	if ver["version"] != "test-version" {
		t.Fatalf("version = %v", ver)
	}
	if ver["kernel_version"] == "" {
		t.Fatalf("kernel_version empty: %v", ver)
	}
	mem := svc.Kernel().KernelMemory(context.Background())
	for _, key := range []string{"alloc", "total", "sys", "num_gc", "heap_inuse", "stack_inuse", "num_goroutine"} {
		if _, ok := mem[key]; !ok {
			t.Fatalf("missing %q in %v", key, mem)
		}
	}
	if err := svc.Kernel().KernelGC(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type fakeServiceInstance struct {
	status model.ServiceStatus
	start  error
	stop   error
}

func (f *fakeServiceInstance) Start() error { return f.start }
func (f *fakeServiceInstance) Stop() error  { return f.stop }
func (f *fakeServiceInstance) Restart() error {
	if f.stop != nil {
		return f.stop
	}
	return f.start
}
func (f *fakeServiceInstance) Status() model.ServiceStatus { return f.status }

func TestServiceControlStatus(t *testing.T) {
	svc := &ServiceSet{service: &ServiceControl{instance: &fakeServiceInstance{status: model.ServiceStatus{Running: true}}}}
	st, err := svc.Service().ServiceStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running {
		t.Fatalf("running = %v", st.Running)
	}
}

func TestServiceControlNilInstance(t *testing.T) {
	svc := &ServiceSet{service: &ServiceControl{}}
	st, err := svc.Service().ServiceStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Running {
		t.Fatalf("running = %v", st.Running)
	}
	if err := svc.Service().ServiceStart(context.Background()); err == nil {
		t.Fatal("expected error with nil instance")
	}
	if err := svc.Service().ServiceStop(context.Background()); err == nil {
		t.Fatal("expected error with nil instance")
	}
	if err := svc.Service().ServiceRestart(context.Background()); err == nil {
		t.Fatal("expected error with nil instance")
	}
}

func TestServiceControlStartStopRestart(t *testing.T) {
	instance := &fakeServiceInstance{}
	svc := &ServiceSet{service: &ServiceControl{instance: instance}}
	if err := svc.Service().ServiceStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := svc.Service().ServiceStop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := svc.Service().ServiceRestart(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestServiceControlErrors(t *testing.T) {
	instance := &fakeServiceInstance{start: errors.New("start failed"), stop: errors.New("stop failed")}
	svc := &ServiceSet{service: &ServiceControl{instance: instance}}
	if err := svc.Service().ServiceStart(context.Background()); err == nil {
		t.Fatal("expected start error")
	}
	if err := svc.Service().ServiceStop(context.Background()); err == nil {
		t.Fatal("expected stop error")
	}
}

func TestSyncOutboundsToConfigBasics(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	if err := nodeMgr.Add(model.Outbound{Tag: "node-a", Type: "vless", Server: "1.1.1.1", Port: 443}); err != nil {
		t.Fatal(err)
	}
	if err := SyncOutboundsToConfig(nodeMgr, subMgr, configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(string(data), "node-a") {
		t.Fatalf("config missing node: %s", data)
	}
}

func TestSyncOutboundsAndRestart(t *testing.T) {
	db := newTestDB(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	nodeMgr := core.NewNodeManager(db)
	subMgr := core.NewSubscriptionManager(db, t.TempDir())
	restarts := 0
	instance := restartFuncAdapter(func() error {
		restarts++
		return nil
	})
	if err := SyncOutboundsAndRestart(nodeMgr, subMgr, configPath, instance); err != nil {
		t.Fatal(err)
	}
	if restarts != 1 {
		t.Fatalf("restarts = %d, want 1 (config changed)", restarts)
	}
}

type restartFuncAdapter func() error

func (f restartFuncAdapter) Restart() error { return f() }
