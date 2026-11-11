package main

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

var errRuntimeUnavailable = errors.New("runtime is not available")

type backend interface {
	Snapshot(context.Context) (snapshot, error)
	StartKernel(context.Context) error
	StopKernel(context.Context) error
	RestartKernel(context.Context) error
	CloseConnection(context.Context, int64) error
	RefreshSubscription(context.Context, string) error
	SetKernelAutostart(context.Context, bool) error
	SetUIPreferences(context.Context, model.UIPreferences) error
	SetTestURL(context.Context, string) error
	ImportLink(context.Context, string) error
}

type runtimeBackend struct {
	rt *runtime.Runtime
}

func newRuntimeBackend(rt *runtime.Runtime) *runtimeBackend {
	return &runtimeBackend{rt: rt}
}

func (b *runtimeBackend) Snapshot(ctx context.Context) (snapshot, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return snapshot{}, err
	}
	if err := ctx.Err(); err != nil {
		return snapshot{}, err
	}
	snap := snapshot{}
	b.collectStatus(ctx, svc, &snap)
	b.collectStats(ctx, svc, &snap)
	b.collectNodes(svc, &snap)
	b.collectSettings(ctx, svc, &snap)
	snap.Logs = sortedLogs(b.rt.Logs())
	snap.LogsLoaded = true
	return snap, nil
}

func (b *runtimeBackend) collectStatus(ctx context.Context, svc *service.ServiceSet, snap *snapshot) {
	status, err := svc.Service().ServiceStatus(ctx)
	if err != nil {
		snap.addWarning("service status", err)
	} else {
		snap.Status = &status
	}
}

func (b *runtimeBackend) collectStats(ctx context.Context, svc *service.ServiceSet, snap *snapshot) {
	if traffic, err := svc.Stats().Traffic(ctx); err != nil {
		snap.addWarning("traffic", err)
	} else {
		snap.Traffic = trafficTotalsFrom(traffic)
	}
	if connections, err := svc.Stats().Connections(ctx); err != nil {
		snap.addWarning("connections", err)
	} else {
		snap.Connections = connectionViewsFrom(connections)
		snap.ConnectionsLoaded = true
	}
}

func (b *runtimeBackend) collectNodes(svc *service.ServiceSet, snap *snapshot) {
	if svc.Deps.NodeManager != nil {
		snap.Nodes = nodeViewsFrom(svc.Deps.NodeManager.List(), b.rt.NodeTestResults())
		snap.NodesLoaded = true
	}
}

func (b *runtimeBackend) collectSettings(ctx context.Context, svc *service.ServiceSet, snap *snapshot) {
	if subs, err := svc.Subscriptions().List(ctx); err != nil {
		snap.addWarning("subscriptions", err)
	} else {
		snap.Subscriptions = subs
		snap.SubscriptionsLoaded = true
	}
	if prefs, err := svc.Settings().GetUIPreferences(ctx); err != nil {
		snap.addWarning("preferences", err)
	} else {
		snap.Preferences = prefs
		snap.PreferencesLoaded = true
	}
	if autostart, err := svc.Settings().GetKernelAutostart(ctx); err != nil {
		snap.addWarning("kernel autostart", err)
	} else {
		snap.KernelAutostart = autostart["enabled"]
		snap.KernelAutostartLoaded = true
	}
	if testURL, err := svc.Settings().GetTestURL(ctx); err != nil {
		snap.addWarning("test URL", err)
	} else {
		snap.TestURL = testURL["url"]
		snap.TestURLLoaded = true
	}
}

func (b *runtimeBackend) StartKernel(context.Context) error {
	if _, err := b.serviceSet(); err != nil {
		return err
	}
	return b.rt.StartKernel()
}

func (b *runtimeBackend) StopKernel(ctx context.Context) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Service().ServiceStop(ctx)
}

func (b *runtimeBackend) RestartKernel(ctx context.Context) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Service().ServiceRestart(ctx)
}

func (b *runtimeBackend) CloseConnection(ctx context.Context, id int64) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Stats().CloseConnection(ctx, id)
	return err
}

func (b *runtimeBackend) RefreshSubscription(ctx context.Context, id string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Subscriptions().Refresh(ctx, id)
}

func (b *runtimeBackend) SetKernelAutostart(ctx context.Context, enabled bool) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Settings().SetKernelAutostart(ctx, enabled)
	return err
}

func (b *runtimeBackend) SetUIPreferences(ctx context.Context, prefs model.UIPreferences) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Settings().SetUIPreferences(ctx, prefs)
	return err
}

func (b *runtimeBackend) SetTestURL(ctx context.Context, url string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Settings().SetTestURL(ctx, url)
	return err
}

func (b *runtimeBackend) ImportLink(ctx context.Context, link string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Import().ParseLink(ctx, link)
	return err
}

func (b *runtimeBackend) serviceSet() (*service.ServiceSet, error) {
	if b == nil || b.rt == nil || b.rt.Service() == nil {
		return nil, errRuntimeUnavailable
	}
	return b.rt.Service(), nil
}

func (s *snapshot) addWarning(section string, err error) {
	if err == nil {
		return
	}
	s.Warnings = append(s.Warnings, section+": "+shortError(err))
}

func trafficTotalsFrom(values map[string]any) *trafficTotals {
	total := &trafficTotals{}
	switch value := values["upload_bytes"].(type) {
	case int64:
		total.Up = value
	case int:
		total.Up = int64(value)
	case float64:
		total.Up = int64(value)
	}
	switch value := values["download_bytes"].(type) {
	case int64:
		total.Down = value
	case int:
		total.Down = int64(value)
	case float64:
		total.Down = int64(value)
	}
	return total
}

func connectionViewsFrom(values map[string]any) []connectionView {
	list, _ := values["list"].([]core.TrafficConn)
	out := make([]connectionView, 0, len(list))
	now := time.Now()
	for _, conn := range list {
		out = append(out, connectionView{
			ID:       conn.ID,
			Target:   conn.Target,
			Outbound: conn.Outbound,
			Rule:     conn.Rule,
			Upload:   conn.Upload,
			Download: conn.Download,
			Duration: max(0, now.Sub(conn.Start)).Round(time.Second).String(),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func nodeViewsFrom(nodes []model.Outbound, results map[string]map[string]model.TestResult) []nodeView {
	out := make([]nodeView, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, nodeView{
			Tag:    node.Tag,
			Type:   node.Type,
			Server: node.Server,
			Port:   node.Port,
			Test:   nodeTestSummary(results, node.Tag),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

func nodeTestSummary(results map[string]map[string]model.TestResult, tag string) string {
	var latest model.TestResult
	found := false
	for key, byType := range results {
		if key != tag && !strings.HasPrefix(key, tag+"_") {
			continue
		}
		for _, result := range byType {
			if !found || result.Timestamp.After(latest.Timestamp) {
				latest, found = result, true
			}
		}
	}
	if !found {
		return "Not tested"
	}
	if !latest.Success {
		return "Failed"
	}
	return strconv.FormatFloat(latest.LatencyMs, 'f', 0, 64) + " ms"
}

func sortedLogs(entries []core.LogEntry) []core.LogEntry {
	out := append([]core.LogEntry(nil), entries...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}
