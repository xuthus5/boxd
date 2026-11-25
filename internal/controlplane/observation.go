package controlplane

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const observationRetention = 10 * time.Minute

type observedConnection struct {
	Upload     int64
	Download   int64
	RuleID     string
	LastSeenAt time.Time
}

type trafficAccumulator struct {
	Traffic      control.ObservedTraffic
	LastUpload   int64
	LastDownload int64
}

type observationState struct {
	runKey       string
	lastSampleAt time.Time
	connections  map[int64]observedConnection
	nodes        map[string]*trafficAccumulator
	inbounds     map[string]*trafficAccumulator
	ruleHits     map[string]uint64
}

// liveObservation converts real kernel connection snapshots into bounded,
// per-entity counters. Counters only advance from values the engine reports.
type liveObservation struct {
	mu    sync.Mutex
	state observationState
}

func newLiveObservation() *liveObservation {
	return &liveObservation{state: emptyObservationState()}
}

func emptyObservationState() observationState {
	return observationState{
		connections: make(map[int64]observedConnection),
		nodes:       make(map[string]*trafficAccumulator),
		inbounds:    make(map[string]*trafficAccumulator),
		ruleHits:    make(map[string]uint64),
	}
}

func (o *liveObservation) update(
	status kernel.Status,
	connections []kernel.Connection,
	closedConnections []kernel.Connection,
	routes control.RouteProfile,
	inbounds []control.InboundProfile,
	nodes []control.Node,
	chains []control.ProxyChain,
	now time.Time,
) control.ObservationSnapshot {
	if o == nil {
		return emptyObservationSnapshot()
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if status.State != kernel.StateRunning || status.StartedAt.IsZero() {
		o.state = emptyObservationState()
		return emptyObservationSnapshot()
	}
	runKey := status.StartedAt.UTC().Format(time.RFC3339Nano) + "\x00" + status.Revision
	if o.state.runKey != runKey {
		o.state = emptyObservationState()
		o.state.runKey = runKey
	}

	nodeByTag := make(map[string]string, len(nodes))
	for _, node := range nodes {
		nodeByTag[control.NodeTag(node.ID)] = node.ID
	}
	chainByTag := make(map[string]control.ProxyChain, len(chains))
	for _, chain := range chains {
		chainByTag[control.ChainTag(chain.ID)] = chain
	}
	ruleIDs := compiledRouteRuleIDs(routes, inbounds)
	observed := make(map[int64]struct{}, len(connections)+len(closedConnections))
	record := func(connection kernel.Connection) {
		if _, duplicate := observed[connection.ID]; duplicate {
			return
		}
		observed[connection.ID] = struct{}{}
		previous, seen := o.state.connections[connection.ID]
		uploadDelta := connection.Upload
		downloadDelta := connection.Download
		if seen {
			uploadDelta = nonNegativeDelta(connection.Upload, previous.Upload)
			downloadDelta = nonNegativeDelta(connection.Download, previous.Download)
		}
		if nodeID := resolveConnectionNode(connection.Outbound, nodeByTag, chainByTag, nodes); nodeID != "" {
			o.addNode(nodeID, uploadDelta, downloadDelta)
		}
		for _, inbound := range inbounds {
			if connectionInboundMatches(connection.Inbound, inbound) {
				o.addInbound(inbound.ID, uploadDelta, downloadDelta)
				break
			}
		}
		ruleID := ""
		if connection.RuleIndex >= 0 {
			ruleID = ruleIDs[connection.RuleIndex]
		}
		if ruleID != "" && (!seen || previous.RuleID != ruleID) {
			o.state.ruleHits[ruleID]++
		}
		o.state.connections[connection.ID] = observedConnection{
			Upload:     connection.Upload,
			Download:   connection.Download,
			RuleID:     ruleID,
			LastSeenAt: now,
		}
	}
	for _, connection := range connections {
		record(connection)
	}
	for _, connection := range closedConnections {
		record(connection)
	}
	o.removeStaleConnections(observed, now)
	o.updateRates(now)
	return o.snapshot(now)
}

func (o *liveObservation) addNode(nodeID string, upload, download int64) {
	counter := o.state.nodes[nodeID]
	if counter == nil {
		counter = &trafficAccumulator{}
		o.state.nodes[nodeID] = counter
	}
	counter.Traffic.UploadTotal += upload
	counter.Traffic.DownloadTotal += download
}

func (o *liveObservation) addInbound(inboundID string, upload, download int64) {
	counter := o.state.inbounds[inboundID]
	if counter == nil {
		counter = &trafficAccumulator{}
		o.state.inbounds[inboundID] = counter
	}
	counter.Traffic.UploadTotal += upload
	counter.Traffic.DownloadTotal += download
}

func (o *liveObservation) updateRates(now time.Time) {
	if o.state.lastSampleAt.IsZero() {
		o.state.lastSampleAt = now
		for _, counter := range o.state.nodes {
			counter.LastUpload = counter.Traffic.UploadTotal
			counter.LastDownload = counter.Traffic.DownloadTotal
		}
		for _, counter := range o.state.inbounds {
			counter.LastUpload = counter.Traffic.UploadTotal
			counter.LastDownload = counter.Traffic.DownloadTotal
		}
		return
	}
	elapsed := now.Sub(o.state.lastSampleAt)
	if elapsed < time.Second {
		return
	}
	seconds := elapsed.Seconds()
	for _, counter := range o.state.nodes {
		counter.Traffic.UploadRate = ratePerSecond(counter.Traffic.UploadTotal-counter.LastUpload, seconds)
		counter.Traffic.DownloadRate = ratePerSecond(counter.Traffic.DownloadTotal-counter.LastDownload, seconds)
		counter.LastUpload = counter.Traffic.UploadTotal
		counter.LastDownload = counter.Traffic.DownloadTotal
	}
	for _, counter := range o.state.inbounds {
		counter.Traffic.UploadRate = ratePerSecond(counter.Traffic.UploadTotal-counter.LastUpload, seconds)
		counter.Traffic.DownloadRate = ratePerSecond(counter.Traffic.DownloadTotal-counter.LastDownload, seconds)
		counter.LastUpload = counter.Traffic.UploadTotal
		counter.LastDownload = counter.Traffic.DownloadTotal
	}
	o.state.lastSampleAt = now
}

func (o *liveObservation) removeStaleConnections(active map[int64]struct{}, now time.Time) {
	for id, connection := range o.state.connections {
		if _, exists := active[id]; exists {
			continue
		}
		if now.Sub(connection.LastSeenAt) >= observationRetention {
			delete(o.state.connections, id)
		}
	}
}

func (o *liveObservation) snapshot(now time.Time) control.ObservationSnapshot {
	result := control.ObservationSnapshot{
		Available: true,
		SampledAt: now.UTC(),
		Nodes:     make(map[string]control.ObservedTraffic, len(o.state.nodes)),
		Inbounds:  make(map[string]control.ObservedTraffic, len(o.state.inbounds)),
		RuleHits:  make(map[string]uint64, len(o.state.ruleHits)),
	}
	for id, counter := range o.state.nodes {
		result.Nodes[id] = counter.Traffic
	}
	for id, counter := range o.state.inbounds {
		result.Inbounds[id] = counter.Traffic
	}
	for id, hits := range o.state.ruleHits {
		result.RuleHits[id] = hits
	}
	return result
}

func emptyObservationSnapshot() control.ObservationSnapshot {
	return control.ObservationSnapshot{
		Nodes:    map[string]control.ObservedTraffic{},
		Inbounds: map[string]control.ObservedTraffic{},
		RuleHits: map[string]uint64{},
	}
}

func nonNegativeDelta(current, previous int64) int64 {
	if current <= previous {
		return 0
	}
	return current - previous
}

func ratePerSecond(delta int64, seconds float64) int64 {
	if delta <= 0 || seconds <= 0 {
		return 0
	}
	return int64(float64(delta) / seconds)
}

func resolveConnectionNode(tag string, nodeByTag map[string]string, chains map[string]control.ProxyChain, nodes []control.Node) string {
	tag = strings.TrimSpace(tag)
	if nodeID, exists := nodeByTag[tag]; exists {
		return nodeID
	}
	if chain, exists := chains[tag]; exists && len(chain.Hops) > 0 {
		last := chain.Hops[len(chain.Hops)-1]
		if last.Kind == control.EgressKindNode {
			return last.ID
		}
	}
	for _, node := range nodes {
		if strings.HasSuffix(tag, "-"+control.NodeTag(node.ID)) {
			return node.ID
		}
	}
	return ""
}

func connectionInboundMatches(value string, inbound control.InboundProfile) bool {
	value = strings.TrimSpace(value)
	if index := strings.LastIndexByte(value, '/'); index >= 0 && index+1 < len(value) {
		value = value[index+1:]
	}
	return value == inbound.ID || value == control.InboundTag(inbound.ID) || strings.EqualFold(value, inbound.Name)
}

func compiledRouteRuleIDs(routes control.RouteProfile, inbounds []control.InboundProfile) map[int]string {
	result := make(map[int]string)
	if routes.Mode != control.RouteModeRules {
		return result
	}
	index := 1 // sing-box always starts with the automatic sniff action.
	for _, inbound := range inbounds {
		if inbound.Enabled && inbound.Egress != nil {
			index++
		}
	}
	if routes.Presets.BypassLAN {
		index++
	}
	if routes.Presets.ChinaDirect {
		index++
	}
	if routes.Presets.AdBlock {
		index++
	}
	for _, rule := range routes.Rules {
		if rule.Disabled {
			continue
		}
		result[index] = rule.ID
		index++
	}
	return result
}

type ObservabilityService struct {
	manager kernel.Manager
	profile *ProfileService
	live    *liveObservation

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

var _ control.ObservabilityAPI = (*ObservabilityService)(nil)

func NewObservabilityService(manager kernel.Manager, profile *ProfileService) *ObservabilityService {
	return &ObservabilityService{
		manager: manager,
		profile: profile,
		live:    newLiveObservation(),
	}
}

func (s *ObservabilityService) StartSampling(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return nil
	}
	// Sampling belongs to the application lifetime, not to the short-lived
	// context callers use while constructing the control plane. The scheduler
	// follows the same rule; StopSampling remains the explicit shutdown path.
	samplerCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		s.sample(samplerCtx)
		for {
			select {
			case <-samplerCtx.Done():
				return
			case <-ticker.C:
				s.sample(samplerCtx)
			}
		}
	}()
	return nil
}

func (s *ObservabilityService) StopSampling() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (s *ObservabilityService) Snapshot(ctx context.Context) (control.ObservationSnapshot, error) {
	return s.sample(ctx), nil
}

func (s *ObservabilityService) sample(ctx context.Context) control.ObservationSnapshot {
	if s.manager == nil {
		return emptyObservationSnapshot()
	}
	status := s.manager.Status()
	now := time.Now().UTC()
	if status.State != kernel.StateRunning {
		return s.live.update(status, nil, nil, control.RouteProfile{}, nil, nil, nil, now)
	}
	connections, err := s.manager.ListConnections(ctx)
	if err != nil && !isKernelNotRunning(err) {
		return emptyObservationSnapshot()
	}
	closedConnections, err := s.manager.ListClosedConnections(ctx)
	if err != nil && !isKernelNotRunning(err) {
		return emptyObservationSnapshot()
	}
	if s.profile == nil {
		return s.live.update(status, connections, closedConnections, control.RouteProfile{}, nil, nil, nil, now)
	}
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return emptyObservationSnapshot()
	}
	return s.live.update(status, connections, closedConnections, state.Routes, state.Inbounds, state.Nodes, state.Chains, now)
}

func (s *ObservabilityService) RecentLogs(limit int) []kernel.LogEntry {
	return s.manager.RecentLogs(limit)
}

func (s *ObservabilityService) Traffic(context.Context) (kernel.TrafficSnapshot, error) {
	return s.manager.TrafficSnapshot(), nil
}

func (s *ObservabilityService) Connections(ctx context.Context) ([]kernel.Connection, error) {
	return s.manager.ListConnections(ctx)
}

func (s *ObservabilityService) ClosedConnections(ctx context.Context) ([]kernel.Connection, error) {
	return s.manager.ListClosedConnections(ctx)
}

func (s *ObservabilityService) CloseConnection(ctx context.Context, id int64) error {
	return s.manager.CloseConnection(ctx, id)
}

func (s *ObservabilityService) CloseConnections(ctx context.Context, filter kernel.ConnectionFilter) (int, error) {
	return s.manager.CloseConnections(ctx, filter)
}

func (s *ObservabilityService) Subscribe(ctx context.Context, types ...kernel.EventType) (<-chan kernel.Event, error) {
	return s.manager.Subscribe(ctx, types...)
}

func isKernelNotRunning(err error) bool {
	var serviceErr *kernel.Error
	return errors.As(err, &serviceErr) && (serviceErr.Code == kernel.ErrCodeNotRunning || serviceErr.Code == kernel.ErrCodeInvalidState)
}
