package singbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"

	"github.com/xuthus5/boxd/kernel"
)

// Engine is the first production sing-box driver. Configuration reloads use a
// restart because sing-box does not expose a stable, engine-neutral hot reload
// contract for every component in v1.14.x.
type Engine struct {
	mu sync.Mutex

	name      string
	workDir   string
	config    kernel.ConfigDocument
	options   option.Options
	instance  *box.Box
	cancel    context.CancelFunc
	logWriter *platformLogWriter
	runtime   engineRuntime

	state     kernel.State
	startedAt time.Time
	lastError *kernel.Error
	closed    bool
	version   string
}

var _ kernel.Engine = (*Engine)(nil)
var _ kernel.ConfigDiagnosticsFeature = (*Engine)(nil)

func newEngine(openOptions kernel.OpenOptions, options option.Options) *Engine {
	return &Engine{
		name:      openOptions.Name,
		workDir:   openOptions.WorkDir,
		config:    cloneDocument(openOptions.Config),
		options:   options,
		logWriter: newPlatformLogWriter(openOptions.LogWriter, openOptions.EventSink),
		state:     kernel.StateStopped,
		version:   CurrentVersion().EffectiveVersion,
	}
}

func (e *Engine) Kind() kernel.EngineKind { return kernel.EngineSingBox }

func (e *Engine) Version() string {
	if e.version == "" {
		return CurrentVersion().EffectiveVersion
	}
	return e.version
}

func (e *Engine) Capabilities() kernel.Capabilities {
	return kernel.Capabilities{
		HotReload:          false,
		ClashMode:          true,
		DNSFlush:           true,
		ConnectionControl:  true,
		TrafficObservation: true,
		ConfigDiagnostics:  true,
	}
}

func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return kernel.NewError(kernel.ErrCodeInvalidState, "driver.singbox.start", "engine is closed")
	}
	if e.state == kernel.StateRunning {
		return nil
	}
	return e.startLocked(ctx)
}

func (e *Engine) startLocked(ctx context.Context) error {
	e.state = kernel.StateStarting
	e.lastError = nil

	lifecycleContext, cancel := context.WithCancel(include.Context(context.WithoutCancel(ctx)))
	instance, err := box.New(box.Options{
		Context:           lifecycleContext,
		Options:           e.options,
		PlatformLogWriter: e.logWriter,
	})
	if err != nil {
		cancel()
		return e.failLocked(kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.start", "create sing-box instance", err))
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		cancel()
		return e.failLocked(kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.start", "start sing-box instance", err))
	}

	e.instance = instance
	e.cancel = cancel
	e.runtime = bindEngineRuntime(lifecycleContext)
	e.state = kernel.StateRunning
	e.startedAt = time.Now().UTC()
	return nil
}

func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	return e.stopLocked(ctx)
}

func (e *Engine) stopLocked(ctx context.Context) error {
	if e.state == kernel.StateStopped && e.instance == nil {
		return nil
	}
	e.state = kernel.StateStopping
	instance := e.instance
	cancel := e.cancel
	e.instance = nil
	e.cancel = nil
	e.runtime = engineRuntime{}

	var closeErr error
	if instance != nil {
		closeErr = instance.Close()
	}
	if cancel != nil {
		cancel()
	}
	if closeErr != nil {
		return e.failLocked(kernel.WrapError(kernel.ErrCodeInternal, "driver.singbox.stop", "close sing-box instance", closeErr))
	}
	e.state = kernel.StateStopped
	e.startedAt = time.Time{}
	e.lastError = nil
	return nil
}

func (e *Engine) Restart(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return kernel.NewError(kernel.ErrCodeInvalidState, "driver.singbox.restart", "engine is closed")
	}
	if err := e.stopLocked(ctx); err != nil {
		return err
	}
	return e.startLocked(ctx)
}

func (e *Engine) Status() kernel.Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return kernel.Status{
		Kind:      kernel.EngineSingBox,
		State:     e.state,
		Version:   e.Version(),
		StartedAt: e.startedAt,
		Revision:  e.config.Revision,
		LastError: cloneError(e.lastError),
	}
}

func (e *Engine) ReadConfig(context.Context) (kernel.ConfigDocument, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return cloneDocument(e.config), nil
}

func (e *Engine) ValidateConfig(ctx context.Context, document kernel.ConfigDocument) (kernel.ValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return kernel.ValidationResult{}, err
	}
	if _, err := parseOptions(ctx, document, e.workDir); err != nil {
		return kernel.ValidationResult{
			Valid: false,
			Diagnostics: []kernel.Diagnostic{{
				Severity: "error",
				Code:     "singbox.invalid_config",
				Path:     "config",
				Message:  err.Error(),
				Hint:     "check the sing-box JSON schema and protocol fields",
			}},
		}, nil
	}
	return kernel.ValidationResult{Valid: true}, nil
}

// DiagnoseConfig implements kernel.ConfigDiagnosticsFeature without exposing
// sing-box option types to the control plane.
func (e *Engine) DiagnoseConfig(ctx context.Context, document kernel.ConfigDocument) ([]kernel.Diagnostic, error) {
	result, err := e.ValidateConfig(ctx, document)
	if err != nil {
		return nil, err
	}
	return append([]kernel.Diagnostic(nil), result.Diagnostics...), nil
}

func (e *Engine) ApplyConfig(ctx context.Context, document kernel.ConfigDocument, _ kernel.ApplyOptions) (kernel.ApplyResult, error) {
	parsed, err := parseOptions(ctx, document, e.workDir)
	if err != nil {
		return kernel.ApplyResult{}, kernel.WrapError(kernel.ErrCodeInvalidConfig, "driver.singbox.apply", "parse candidate configuration", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return kernel.ApplyResult{}, kernel.NewError(kernel.ErrCodeInvalidState, "driver.singbox.apply", "engine is closed")
	}
	previousConfig := cloneDocument(e.config)
	previousOptions := e.options
	e.config = cloneDocument(document)
	e.options = parsed

	if e.state != kernel.StateRunning {
		return kernel.ApplyResult{Revision: document.Revision}, nil
	}
	if err := e.stopLocked(ctx); err != nil {
		e.config = previousConfig
		e.options = previousOptions
		return kernel.ApplyResult{}, kernel.WrapError(kernel.ErrCodeApplyFailed, "driver.singbox.apply", "stop current instance", err)
	}
	if err := e.startLocked(ctx); err != nil {
		e.config = previousConfig
		e.options = previousOptions
		return kernel.ApplyResult{}, kernel.WrapError(kernel.ErrCodeApplyFailed, "driver.singbox.apply", "restart with candidate configuration", err)
	}
	return kernel.ApplyResult{Revision: document.Revision, Restarted: true}, nil
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	err := e.stopLocked(context.Background())
	e.closed = true
	return err
}

func (e *Engine) failLocked(err *kernel.Error) error {
	e.state = kernel.StateFailed
	e.lastError = cloneError(err)
	e.startedAt = time.Time{}
	return err
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

func (e *Engine) String() string {
	return fmt.Sprintf("sing-box %s (%s)", e.Version(), e.Status().State)
}
