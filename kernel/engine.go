package kernel

import "context"

// EngineInfo describes an engine implementation.
type EngineInfo interface {
	Kind() EngineKind
	Version() string
	Capabilities() Capabilities
}

// Engine is a single engine instance. Driver packages are the only place that
// may implement it with engine-specific types.
type Engine interface {
	EngineInfo
	Lifecycle
	ConfigController
	RuntimeController
	ConnectionController
	Observer
	Close() error
}

// OpenOptions contains everything a factory needs to open an engine instance.
type OpenOptions struct {
	Name      string
	WorkDir   string
	Config    ConfigDocument
	LogWriter LogSink
	EventSink EventSink
}

// Factory opens engine instances for the manager.
type Factory interface {
	Kind() EngineKind
	Open(ctx context.Context, opts OpenOptions) (Engine, error)
}

// Manager is the only kernel surface exposed to the control plane.
type Manager interface {
	EngineInfo
	Lifecycle
	ConfigController
	RuntimeController
	ConnectionController
	Observer
}
