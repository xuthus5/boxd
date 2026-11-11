package main

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/core"
)

func TestAppModelApplySnapshot(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	m := newAppModel()
	m.applySnapshot(seededSnapshot(themeLight), now)
	if !m.loaded || m.stale || m.lastUpdated != now {
		t.Fatalf("model state = loaded %v stale %v updated %v", m.loaded, m.stale, m.lastUpdated)
	}
	if !m.hasStatus || !m.status.Running || len(m.nodes) != 240 || len(m.subs) != 3 || len(m.connections) != 3 {
		t.Fatalf("unexpected snapshot data: status=%v nodes=%d subs=%d conns=%d", m.status, len(m.nodes), len(m.subs), len(m.connections))
	}
	if m.testURL != seededSnapshot(themeLight).TestURL || m.testURLDraft != m.testURL {
		t.Fatalf("test URL state = %q / %q", m.testURL, m.testURLDraft)
	}
}

func TestAppModelTrafficRatesAndWindow(t *testing.T) {
	m := newAppModel()
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	m.applyTraffic(trafficTotals{Up: 1000, Down: 2000}, now)
	if len(m.traffic) != 1 || m.traffic[0].Up != 0 || m.traffic[0].Down != 0 {
		t.Fatalf("first sample = %+v", m.traffic)
	}
	m.applyTraffic(trafficTotals{Up: 3000, Down: 6000}, now.Add(2*time.Second))
	if got := m.traffic[1]; got.Up != 1000 || got.Down != 2000 {
		t.Fatalf("second sample = %+v", got)
	}
	for i := 0; i < trafficWindow+10; i++ {
		m.applyTraffic(trafficTotals{Up: int64(i + 10000), Down: int64(i + 20000)}, now.Add(time.Duration(i+3)*time.Second))
	}
	if len(m.traffic) != trafficWindow {
		t.Fatalf("traffic window = %d, want %d", len(m.traffic), trafficWindow)
	}
}

func TestAppModelStaleOnRefreshError(t *testing.T) {
	app := newApplication(&fakeBackend{}, testConfig(), nil, false)
	app.model.loaded = true
	app.model.status.Running = true
	app.applyRefresh(snapshot{}, errors.New("temporary failure"), time.Now())
	if !app.model.stale || !app.model.status.Running {
		t.Fatalf("stale=%v status=%+v", app.model.stale, app.model.status)
	}
	if len(app.model.warnings) != 1 {
		t.Fatalf("warnings = %#v", app.model.warnings)
	}
}

func TestAppModelLogsAndFilters(t *testing.T) {
	m := newAppModel()
	logs := make([]core.LogEntry, 0, maxDisplayLogs+20)
	for i := 0; i < maxDisplayLogs+20; i++ {
		level := []string{"debug", "info", "warn", "error"}[i%4]
		logs = append(logs, core.LogEntry{Level: level, Message: "entry"})
	}
	m.applySnapshot(snapshot{LogsLoaded: true, Logs: logs}, time.Now())
	if len(m.logs) != maxDisplayLogs {
		t.Fatalf("logs kept = %d, want %d", len(m.logs), maxDisplayLogs)
	}
	for filter, want := range map[int]string{1: "info", 2: "warn", 3: "error"} {
		m.logFilter = filter
		for _, entry := range m.visibleLogs() {
			if entry.Level != want {
				t.Fatalf("filter %d returned %q", filter, entry.Level)
			}
		}
	}
	m.logFilter = 0
	if got := len(m.visibleLogs()); got != maxDisplayLogs {
		t.Fatalf("unfiltered logs = %d", got)
	}
}

func TestTrafficSeriesAndPrefs(t *testing.T) {
	m := newAppModel()
	m.applyTraffic(trafficTotals{Up: 1, Down: 2}, time.Now())
	m.applyTraffic(trafficTotals{Up: 3, Down: 4}, time.Now().Add(time.Second))
	up, down := m.trafficSeries()
	if len(up) != 2 || len(down) != 2 || math.Abs(float64(up[1])-2) > 0.001 || math.Abs(float64(down[1])-2) > 0.001 {
		t.Fatalf("series up=%v down=%v", up, down)
	}
	if got := prefsWithTheme(m.prefs, themeDark); got.Theme != themeDark {
		t.Fatalf("prefs theme = %q", got.Theme)
	}
}
