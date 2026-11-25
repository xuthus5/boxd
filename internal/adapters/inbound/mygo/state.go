package mygo

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

type snapshot struct {
	Kernel        kernel.Status
	Capabilities  kernel.EngineDescriptor
	Scheduler     control.SchedulerStatus
	Inbounds      []control.InboundProfile
	Catalog       control.EgressCatalog
	Routes        control.RouteProfile
	DNS           control.DNSProfile
	RuleSets      []control.RuleSet
	Subscriptions []control.Subscription
	Settings      []control.Setting
	Connections   []kernel.Connection
	Traffic       kernel.TrafficSnapshot
	Observation   control.ObservationSnapshot
	Activity      []control.ActivityLogEntry
	KernelLogs    []kernel.LogEntry
	Audit         []control.AuditEvent

	SubscriptionsByID map[string]control.Subscription
	NodesByID         map[string]control.Node
	GroupsByID        map[string]control.Group
	ChainsByID        map[string]control.ProxyChain
	SettingsByKey     map[string]string
}

func loadSnapshot(ctx context.Context, app control.Application) (snapshot, error) {
	var result snapshot
	if app == nil {
		return result, errors.New("control plane is not ready")
	}
	var err error
	result.Kernel, err = app.Kernel().Status(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Capabilities, err = app.Setup().Capabilities(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Scheduler, err = app.Scheduler().Status(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Inbounds, err = app.Inbounds().List(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Catalog, err = app.Egress().Catalog(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Routes, err = app.Routes().Get(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.DNS, err = app.DNS().Get(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.RuleSets, err = app.RuleSets().List(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Subscriptions, err = app.Subscriptions().List(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Settings, err = app.Settings().List(ctx)
	if err != nil {
		return snapshot{}, err
	}
	result.Connections, err = app.Observability().Connections(ctx)
	if err != nil && !kernelUnavailable(err) {
		return snapshot{}, err
	}
	result.Traffic, err = app.Observability().Traffic(ctx)
	if err != nil && !kernelUnavailable(err) {
		return snapshot{}, err
	}
	result.Observation, err = app.Observability().Snapshot(ctx)
	if err != nil && !kernelUnavailable(err) {
		return snapshot{}, err
	}
	result.Activity = app.ActivityLogs().Recent(400)
	result.KernelLogs = app.Observability().RecentLogs(400)
	result.Audit, err = app.Audit().List(ctx, control.AuditFilter{Limit: 100})
	if err != nil {
		result.Audit = nil
	}

	result.SubscriptionsByID = make(map[string]control.Subscription, len(result.Subscriptions))
	for _, subscription := range result.Subscriptions {
		result.SubscriptionsByID[subscription.ID] = subscription
	}
	result.NodesByID = make(map[string]control.Node, len(result.Catalog.Nodes))
	for _, node := range result.Catalog.Nodes {
		result.NodesByID[node.ID] = node
	}
	result.GroupsByID = make(map[string]control.Group, len(result.Catalog.Groups))
	for _, group := range result.Catalog.Groups {
		result.GroupsByID[group.ID] = group
	}
	result.ChainsByID = make(map[string]control.ProxyChain, len(result.Catalog.Chains))
	for _, chain := range result.Catalog.Chains {
		result.ChainsByID[chain.ID] = chain
	}
	result.SettingsByKey = make(map[string]string, len(result.Settings))
	for _, setting := range result.Settings {
		result.SettingsByKey[setting.Key] = setting.Value
	}
	return result, nil
}

func kernelUnavailable(err error) bool {
	var serviceErr *kernel.Error
	return errors.As(err, &serviceErr) && (serviceErr.Code == kernel.ErrCodeNotRunning || serviceErr.Code == kernel.ErrCodeInvalidState)
}

func (s snapshot) nodeCountBySubscription() map[string]int {
	result := make(map[string]int)
	for _, node := range s.Catalog.Nodes {
		if node.SubscriptionID != "" {
			result[node.SubscriptionID]++
		}
	}
	return result
}

func (s snapshot) groupNamesForNode(nodeID string) []string {
	var result []string
	for _, group := range s.Catalog.Groups {
		for _, member := range group.Members {
			if member == nodeID {
				result = append(result, group.Name)
				break
			}
		}
	}
	return result
}

type mergedLog struct {
	ID      string
	Time    time.Time
	Level   string
	Source  string
	Message string
	App     bool
}

func (s snapshot) mergedLogs(appOnly, kernelOnly bool) []mergedLog {
	result := make([]mergedLog, 0, len(s.Activity)+len(s.KernelLogs))
	if !kernelOnly {
		for _, entry := range s.Activity {
			result = append(result, mergedLog{
				ID: "app-" + formatUint(entry.ID), Time: entry.Time, Level: entry.Level,
				Source: entry.Source, Message: entry.Message, App: true,
			})
		}
	}
	if !appOnly {
		for index, entry := range s.KernelLogs {
			result = append(result, mergedLog{
				ID: "kernel-" + formatUint(uint64(index)), Time: entry.Time, Level: strings.ToLower(entry.Level),
				Source: entry.Source, Message: entry.Message,
			})
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Time.Before(result[j].Time) })
	return result
}

func formatUint(value uint64) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}

func settingBool(values map[string]string, key string, fallback bool) bool {
	value, exists := values[key]
	if !exists {
		return fallback
	}
	return strings.EqualFold(value, "true") || value == "1"
}

type clock struct {
	mu  sync.RWMutex
	now time.Time
}

func newClock() *clock { return &clock{now: time.Now()} }

func (c *clock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *clock) Tick() {
	c.mu.Lock()
	c.now = time.Now()
	c.mu.Unlock()
}
