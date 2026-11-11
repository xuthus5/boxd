package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

type fakeBackend struct {
	mu          sync.Mutex
	snapshot    snapshot
	snapshotErr error
	actionErr   error
	calls       []string
	lastPrefs   model.UIPreferences
}

func (f *fakeBackend) Snapshot(context.Context) (snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snapshot, f.snapshotErr
}

func (f *fakeBackend) StartKernel(context.Context) error {
	f.record("start")
	return f.actionErr
}

func (f *fakeBackend) StopKernel(context.Context) error {
	f.record("stop")
	return f.actionErr
}

func (f *fakeBackend) RestartKernel(context.Context) error {
	f.record("restart")
	return f.actionErr
}

func (f *fakeBackend) CloseConnection(context.Context, int64) error {
	f.record("close-connection")
	return f.actionErr
}

func (f *fakeBackend) RefreshSubscription(context.Context, string) error {
	f.record("refresh-subscription")
	return f.actionErr
}

func (f *fakeBackend) SetKernelAutostart(context.Context, bool) error {
	f.record("kernel-autostart")
	return f.actionErr
}

func (f *fakeBackend) SetUIPreferences(_ context.Context, prefs model.UIPreferences) error {
	f.record("preferences")
	f.mu.Lock()
	f.lastPrefs = prefs
	f.mu.Unlock()
	return f.actionErr
}

func (f *fakeBackend) SetTestURL(context.Context, string) error {
	f.record("test-url")
	return f.actionErr
}

func (f *fakeBackend) ImportLink(context.Context, string) error {
	f.record("import-link")
	return f.actionErr
}

func (f *fakeBackend) record(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name)
}

func (f *fakeBackend) callCount(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, call := range f.calls {
		if call == name {
			count++
		}
	}
	return count
}

func testConfig() runtime.Config {
	return runtime.Config{
		Mode: "embedded", DataDir: "/tmp/boxd-test", ConfigPath: "/tmp/boxd-test/config.json",
		RefreshInterval: 1,
	}
}

func seededModel(theme string) *appModel {
	m := newAppModel()
	now := time.Date(2026, 11, 11, 15, 4, 5, 0, time.Local)
	m.applySnapshot(seededSnapshot(theme), now)
	for i := 0; i < trafficWindow; i++ {
		m.applyTraffic(trafficTotals{
			Up:   int64(i) * 1400,
			Down: int64(i) * 5600,
		}, now.Add(time.Duration(i)*time.Second))
	}
	m.lastUpdated = now.Add(time.Duration(trafficWindow) * time.Second)
	return m
}

func seededSnapshot(theme string) snapshot {
	status := model.ServiceStatus{
		Running: true, Uptime: "2h 18m", Version: "1.14.4",
		ConfigPath: "/home/demo/.boxd/config.json",
	}
	return snapshot{
		Status: &status, Traffic: &trafficTotals{Up: 182_000_000, Down: 1_240_000_000},
		ConnectionsLoaded: true, Connections: seededConnections(),
		NodesLoaded: true, Nodes: seededNodes(240),
		SubscriptionsLoaded: true, Subscriptions: seededSubscriptions(),
		PreferencesLoaded: true, Preferences: model.UIPreferences{
			Theme: theme, Language: "zh", MinimumLogLevel: "all",
		},
		KernelAutostartLoaded: true, KernelAutostart: true,
		TestURLLoaded: true, TestURL: "https://www.gstatic.com/generate_204",
		LogsLoaded: true, Logs: seededLogs(),
	}
}

func seededNodes(count int) []nodeView {
	nodes := make([]nodeView, 0, count)
	types := []string{"hysteria2", "vless", "trojan", "shadowsocks"}
	for i := 0; i < count; i++ {
		nodes = append(nodes, nodeView{
			Tag: fmt.Sprintf("node-%03d", i), Type: types[i%len(types)],
			Server: fmt.Sprintf("edge-%02d.example.net", i%32), Port: 443,
			Test: map[bool]string{true: fmt.Sprintf("%d ms", 18+i%90), false: "Failed"}[i%9 != 0],
		})
	}
	return nodes
}

func seededConnections() []connectionView {
	return []connectionView{
		{ID: 41, Target: "api.github.com:443", Outbound: "proxy-a", Rule: "geosite-github", Upload: 184_320, Download: 2_941_210, Duration: "18s"},
		{ID: 42, Target: "cdn.example.net:443", Outbound: "proxy-b", Rule: "final", Upload: 91_120, Download: 8_421_770, Duration: "1m42s"},
		{ID: 43, Target: "1.1.1.1:53", Outbound: "direct", Rule: "dns", Upload: 2_048, Download: 4_096, Duration: "3s"},
	}
}

func seededSubscriptions() []model.Subscription {
	now := time.Date(2026, 11, 11, 14, 20, 0, 0, time.Local)
	expires := now.Add(20 * 24 * time.Hour)
	return []model.Subscription{
		{ID: "1", Name: "Primary airport", URL: "https://example.invalid/sub?token=demo", LastUpdated: now,
			Traffic: &model.SubscriptionTraffic{Upload: 120_000_000, Download: 780_000_000, Total: 2_000_000_000, Expire: &expires}},
		{ID: "2", Name: "Backup", URL: "https://backup.example.invalid/sub", LastUpdated: now.Add(-2 * time.Hour), Error: "temporary upstream failure"},
		{ID: "3", Name: "Manual nodes", URL: "https://manual.example.invalid/list", LastUpdated: now.Add(-48 * time.Hour)},
	}
}

func seededLogs() []core.LogEntry {
	now := time.Date(2026, 11, 11, 15, 3, 0, 0, time.Local)
	levels := []string{"info", "warn", "error", "debug"}
	messages := []string{
		"kernel started with generated configuration",
		"subscription refresh finished",
		"connection 41 closed by native client",
		"dns cache warmed",
		"rule set update skipped: already current",
	}
	logs := make([]core.LogEntry, 0, 28)
	for i := 0; i < 28; i++ {
		logs = append(logs, core.LogEntry{
			Level: levels[i%len(levels)], Message: messages[i%len(messages)],
			Timestamp: now.Add(time.Duration(i) * time.Second),
		})
	}
	return logs
}
