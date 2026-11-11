package main

import (
	"context"
	"errors"
	goruntime "runtime"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

func (b *runtimeBackend) SupportBundle(ctx context.Context) (supportBundleCollection, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return supportBundleCollection{}, err
	}
	preferences, _ := svc.Settings().GetUIPreferences(ctx)
	return collectSupportBundle(runtimeSupportBundleLoaders(ctx, svc), preferences), nil
}

func runtimeSupportBundleLoaders(ctx context.Context, svc *service.ServiceSet) supportBundleLoaders {
	return supportBundleLoaders{
		Version: func() (supportBundleVersion, error) {
			version := supportBundleVersion{Version: core.Version}
			status, statusErr := svc.Service().ServiceStatus(ctx)
			if statusErr == nil {
				version.KernelVersion = status.Version
			}
			return version, nil
		},
		Service: func() (model.ServiceStatus, error) {
			return svc.Service().ServiceStatus(ctx)
		},
		Readiness: func() (string, error) {
			result, err := svc.Health().Readiness(ctx)
			if err != nil {
				return "", err
			}
			return result["status"], nil
		},
		Memory: func() (supportBundleMemory, error) {
			var stats goruntime.MemStats
			goruntime.ReadMemStats(&stats)
			return supportBundleMemory{
				Alloc: stats.Alloc, Total: stats.TotalAlloc, Sys: stats.Sys, NumGC: stats.NumGC,
				HeapInuse: stats.HeapInuse, StackInuse: stats.StackInuse, NumGoroutine: goruntime.NumGoroutine(),
			}, nil
		},
		Diagnostics: func() (model.ConfigDiagnostics, error) {
			return svc.Config().Diagnostics(), nil
		},
		RuleSets: func() ([]model.RuleSetStatusItem, error) {
			return svc.RuleSets().Status(ctx)
		},
		RuleSetAuto: func() (model.RuleSetAutoUpdate, error) {
			return svc.RuleSets().AutoUpdate()
		},
		Subscriptions: func() ([]model.Subscription, error) {
			return svc.Subscriptions().List(ctx)
		},
		Nodes:        func() ([]supportBundleNode, error) { return supportNodes(svc) },
		NodeHistory:  func() (map[string]map[string][]model.LatencyPoint, error) { return supportNodeHistory(svc) },
		ApplyHistory: func() ([]model.ConfigApplyEvent, error) { return svc.Config().ApplyHistory() },
		Network:      func() ([]service.InterfaceInfo, error) { return svc.Network().ListInterfaces(ctx) },
	}
}

func supportNodes(svc *service.ServiceSet) ([]supportBundleNode, error) {
	if svc.Deps.NodeManager == nil || svc.Deps.SubManager == nil {
		return nil, errors.New("node managers are not available")
	}
	nodes := make([]supportBundleNode, 0, len(svc.Deps.NodeManager.List()))
	for _, node := range svc.Deps.NodeManager.List() {
		nodes = append(nodes, supportBundleNode{Type: node.Type, Source: "import"})
	}
	subscriptions, err := svc.Deps.SubManager.List()
	if err != nil {
		return nil, err
	}
	for _, subscription := range subscriptions {
		for _, outbound := range subscription.Outbounds {
			nodes = append(nodes, supportBundleNode{Type: outbound.Type, Source: "subscription"})
		}
	}
	return nodes, nil
}

func supportNodeHistory(svc *service.ServiceSet) (map[string]map[string][]model.LatencyPoint, error) {
	if svc.Deps.NodeManager == nil {
		return nil, errors.New("node manager is not available")
	}
	return svc.Deps.NodeManager.GetAllTestHistory(), nil
}
