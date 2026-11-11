package main

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"

	"github.com/xuthus5/boxd/desktop-mygo/internal/runtime"
)

var errRuntimeUnavailable = errors.New("runtime is not available")

type backend interface {
	Snapshot(context.Context) (snapshot, error)
	LoadConfig(context.Context) (string, error)
	LoadConfigSection(context.Context, string) (string, error)
	ValidateConfig(context.Context, string) error
	ValidateConfigSection(context.Context, string, string) error
	ApplyConfig(context.Context, string) error
	ApplyConfigSection(context.Context, string, string) error
	InstallDefaultConfigSection(context.Context, string) error
	ConfigDiagnostics(context.Context) (model.ConfigDiagnostics, error)
	ConfigApplyHistory(context.Context) ([]model.ConfigApplyEvent, error)
	ConfigSnapshot(context.Context, string) (string, error)
	RestoreConfigSnapshot(context.Context, string) error
	SetupStatus(context.Context) (core.SetupStatus, error)
	RuleSetStatus(context.Context) ([]model.RuleSetStatusItem, error)
	UpdateRuleSets(context.Context, core.RuleSetUpdateRequest) (model.RuleSetUpdateResponse, error)
	RuleSetAutoUpdate(context.Context) (model.RuleSetAutoUpdate, error)
	SetRuleSetAutoUpdate(context.Context, model.RuleSetAutoUpdate) error
	StartKernel(context.Context) error
	StopKernel(context.Context) error
	RestartKernel(context.Context) error
	CloseConnection(context.Context, int64) error
	CloseConnections(context.Context, service.ConnectionCloseFilters) (map[string]any, error)
	RefreshSubscription(context.Context, string) error
	RefreshAllSubscriptions(context.Context) (service.RefreshAllResult, error)
	CreateSubscription(context.Context, service.SubscriptionInput) (*model.Subscription, error)
	UpdateSubscription(context.Context, string, service.SubscriptionInput) error
	DeleteSubscription(context.Context, string) error
	SelectSubscription(context.Context, string, string) error
	ImportNode(context.Context, string) (model.ImportResult, error)
	SaveNode(context.Context, service.NodeInput) error
	UpdateNode(context.Context, string, service.NodeInput) error
	DeleteNode(context.Context, string) error
	TestNode(context.Context, service.TestRequest) (model.TestResult, error)
	TestNodes(context.Context, service.TestBatchRequest) ([]model.TestResult, error)
	SetKernelAutostart(context.Context, bool) error
	SetUIPreferences(context.Context, model.UIPreferences) error
	SetTestURL(context.Context, string) error
	ChangePassword(context.Context, string, string) error
	CreateBackup(context.Context, string) error
	ImportLink(context.Context, string) error
	SupportBundle(context.Context) (supportBundleCollection, error)
}

func (b *runtimeBackend) LoadConfig(ctx context.Context) (string, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return "", err
	}
	config, err := svc.Config().GetConfig(ctx)
	if err != nil {
		return "", err
	}
	body, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(body) + "\n", nil
}

func (b *runtimeBackend) ValidateConfig(ctx context.Context, body string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Config().ValidateConfig(ctx, []byte(body), "desktop-mygo")
}

func (b *runtimeBackend) ApplyConfig(ctx context.Context, body string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	_, err = svc.Config().ApplyConfig(ctx, []byte(body), "desktop-mygo")
	return err
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
	b.collectSettings(ctx, svc, &snap)
	b.collectNodes(svc, &snap)
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

func (b *runtimeBackend) collectNodes(svc *service.ServiceSet, snap *snapshot) {
	if svc.Deps.NodeManager == nil {
		return
	}
	nodes := make([]nodeView, 0, len(svc.Deps.NodeManager.List()))
	results := b.rt.NodeTestResults()
	for _, node := range svc.Deps.NodeManager.List() {
		nodes = append(nodes, nodeViewFrom(node, "import", "", results))
	}
	for _, sub := range snap.Subscriptions {
		for _, outbound := range sub.Outbounds {
			nodes = append(nodes, nodeViewFrom(outbound, "subscription", sub.Name, results))
		}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Source != nodes[j].Source {
			return nodes[i].Source < nodes[j].Source
		}
		if nodes[i].SourceName != nodes[j].SourceName {
			return nodes[i].SourceName < nodes[j].SourceName
		}
		return nodes[i].Tag < nodes[j].Tag
	})
	snap.Nodes = nodes
	snap.NodesLoaded = true
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
	_, err := b.ImportNode(ctx, link)
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
