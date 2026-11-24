// Package fake provides a deterministic engine for contract and control-plane tests.
package fake

import (
	"context"
	"sync"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

// Engine is a configurable in-memory implementation of kernel.Engine.
type Engine struct {
	Descriptor  kernel.EngineDescriptor
	StartErr    error
	StopErr     error
	CloseErr    error
	ValidateErr error
	ApplyErr    error

	Validation        kernel.ValidationResult
	Apply             kernel.ApplyResult
	Groups            []kernel.OutboundGroup
	Mode              kernel.ClashMode
	Connections       []kernel.Connection
	ClosedConnections []kernel.Connection
	Traffic           kernel.TrafficSnapshot

	mu         sync.RWMutex
	state      kernel.State
	config     kernel.ConfigDocument
	startedAt  time.Time
	closed     bool
	startCalls int
	stopCalls  int
	applyCalls int
}

var _ kernel.Engine = (*Engine)(nil)

// New returns a stopped fake engine with valid default validation.
func New(document kernel.ConfigDocument) *Engine {
	return &Engine{
		Descriptor: kernel.EngineDescriptor{
			Kind:    kernel.EngineSingBox,
			Version: "fake-1",
		},
		Validation: kernel.ValidationResult{Valid: true},
		config:     document,
		state:      kernel.StateStopped,
	}
}

func (e *Engine) Kind() kernel.EngineKind           { return e.Descriptor.Kind }
func (e *Engine) Version() string                   { return e.Descriptor.Version }
func (e *Engine) Capabilities() kernel.Capabilities { return e.Descriptor.Capabilities }

func (e *Engine) Start(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.startCalls++
	if e.StartErr != nil {
		e.state = kernel.StateFailed
		return e.StartErr
	}
	e.state = kernel.StateRunning
	e.startedAt = time.Now().UTC()
	return nil
}

func (e *Engine) Stop(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopCalls++
	if e.StopErr != nil {
		return e.StopErr
	}
	e.state = kernel.StateStopped
	e.startedAt = time.Time{}
	return nil
}

func (e *Engine) Restart(ctx context.Context) error {
	if err := e.Stop(ctx); err != nil {
		return err
	}
	return e.Start(ctx)
}

func (e *Engine) Status() kernel.Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return kernel.Status{
		Kind:      e.Descriptor.Kind,
		State:     e.state,
		Version:   e.Descriptor.Version,
		StartedAt: e.startedAt,
		Revision:  e.config.Revision,
	}
}

func (e *Engine) ReadConfig(context.Context) (kernel.ConfigDocument, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config, nil
}

func (e *Engine) ValidateConfig(context.Context, kernel.ConfigDocument) (kernel.ValidationResult, error) {
	return e.Validation, e.ValidateErr
}

func (e *Engine) ApplyConfig(_ context.Context, document kernel.ConfigDocument, _ kernel.ApplyOptions) (kernel.ApplyResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.applyCalls++
	if e.ApplyErr != nil {
		return kernel.ApplyResult{}, e.ApplyErr
	}
	e.config = document
	result := e.Apply
	if result.Revision == "" {
		result.Revision = document.Revision
	}
	return result, nil
}

func (e *Engine) OutboundGroups(context.Context) ([]kernel.OutboundGroup, error) {
	return append([]kernel.OutboundGroup(nil), e.Groups...), nil
}

func (e *Engine) SelectOutbound(context.Context, string, string) error { return nil }
func (e *Engine) OutboundDelay(context.Context, string, string) (time.Duration, error) {
	return 0, nil
}
func (e *Engine) ClashMode(context.Context) (kernel.ClashMode, error) { return e.Mode, nil }
func (e *Engine) SetClashMode(context.Context, string) error          { return nil }
func (e *Engine) FlushDNS(context.Context) error                      { return nil }

func (e *Engine) ListConnections(context.Context) ([]kernel.Connection, error) {
	return append([]kernel.Connection(nil), e.Connections...), nil
}
func (e *Engine) ListClosedConnections(context.Context) ([]kernel.Connection, error) {
	return append([]kernel.Connection(nil), e.ClosedConnections...), nil
}
func (e *Engine) CloseConnection(context.Context, int64) error { return nil }
func (e *Engine) CloseConnections(context.Context, kernel.ConnectionFilter) (int, error) {
	return 0, nil
}

func (e *Engine) RecentLogs(int) []kernel.LogEntry        { return nil }
func (e *Engine) TrafficSnapshot() kernel.TrafficSnapshot { return e.Traffic }

func (e *Engine) Subscribe(ctx context.Context, _ ...kernel.EventType) (<-chan kernel.Event, error) {
	channel := make(chan kernel.Event)
	go func() {
		<-ctx.Done()
		close(channel)
	}()
	return channel, nil
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	return e.CloseErr
}

// Counts returns call counters for assertions.
func (e *Engine) Counts() (start, stop, apply int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.startCalls, e.stopCalls, e.applyCalls
}

// Closed reports whether Close was called.
func (e *Engine) Closed() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.closed
}

// Factory creates fresh fake engines and records open options.
type Factory struct {
	mu        sync.Mutex
	OpenErr   error
	NewEngine func(kernel.OpenOptions) (*Engine, error)
	opened    []*Engine
	options   []kernel.OpenOptions
}

var _ kernel.Factory = (*Factory)(nil)

// NewFactory returns a factory that creates default fake engines.
func NewFactory() *Factory {
	return &Factory{
		NewEngine: func(options kernel.OpenOptions) (*Engine, error) {
			return New(options.Config), nil
		},
	}
}

func (f *Factory) Kind() kernel.EngineKind { return kernel.EngineSingBox }

func (f *Factory) Open(ctx context.Context, options kernel.OpenOptions) (kernel.Engine, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.OpenErr != nil {
		return nil, f.OpenErr
	}
	engine, err := f.NewEngine(options)
	if err != nil {
		return nil, err
	}
	f.opened = append(f.opened, engine)
	f.options = append(f.options, options)
	return engine, nil
}

// Opened returns a copy of all engines created by this factory.
func (f *Factory) Opened() []*Engine {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*Engine(nil), f.opened...)
}
