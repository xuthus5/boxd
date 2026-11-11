package main

import (
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

const (
	trafficWindow     = 120
	maxDisplayLogs    = 400
	defaultLogLevel   = 0
	themeSystem       = "system"
	themeLight        = "light"
	themeDark         = "dark"
	languageChinese   = "zh"
	languageEnglish   = "en"
	minimumLogLevel   = "all"
	dataStaleWarning  = "Data may be out of date. Retrying in the background."
	noDataPlaceholder = "Waiting for the first snapshot…"
)

type trafficTotals struct {
	Up   int64
	Down int64
}

type trafficSample struct {
	Up   float32
	Down float32
}

type nodeView struct {
	Tag        string
	Type       string
	Server     string
	Port       int
	Test       string
	Source     string
	SourceName string
	Config     any
}

type connectionView struct {
	ID       int64
	Target   string
	Outbound string
	Rule     string
	Upload   int64
	Download int64
	Duration string
}

type snapshot struct {
	Status        *model.ServiceStatus
	Traffic       *trafficTotals
	Connections   []connectionView
	Nodes         []nodeView
	Subscriptions []model.Subscription
	Preferences   model.UIPreferences
	TestURL       string
	Logs          []core.LogEntry

	ConnectionsLoaded     bool
	NodesLoaded           bool
	SubscriptionsLoaded   bool
	PreferencesLoaded     bool
	TestURLLoaded         bool
	LogsLoaded            bool
	KernelAutostartLoaded bool
	KernelAutostart       bool
	Warnings              []string
}

type appModel struct {
	status          model.ServiceStatus
	hasStatus       bool
	connections     []connectionView
	nodes           []nodeView
	subs            []model.Subscription
	prefs           model.UIPreferences
	testURL         string
	testURLDraft    string
	testURLDraftSet bool
	logs            []core.LogEntry
	logFilter       int

	kernelAutostart bool
	appAutostart    bool
	appAutostartSet bool

	traffic         []trafficSample
	totalUp         int64
	totalDown       int64
	lastTrafficUp   int64
	lastTrafficDown int64
	lastTrafficAt   time.Time

	loaded      bool
	stale       bool
	warnings    []string
	lastUpdated time.Time
	busy        bool
	busyLabel   string
	notice      string
	actionError string
	startupErr  error
}

func newAppModel() *appModel {
	return &appModel{prefs: model.UIPreferences{
		Theme: themeSystem, Language: languageChinese, MinimumLogLevel: minimumLogLevel,
	}, logFilter: defaultLogLevel}
}

func (m *appModel) applySnapshot(snap snapshot, now time.Time) {
	if snap.Status != nil {
		m.status = *snap.Status
		m.hasStatus = true
	}
	if snap.Traffic != nil {
		m.applyTraffic(*snap.Traffic, now)
	}
	if snap.ConnectionsLoaded {
		m.connections = snap.Connections
	}
	if snap.NodesLoaded {
		m.nodes = snap.Nodes
	}
	if snap.SubscriptionsLoaded {
		m.subs = snap.Subscriptions
	}
	if snap.PreferencesLoaded {
		m.prefs = snap.Preferences
	}
	if snap.KernelAutostartLoaded {
		m.kernelAutostart = snap.KernelAutostart
	}
	if snap.TestURLLoaded {
		m.testURL = snap.TestURL
		if !m.testURLDraftSet {
			m.testURLDraft = snap.TestURL
			m.testURLDraftSet = true
		}
	}
	if snap.LogsLoaded {
		m.logs = snap.Logs
		if len(m.logs) > maxDisplayLogs {
			m.logs = m.logs[len(m.logs)-maxDisplayLogs:]
		}
	}
	m.loaded = true
	m.lastUpdated = now
	m.warnings = snap.Warnings
	m.stale = len(snap.Warnings) > 0
}

func (m *appModel) applyTraffic(total trafficTotals, now time.Time) {
	upRate, downRate := float32(0), float32(0)
	if !m.lastTrafficAt.IsZero() {
		seconds := now.Sub(m.lastTrafficAt).Seconds()
		if seconds > 0 {
			upRate = float32(max(0, total.Up-m.lastTrafficUp)) / float32(seconds)
			downRate = float32(max(0, total.Down-m.lastTrafficDown)) / float32(seconds)
		}
	}
	m.lastTrafficUp = total.Up
	m.lastTrafficDown = total.Down
	m.lastTrafficAt = now
	m.totalUp = total.Up
	m.totalDown = total.Down
	m.traffic = append(m.traffic, trafficSample{Up: upRate, Down: downRate})
	if len(m.traffic) > trafficWindow {
		m.traffic = m.traffic[len(m.traffic)-trafficWindow:]
	}
}

func (m *appModel) setActionError(err error) {
	if err == nil {
		m.actionError = ""
		return
	}
	m.actionError = shortError(err)
}

func (m *appModel) visibleLogs() []core.LogEntry {
	if m.logFilter <= 0 || m.logFilter > 3 {
		return m.logs
	}
	want := []string{"info", "warn", "error"}[m.logFilter-1]
	out := make([]core.LogEntry, 0, len(m.logs))
	for _, entry := range m.logs {
		if strings.EqualFold(entry.Level, want) {
			out = append(out, entry)
		}
	}
	return out
}

func (m *appModel) trafficSeries() ([]float32, []float32) {
	xs := make([]float32, len(m.traffic))
	up := make([]float32, len(m.traffic))
	down := make([]float32, len(m.traffic))
	for i, sample := range m.traffic {
		if len(m.traffic) > 1 {
			xs[i] = float32(i) / float32(len(m.traffic)-1)
		}
		up[i] = sample.Up
		down[i] = sample.Down
	}
	return up, down
}
