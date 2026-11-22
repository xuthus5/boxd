// Package manager owns the single-writer kernel state machine.
package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xuthus5/boxd/internal/kernel/events"
	"github.com/xuthus5/boxd/internal/kernel/logbuf"
	"github.com/xuthus5/boxd/kernel"
)

const (
	defaultStartTimeout = 30 * time.Second
	defaultStopTimeout  = 10 * time.Second
	defaultApplyTimeout = 60 * time.Second
	defaultLogCapacity  = 256
	defaultEventBuffer  = 64
)

// Timeouts bounds state-changing operations.
type Timeouts struct {
	Start time.Duration
	Stop  time.Duration
	Apply time.Duration
}

// DefaultTimeouts returns the accepted defaults from the design document.
func DefaultTimeouts() Timeouts {
	return Timeouts{
		Start: defaultStartTimeout,
		Stop:  defaultStopTimeout,
		Apply: defaultApplyTimeout,
	}
}

// Options configures a kernel manager.
type Options struct {
	Factory     kernel.Factory
	Store       kernel.ConfigStore
	Open        kernel.OpenOptions
	Timeouts    Timeouts
	LogCapacity int
	EventBuffer int
}

// Manager serializes every state-changing operation and publishes state events.
type Manager struct {
	factory  kernel.Factory
	store    kernel.ConfigStore
	open     kernel.OpenOptions
	timeouts Timeouts
	events   *events.Bus
	logs     *logbuf.Buffer

	opMu      sync.Mutex
	stateMu   sync.RWMutex
	closed    bool
	state     kernel.State
	engine    kernel.Engine
	config    kernel.ConfigDocument
	hasConfig bool
	startedAt time.Time
	lastError *kernel.Error
	version   string
	revision  atomic.Uint64
}

var _ kernel.Manager = (*Manager)(nil)

// New loads the current configuration and constructs an idle manager. The
// engine itself is opened only when Start is called.
func New(ctx context.Context, opts Options) (*Manager, error) {
	if opts.Factory == nil {
		return nil, kernel.NewError(kernel.ErrCodeInternal, "manager.new", "engine factory is required")
	}
	if opts.Store == nil {
		return nil, kernel.NewError(kernel.ErrCodeInternal, "manager.new", "config store is required")
	}
	timeouts := normalizeTimeouts(opts.Timeouts)
	if opts.LogCapacity <= 0 {
		opts.LogCapacity = defaultLogCapacity
	}
	if opts.EventBuffer <= 0 {
		opts.EventBuffer = defaultEventBuffer
	}

	manager := &Manager{
		factory:  opts.Factory,
		store:    opts.Store,
		open:     opts.Open,
		timeouts: timeouts,
		events:   events.New(opts.EventBuffer),
		logs:     logbuf.New(opts.LogCapacity),
		state:    kernel.StateStopped,
	}
	manager.open.LogWriter = manager.logs
	manager.open.EventSink = manager.events

	document, err := opts.Store.Load(ctx)
	if err != nil && !errors.Is(err, kernel.ErrConfigNotFound) {
		return nil, kernel.WrapError(kernel.ErrCodeInternal, "manager.new", "load current configuration", err)
	}
	if err == nil {
		manager.config = cloneDocument(document)
		manager.hasConfig = true
	}
	return manager, nil
}

// Kind implements kernel.EngineInfo.
func (m *Manager) Kind() kernel.EngineKind {
	return m.factory.Kind()
}

// Version reports the last engine version observed by this manager.
func (m *Manager) Version() string {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.version
}

// Capabilities reports the capabilities of the currently open engine.
func (m *Manager) Capabilities() kernel.Capabilities {
	m.stateMu.RLock()
	engine := m.engine
	m.stateMu.RUnlock()
	if engine == nil {
		return kernel.Capabilities{}
	}
	return engine.Capabilities()
}

// Status returns an immutable snapshot without taking the operation lock.
func (m *Manager) Status() kernel.Status {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.statusLocked()
}

func (m *Manager) statusLocked() kernel.Status {
	return kernel.Status{
		Kind:      m.Kind(),
		State:     m.state,
		Version:   m.version,
		StartedAt: m.startedAt,
		Revision:  m.config.Revision,
		LastError: cloneError(m.lastError),
	}
}

func (m *Manager) transition(state kernel.State, lastError *kernel.Error) {
	m.stateMu.Lock()
	m.state = state
	m.lastError = lastError
	if state == kernel.StateRunning {
		m.startedAt = time.Now().UTC()
	}
	if state == kernel.StateStopped {
		m.startedAt = time.Time{}
	}
	payload := kernel.StateChanged{
		State:     state,
		Revision:  m.config.Revision,
		StartedAt: m.startedAt,
		LastError: cloneError(lastError),
	}
	m.stateMu.Unlock()
	m.events.Publish(kernel.Event{Type: kernel.EventKernelState, Payload: payload})
}

func (m *Manager) setConfig(document kernel.ConfigDocument, exists bool) {
	m.stateMu.Lock()
	m.config = cloneDocument(document)
	m.hasConfig = exists
	m.stateMu.Unlock()
}

func (m *Manager) currentConfig() (kernel.ConfigDocument, bool) {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return cloneDocument(m.config), m.hasConfig
}

func (m *Manager) currentEngine() (kernel.Engine, kernel.State) {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.engine, m.state
}

func (m *Manager) nextRevision() string {
	return fmt.Sprintf("r-%d-%d", time.Now().UTC().UnixMilli(), m.revision.Add(1))
}

func normalizeTimeouts(timeouts Timeouts) Timeouts {
	defaults := DefaultTimeouts()
	if timeouts.Start <= 0 {
		timeouts.Start = defaults.Start
	}
	if timeouts.Stop <= 0 {
		timeouts.Stop = defaults.Stop
	}
	if timeouts.Apply <= 0 {
		timeouts.Apply = defaults.Apply
	}
	return timeouts
}

func cloneDocument(document kernel.ConfigDocument) kernel.ConfigDocument {
	document.Body = append([]byte(nil), document.Body...)
	return document
}

func cloneError(err *kernel.Error) *kernel.Error {
	if err == nil {
		return nil
	}
	copy := *err
	return &copy
}

func mapOperationError(code kernel.ErrorCode, op, message string, err error) *kernel.Error {
	if err == nil {
		return nil
	}
	var kernelError *kernel.Error
	if errors.As(err, &kernelError) {
		return kernelError
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return kernel.WrapError(kernel.ErrCodeTimeout, op, message+" timed out", err)
	}
	if errors.Is(err, context.Canceled) {
		return kernel.WrapError(kernel.ErrCodeCanceled, op, message+" canceled", err)
	}
	return kernel.WrapError(code, op, message, err)
}
