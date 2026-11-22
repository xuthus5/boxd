package singbox

import (
	"context"
	"fmt"
	"time"

	"github.com/sagernet/sing-box/adapter"

	"github.com/xuthus5/boxd/kernel"
)

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

func (e *Engine) OutboundDelay(context.Context, string, string) (time.Duration, error) {
	return 0, featureUnavailable("outbound delay measurement")
}

func (e *Engine) ClashMode(context.Context) (kernel.ClashMode, error) {
	return kernel.ClashMode{}, featureUnavailable("clash mode control")
}

func (e *Engine) SetClashMode(context.Context, string) error {
	return featureUnavailable("clash mode control")
}

func (e *Engine) FlushDNS(context.Context) error {
	return featureUnavailable("DNS flush")
}

func (e *Engine) ListConnections(context.Context) ([]kernel.Connection, error) {
	return nil, featureUnavailable("connection observation")
}

func (e *Engine) CloseConnection(context.Context, int64) error {
	return featureUnavailable("connection control")
}

func (e *Engine) CloseConnections(context.Context, kernel.ConnectionFilter) (int, error) {
	return 0, featureUnavailable("connection control")
}

func (e *Engine) RecentLogs(int) []kernel.LogEntry { return nil }

func (e *Engine) TrafficSnapshot() kernel.TrafficSnapshot {
	return kernel.TrafficSnapshot{At: time.Now().UTC()}
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
		feature+" is not implemented in the current sing-box driver",
	).WithHint("implement the feature behind the kernel contract instead of exposing engine-specific APIs")
}
