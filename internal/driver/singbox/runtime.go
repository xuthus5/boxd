package singbox

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/trafficcontrol"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/experimental/clashmode"
	"github.com/sagernet/sing/service"

	"github.com/xuthus5/boxd/kernel"
)

const defaultDelayTimeout = 10 * time.Second

type engineRuntime struct {
	traffic *trafficcontrol.Manager
	modes   *clashmode.Manager
	dns     adapter.DNSRouter
	cache   adapter.CacheFile
	history *urltest.HistoryStorage

	lastTrafficAt     time.Time
	lastUploadTotal   int64
	lastDownloadTotal int64
}

func bindEngineRuntime(ctx context.Context) engineRuntime {
	return engineRuntime{
		traffic: service.PtrFromContext[trafficcontrol.Manager](ctx),
		modes:   service.PtrFromContext[clashmode.Manager](ctx),
		dns:     service.FromContext[adapter.DNSRouter](ctx),
		cache:   service.FromContext[adapter.CacheFile](ctx),
		history: service.PtrFromContext[urltest.HistoryStorage](ctx),
	}
}

func (e *Engine) OutboundGroups(context.Context) ([]kernel.OutboundGroup, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return nil, kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.outbound_groups", "engine is not running")
	}
	outbounds := e.instance.Outbound().Outbounds()
	groups := make([]kernel.OutboundGroup, 0)
	for _, outbound := range outbounds {
		group, ok := outbound.(adapter.OutboundGroup)
		if !ok {
			continue
		}
		selected := ""
		if current := group.Selected("tcp"); current != nil {
			selected = current.Tag()
		}
		groups = append(groups, kernel.OutboundGroup{
			Type:     outbound.Type(),
			Tag:      outbound.Tag(),
			Selected: selected,
			Members:  append([]string(nil), group.All()...),
		})
	}
	return groups, nil
}

func (e *Engine) SelectOutbound(_ context.Context, groupTag, outboundTag string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.select_outbound", "engine is not running")
	}
	outbound, found := e.instance.Outbound().Outbound(groupTag)
	if !found {
		return kernel.NewError(kernel.ErrCodeNotFound, "driver.singbox.select_outbound", fmt.Sprintf("outbound group %q was not found", groupTag))
	}
	group, ok := outbound.(interface {
		SelectOutbound(string) bool
	})
	if !ok {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "driver.singbox.select_outbound", fmt.Sprintf("outbound %q is not selectable", groupTag))
	}
	if !group.SelectOutbound(outboundTag) {
		return kernel.NewError(kernel.ErrCodeNotFound, "driver.singbox.select_outbound", fmt.Sprintf("outbound %q is not a member of %q", outboundTag, groupTag))
	}
	return nil
}

func (e *Engine) OutboundDelay(ctx context.Context, tag, url string) (time.Duration, error) {
	e.mu.Lock()
	if e.state != kernel.StateRunning || e.instance == nil {
		e.mu.Unlock()
		return 0, kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.outbound_delay", "engine is not running")
	}
	outbound, found := e.instance.Outbound().Outbound(tag)
	history := e.runtime.history
	e.mu.Unlock()
	if !found {
		return 0, kernel.NewError(kernel.ErrCodeNotFound, "driver.singbox.outbound_delay", fmt.Sprintf("outbound %q was not found", tag))
	}
	operationContext := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		operationContext, cancel = context.WithTimeout(ctx, defaultDelayTimeout)
		defer cancel()
	}
	delay, err := urltest.URLTest(operationContext, url, outbound)
	if err != nil {
		return 0, kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.outbound_delay", "measure outbound delay", err)
	}
	if history != nil {
		history.StoreURLTestHistory(tag, &adapter.URLTestHistory{
			Time:  time.Now().UTC(),
			Delay: delay,
		})
	}
	return time.Duration(delay) * time.Millisecond, nil
}

func (e *Engine) ClashMode(context.Context) (kernel.ClashMode, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return kernel.ClashMode{}, kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.clash_mode", "engine is not running")
	}
	if e.runtime.modes == nil {
		return kernel.ClashMode{}, featureUnavailable("clash mode control")
	}
	return kernel.ClashMode{
		Current:   e.runtime.modes.Mode(),
		Available: append([]string(nil), e.runtime.modes.ModeList()...),
	}, nil
}

func (e *Engine) SetClashMode(_ context.Context, mode string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.set_clash_mode", "engine is not running")
	}
	if e.runtime.modes == nil {
		return featureUnavailable("clash mode control")
	}
	mode = strings.TrimSpace(mode)
	canonical := ""
	for _, candidate := range e.runtime.modes.ModeList() {
		if candidate == mode || strings.EqualFold(candidate, mode) {
			canonical = candidate
			break
		}
	}
	if canonical == "" {
		return kernel.NewError(kernel.ErrCodeInvalidConfig, "driver.singbox.set_clash_mode", fmt.Sprintf("unsupported clash mode %q", mode)).
			WithHint("use one of: " + strings.Join(e.runtime.modes.ModeList(), ", "))
	}
	e.runtime.modes.SetMode(canonical)
	return nil
}

func (e *Engine) FlushDNS(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.flush_dns", "engine is not running")
	}
	flushed := false
	if e.runtime.dns != nil {
		e.runtime.dns.ClearCache()
		flushed = true
	}
	if e.runtime.cache != nil {
		if err := e.runtime.cache.ClearDNSCache(); err != nil {
			return kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.flush_dns", "clear persisted DNS cache", err)
		}
		flushed = true
	}
	if !flushed {
		return featureUnavailable("DNS flush")
	}
	return nil
}

func (e *Engine) ListConnections(context.Context) ([]kernel.Connection, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return nil, kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.list_connections", "engine is not running")
	}
	if e.runtime.traffic == nil {
		return nil, featureUnavailable("connection observation")
	}
	metadata := e.runtime.traffic.Connections()
	routerRules := e.instance.Router().Rules()
	connections := make([]kernel.Connection, 0, len(metadata))
	for _, item := range metadata {
		connections = append(connections, connectionView(item, routerRules))
	}
	return connections, nil
}

// ListClosedConnections returns only sing-box's real closed-connection archive.
// The archive is bounded by the engine and is cleared with the engine runtime.
func (e *Engine) ListClosedConnections(context.Context) ([]kernel.Connection, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil || e.runtime.traffic == nil {
		return nil, featureUnavailable("closed connection observation")
	}
	metadata := e.runtime.traffic.ClosedConnections()
	routerRules := e.instance.Router().Rules()
	connections := make([]kernel.Connection, 0, len(metadata))
	for _, item := range metadata {
		connections = append(connections, connectionView(item, routerRules))
	}
	return connections, nil
}

func (e *Engine) CloseConnection(_ context.Context, id int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.close_connection", "engine is not running")
	}
	if e.runtime.traffic == nil {
		return featureUnavailable("connection control")
	}
	for _, metadata := range e.runtime.traffic.Connections() {
		if connectionID(metadata.ID) != id {
			continue
		}
		connection := e.runtime.traffic.Connection(metadata.ID)
		if connection == nil {
			break
		}
		if err := connection.Close(); err != nil {
			return kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.close_connection", "close connection", err)
		}
		return nil
	}
	return kernel.NewError(kernel.ErrCodeNotFound, "driver.singbox.close_connection", fmt.Sprintf("connection %d was not found", id))
}

func (e *Engine) CloseConnections(_ context.Context, filter kernel.ConnectionFilter) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != kernel.StateRunning || e.instance == nil {
		return 0, kernel.NewError(kernel.ErrCodeNotRunning, "driver.singbox.close_connections", "engine is not running")
	}
	if e.runtime.traffic == nil {
		return 0, featureUnavailable("connection control")
	}
	ids := make(map[int64]struct{}, len(filter.IDs))
	for _, id := range filter.IDs {
		ids[id] = struct{}{}
	}
	closed := 0
	for _, metadata := range e.runtime.traffic.Connections() {
		if !connectionMatches(metadata, filter, ids) {
			continue
		}
		connection := e.runtime.traffic.Connection(metadata.ID)
		if connection == nil {
			continue
		}
		if err := connection.Close(); err != nil {
			return closed, kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.close_connections", "close connection", err)
		}
		closed++
	}
	return closed, nil
}

func (e *Engine) RecentLogs(int) []kernel.LogEntry { return nil }

func (e *Engine) TrafficSnapshot() kernel.TrafficSnapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now().UTC()
	if e.state != kernel.StateRunning || e.runtime.traffic == nil {
		return kernel.TrafficSnapshot{At: now}
	}
	upload, download := e.runtime.traffic.Total()
	uploadRate := int64(0)
	downloadRate := int64(0)
	if !e.runtime.lastTrafficAt.IsZero() {
		elapsed := now.Sub(e.runtime.lastTrafficAt).Seconds()
		if elapsed > 0 {
			uploadRate = int64(float64(upload-e.runtime.lastUploadTotal) / elapsed)
			downloadRate = int64(float64(download-e.runtime.lastDownloadTotal) / elapsed)
			if uploadRate < 0 {
				uploadRate = 0
			}
			if downloadRate < 0 {
				downloadRate = 0
			}
		}
	}
	e.runtime.lastTrafficAt = now
	e.runtime.lastUploadTotal = upload
	e.runtime.lastDownloadTotal = download
	return kernel.TrafficSnapshot{
		UploadTotal:   upload,
		DownloadTotal: download,
		UploadRate:    uploadRate,
		DownloadRate:  downloadRate,
		At:            now,
	}
}

func (e *Engine) Subscribe(ctx context.Context, _ ...kernel.EventType) (<-chan kernel.Event, error) {
	channel := make(chan kernel.Event)
	go func() {
		<-ctx.Done()
		close(channel)
	}()
	return channel, nil
}

func featureUnavailable(feature string) *kernel.Error {
	return kernel.NewError(
		kernel.ErrCodeFeatureDisabled,
		"driver.singbox.feature",
		feature+" is unavailable in the current sing-box runtime",
	).WithHint("enable the corresponding sing-box component or use a configuration that provides it")
}

func connectionView(metadata *trafficcontrol.TrackerMetadata, routerRules []adapter.Rule) kernel.Connection {
	process := ""
	if metadata.Metadata.ProcessInfo != nil {
		process = metadata.Metadata.ProcessInfo.ProcessPath
		if process == "" && len(metadata.Metadata.ProcessInfo.AndroidPackageNames) > 0 {
			process = metadata.Metadata.ProcessInfo.AndroidPackageNames[0]
		}
	}
	inbound := metadata.Metadata.Inbound
	if metadata.Metadata.InboundType != "" {
		inbound = metadata.Metadata.InboundType
		if metadata.Metadata.Inbound != "" {
			inbound += "/" + metadata.Metadata.Inbound
		}
	}
	target := metadata.Metadata.Domain
	if target == "" {
		target = metadata.Metadata.Destination.String()
	}
	rule := "final"
	if metadata.Rule != nil {
		rule = metadata.Rule.String()
	}
	return kernel.Connection{
		ID:        connectionID(metadata.ID),
		Inbound:   inbound,
		Network:   metadata.Metadata.Network,
		Source:    metadata.Metadata.Source.String(),
		Target:    target,
		Outbound:  metadata.Outbound,
		Rule:      rule,
		RuleIndex: matchedRuleIndex(metadata.Rule, routerRules),
		Process:   process,
		Upload:    metadata.Upload.Load(),
		Download:  metadata.Download.Load(),
		StartedAt: metadata.CreatedAt.UTC(),
	}
}

func matchedRuleIndex(matched adapter.Rule, rules []adapter.Rule) int {
	if matched == nil {
		return -1
	}
	for index, rule := range rules {
		if rule == matched {
			return index
		}
	}
	return -1
}

func connectionMatches(metadata *trafficcontrol.TrackerMetadata, filter kernel.ConnectionFilter, ids map[int64]struct{}) bool {
	if len(ids) > 0 {
		if _, exists := ids[connectionID(metadata.ID)]; !exists {
			return false
		}
	}
	if filter.Outbound != "" && metadata.Outbound != filter.Outbound {
		return false
	}
	if filter.Rule != "" {
		rule := "final"
		if metadata.Rule != nil {
			rule = metadata.Rule.String()
		}
		if rule != filter.Rule {
			return false
		}
	}
	if filter.Process != "" {
		process := ""
		if metadata.Metadata.ProcessInfo != nil {
			process = metadata.Metadata.ProcessInfo.ProcessPath
			if process == "" && len(metadata.Metadata.ProcessInfo.AndroidPackageNames) > 0 {
				process = metadata.Metadata.ProcessInfo.AndroidPackageNames[0]
			}
		}
		if process != filter.Process {
			return false
		}
	}
	return true
}

func connectionID(id [16]byte) int64 {
	value := binary.BigEndian.Uint64(id[:8]) ^ binary.BigEndian.Uint64(id[8:])
	return int64(value & math.MaxInt64)
}
