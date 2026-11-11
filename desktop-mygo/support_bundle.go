package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

const supportBundleFormatVersion = 1

var supportBundleSourceNames = []string{
	"version", "service", "readiness", "memory", "config_diagnostics",
	"rule_sets", "rule_set_auto_update", "subscriptions", "nodes",
	"node_history", "apply_history", "network",
}

type supportBundleRequestState struct {
	Status    string `json:"status"`
	ErrorCode string `json:"error_code,omitempty"`
}

type supportBundleVersion struct {
	Version       string
	KernelVersion string
}

type supportBundleMemory struct {
	Alloc        uint64
	Total        uint64
	Sys          uint64
	NumGC        uint32
	HeapInuse    uint64
	StackInuse   uint64
	NumGoroutine int
}

type supportBundleNode struct {
	Type   string
	Source string
}

type supportBundleSources struct {
	Version       supportBundleVersion
	Service       *model.ServiceStatus
	Readiness     *string
	Memory        *supportBundleMemory
	Diagnostics   *model.ConfigDiagnostics
	RuleSets      []model.RuleSetStatusItem
	RuleSetAuto   *model.RuleSetAutoUpdate
	Subscriptions []model.Subscription
	Nodes         []supportBundleNode
	NodeHistory   map[string]map[string][]model.LatencyPoint
	ApplyHistory  []model.ConfigApplyEvent
	Network       []service.InterfaceInfo
	Preferences   model.UIPreferences
}

type supportBundleCollection struct {
	Sources  supportBundleSources
	Requests map[string]supportBundleRequestState
}

type supportBundleLoaders struct {
	Version       func() (supportBundleVersion, error)
	Service       func() (model.ServiceStatus, error)
	Readiness     func() (string, error)
	Memory        func() (supportBundleMemory, error)
	Diagnostics   func() (model.ConfigDiagnostics, error)
	RuleSets      func() ([]model.RuleSetStatusItem, error)
	RuleSetAuto   func() (model.RuleSetAutoUpdate, error)
	Subscriptions func() ([]model.Subscription, error)
	Nodes         func() ([]supportBundleNode, error)
	NodeHistory   func() (map[string]map[string][]model.LatencyPoint, error)
	ApplyHistory  func() ([]model.ConfigApplyEvent, error)
	Network       func() ([]service.InterfaceInfo, error)
}

func collectSupportBundle(loaders supportBundleLoaders, preferences model.UIPreferences) supportBundleCollection {
	collection := supportBundleCollection{
		Sources:  supportBundleSources{Preferences: preferences},
		Requests: make(map[string]supportBundleRequestState, len(supportBundleSourceNames)),
	}
	collectSupportBundleCore(&collection, loaders)
	collectSupportBundleResources(&collection, loaders)
	return collection
}

func collectSupportBundleCore(collection *supportBundleCollection, loaders supportBundleLoaders) {
	version, state := supportBundleLoad(loaders.Version)
	collection.Sources.Version = version
	collection.Requests["version"] = state
	serviceStatus, state := supportBundleLoad(loaders.Service)
	if state.Status == "ok" {
		collection.Sources.Service = &serviceStatus
	}
	collection.Requests["service"] = state
	readiness, state := supportBundleLoad(loaders.Readiness)
	if state.Status == "ok" {
		collection.Sources.Readiness = &readiness
	}
	collection.Requests["readiness"] = state
	memory, state := supportBundleLoad(loaders.Memory)
	if state.Status == "ok" {
		collection.Sources.Memory = &memory
	}
	collection.Requests["memory"] = state
	diagnostics, state := supportBundleLoad(loaders.Diagnostics)
	if state.Status == "ok" {
		collection.Sources.Diagnostics = &diagnostics
	}
	collection.Requests["config_diagnostics"] = state
	ruleSets, state := supportBundleLoad(loaders.RuleSets)
	if state.Status == "ok" {
		collection.Sources.RuleSets = supportNonNilSlice(ruleSets)
	}
	collection.Requests["rule_sets"] = state
	ruleSetAuto, state := supportBundleLoad(loaders.RuleSetAuto)
	if state.Status == "ok" {
		collection.Sources.RuleSetAuto = &ruleSetAuto
	}
	collection.Requests["rule_set_auto_update"] = state
}

func collectSupportBundleResources(collection *supportBundleCollection, loaders supportBundleLoaders) {
	subscriptions, state := supportBundleLoad(loaders.Subscriptions)
	if state.Status == "ok" {
		collection.Sources.Subscriptions = supportNonNilSlice(subscriptions)
	}
	collection.Requests["subscriptions"] = state
	nodes, state := supportBundleLoad(loaders.Nodes)
	if state.Status == "ok" {
		collection.Sources.Nodes = supportNonNilSlice(nodes)
	}
	collection.Requests["nodes"] = state
	nodeHistory, state := supportBundleLoad(loaders.NodeHistory)
	if state.Status == "ok" {
		collection.Sources.NodeHistory = nodeHistory
	}
	collection.Requests["node_history"] = state
	applyHistory, state := supportBundleLoad(loaders.ApplyHistory)
	if state.Status == "ok" {
		collection.Sources.ApplyHistory = supportNonNilSlice(applyHistory)
	}
	collection.Requests["apply_history"] = state
	network, state := supportBundleLoad(loaders.Network)
	if state.Status == "ok" {
		collection.Sources.Network = supportNonNilSlice(network)
	}
	collection.Requests["network"] = state
}

func supportNonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func supportBundleLoad[T any](load func() (T, error)) (T, supportBundleRequestState) {
	var zero T
	if load == nil {
		return zero, supportBundleRequestState{Status: "unavailable", ErrorCode: "unavailable"}
	}
	value, err := load()
	if err != nil {
		return zero, supportBundleRequestState{Status: "unavailable", ErrorCode: supportBundleErrorCode(err)}
	}
	return value, supportBundleRequestState{Status: "ok"}
}

func supportBundleErrorCode(err error) string {
	var domainErr *service.DomainError
	if errors.As(err, &domainErr) && strings.TrimSpace(domainErr.Code) != "" {
		return supportBundleToken(domainErr.Code)
	}
	return "unavailable"
}

func supportBundleToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	for _, char := range trimmed {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || strings.ContainsRune("_.:/-", char) {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
		if builder.Len() >= 64 {
			break
		}
	}
	return builder.String()
}

func formatSupportBundle(bundle supportBundle) (string, error) {
	body, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}
	return string(body) + "\n", nil
}

func supportBundleFilename(now time.Time) string {
	return "boxd-support-bundle-" + now.UTC().Format("2006-01-02T15-04-05-000Z") + ".json"
}

func countUnavailableSupportBundleSources(requests map[string]supportBundleRequestState) int {
	count := 0
	for _, request := range requests {
		if request.Status == "unavailable" {
			count++
		}
	}
	return count
}
