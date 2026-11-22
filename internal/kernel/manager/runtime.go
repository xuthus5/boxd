package manager

import (
	"context"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

// OutboundGroups implements kernel.RuntimeController.
func (m *Manager) OutboundGroups(ctx context.Context) ([]kernel.OutboundGroup, error) {
	engine, err := m.runningEngine()
	if err != nil {
		return nil, err
	}
	result, callErr := engine.OutboundGroups(ctx)
	if callErr == nil {
		return result, nil
	}
	return result, mapOperationError(kernel.ErrCodeInternal, "runtime.outbound_groups", "list outbound groups", callErr)
}

// SelectOutbound implements kernel.RuntimeController.
func (m *Manager) SelectOutbound(ctx context.Context, group, outbound string) error {
	engine, err := m.runningEngine()
	if err != nil {
		return err
	}
	callErr := engine.SelectOutbound(ctx, group, outbound)
	if callErr == nil {
		return nil
	}
	return mapOperationError(kernel.ErrCodeInternal, "runtime.select_outbound", "select outbound", callErr)
}

// OutboundDelay implements kernel.RuntimeController.
func (m *Manager) OutboundDelay(ctx context.Context, tag, url string) (time.Duration, error) {
	engine, err := m.runningEngine()
	if err != nil {
		return 0, err
	}
	delay, callErr := engine.OutboundDelay(ctx, tag, url)
	if callErr == nil {
		return delay, nil
	}
	return delay, mapOperationError(kernel.ErrCodeInternal, "runtime.outbound_delay", "measure outbound delay", callErr)
}

// ClashMode implements kernel.RuntimeController.
func (m *Manager) ClashMode(ctx context.Context) (kernel.ClashMode, error) {
	engine, err := m.runningEngine()
	if err != nil {
		return kernel.ClashMode{}, err
	}
	mode, callErr := engine.ClashMode(ctx)
	if callErr == nil {
		return mode, nil
	}
	return mode, mapOperationError(kernel.ErrCodeInternal, "runtime.clash_mode", "read clash mode", callErr)
}

// SetClashMode implements kernel.RuntimeController.
func (m *Manager) SetClashMode(ctx context.Context, mode string) error {
	engine, err := m.runningEngine()
	if err != nil {
		return err
	}
	callErr := engine.SetClashMode(ctx, mode)
	if callErr == nil {
		return nil
	}
	return mapOperationError(kernel.ErrCodeInternal, "runtime.set_clash_mode", "set clash mode", callErr)
}

// FlushDNS implements kernel.RuntimeController.
func (m *Manager) FlushDNS(ctx context.Context) error {
	engine, err := m.runningEngine()
	if err != nil {
		return err
	}
	callErr := engine.FlushDNS(ctx)
	if callErr == nil {
		return nil
	}
	return mapOperationError(kernel.ErrCodeInternal, "runtime.flush_dns", "flush DNS cache", callErr)
}

// ListConnections implements kernel.ConnectionController.
func (m *Manager) ListConnections(ctx context.Context) ([]kernel.Connection, error) {
	engine, err := m.runningEngine()
	if err != nil {
		return nil, err
	}
	result, callErr := engine.ListConnections(ctx)
	if callErr == nil {
		return result, nil
	}
	return result, mapOperationError(kernel.ErrCodeInternal, "runtime.list_connections", "list connections", callErr)
}

// CloseConnection implements kernel.ConnectionController.
func (m *Manager) CloseConnection(ctx context.Context, id int64) error {
	engine, err := m.runningEngine()
	if err != nil {
		return err
	}
	callErr := engine.CloseConnection(ctx, id)
	if callErr == nil {
		return nil
	}
	return mapOperationError(kernel.ErrCodeInternal, "runtime.close_connection", "close connection", callErr)
}

// CloseConnections implements kernel.ConnectionController.
func (m *Manager) CloseConnections(ctx context.Context, filter kernel.ConnectionFilter) (int, error) {
	engine, err := m.runningEngine()
	if err != nil {
		return 0, err
	}
	count, callErr := engine.CloseConnections(ctx, filter)
	if callErr == nil {
		return count, nil
	}
	return count, mapOperationError(kernel.ErrCodeInternal, "runtime.close_connections", "close connections", callErr)
}

func (m *Manager) runningEngine() (kernel.Engine, error) {
	engine, state := m.currentEngine()
	if state != kernel.StateRunning || engine == nil {
		return nil, kernel.NewError(kernel.ErrCodeNotRunning, "runtime", "kernel is not running").WithHint("start the kernel first")
	}
	return engine, nil
}
