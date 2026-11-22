package manager

import (
	"context"
	"errors"

	"github.com/xuthus5/boxd/kernel"
)

// Start opens and starts the engine. Starting an already running manager is a no-op.
func (m *Manager) Start(ctx context.Context) error {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if err := m.ensureUsableLocked(); err != nil {
		return err
	}
	state := m.currentState()
	if state == kernel.StateRunning {
		return nil
	}
	return m.startLocked(ctx)
}

// Stop stops and closes the current engine.
func (m *Manager) Stop(ctx context.Context) error {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if err := m.ensureUsableLocked(); err != nil {
		return err
	}
	return m.stopLocked(ctx)
}

// Restart performs Stop followed by Start while holding the single writer lock.
func (m *Manager) Restart(ctx context.Context) error {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if err := m.ensureUsableLocked(); err != nil {
		return err
	}
	if state := m.currentState(); state == kernel.StateRunning {
		if err := m.stopLocked(ctx); err != nil {
			return err
		}
	}
	return m.startLocked(ctx)
}

// Close releases the engine and is safe to call more than once.
func (m *Manager) Close() error {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if m.closed {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeouts.Stop)
	defer cancel()
	err := m.stopLocked(ctx)
	m.closed = true
	return err
}

func (m *Manager) startLocked(ctx context.Context) error {
	m.transition(kernel.StateStarting, nil)
	operationCtx, cancel := context.WithTimeout(ctx, m.timeouts.Start)
	defer cancel()

	document, _ := m.currentConfig()
	openOptions := m.open
	openOptions.Config = document
	engine, err := m.factory.Open(operationCtx, openOptions)
	if err != nil {
		transitionErr := mapOperationError(kernel.ErrCodeInternal, "kernel.start", "open engine", err)
		m.transition(kernel.StateFailed, transitionErr)
		return transitionErr
	}
	if startErr := engine.Start(operationCtx); startErr != nil {
		_ = engine.Close()
		transitionErr := mapOperationError(kernel.ErrCodeInternal, "kernel.start", "start engine", startErr)
		m.transition(kernel.StateFailed, transitionErr)
		return transitionErr
	}

	m.stateMu.Lock()
	m.engine = engine
	m.version = engine.Version()
	m.stateMu.Unlock()
	m.transition(kernel.StateRunning, nil)
	return nil
}

func (m *Manager) stopLocked(ctx context.Context) error {
	if m.currentState() == kernel.StateStopped && m.engineForClose() == nil {
		return nil
	}
	m.transition(kernel.StateStopping, nil)
	operationCtx, cancel := context.WithTimeout(ctx, m.timeouts.Stop)
	defer cancel()

	engine := m.engineForClose()
	if engine == nil {
		m.transition(kernel.StateStopped, nil)
		return nil
	}
	stopErr := engine.Stop(operationCtx)
	closeErr := engine.Close()
	m.stateMu.Lock()
	m.engine = nil
	m.stateMu.Unlock()

	joined := errors.Join(stopErr, closeErr)
	if joined != nil {
		transitionErr := mapOperationError(kernel.ErrCodeInternal, "kernel.stop", "stop engine", joined)
		m.transition(kernel.StateFailed, transitionErr)
		return transitionErr
	}
	m.transition(kernel.StateStopped, nil)
	return nil
}

// restoreAfterFailedApply closes a possibly broken engine and restarts the last
// known good configuration when one exists.
func (m *Manager) restoreAfterFailedApply(ctx context.Context, previous kernel.ConfigDocument, hadConfig bool) error {
	if engine := m.engineForClose(); engine != nil {
		_ = engine.Stop(context.Background())
		_ = engine.Close()
		m.stateMu.Lock()
		m.engine = nil
		m.stateMu.Unlock()
	}
	m.setConfig(previous, hadConfig)
	m.transition(kernel.StateStopped, nil)
	if !hadConfig {
		return nil
	}
	return m.startLocked(ctx)
}

func (m *Manager) ensureUsableLocked() error {
	if m.closed {
		return kernel.NewError(kernel.ErrCodeInvalidState, "kernel", "kernel manager is closed")
	}
	return nil
}

func (m *Manager) currentState() kernel.State {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.state
}

func (m *Manager) engineForClose() kernel.Engine {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.engine
}
