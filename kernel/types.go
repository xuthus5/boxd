// Package kernel defines the stable boundary between boxd and a network engine.
package kernel

import "time"

// EngineKind identifies an engine implementation.
type EngineKind string

const (
	EngineSingBox EngineKind = "sing-box"
)

// State is the externally visible state of the kernel manager.
type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateFailed   State = "failed"
)

// Status is an immutable snapshot of the kernel manager.
type Status struct {
	Kind      EngineKind
	State     State
	Version   string
	StartedAt time.Time
	Revision  string
	LastError *Error
}

// ConfigDocument is an opaque engine configuration owned by the control plane.
type ConfigDocument struct {
	Format   string
	Body     []byte
	Revision string
}

// Diagnostic describes a configuration or runtime problem without leaking
// engine-specific types.
type Diagnostic struct {
	Severity string
	Code     string
	Path     string
	Message  string
	Hint     string
}

// ValidationResult is returned by an engine before a configuration is saved.
type ValidationResult struct {
	Valid       bool
	Diagnostics []Diagnostic
	Normalized  []byte
}

// ApplyOptions controls how a candidate configuration is applied.
type ApplyOptions struct {
	Source           string
	ExpectedRevision string
	RestartIfRunning bool
}

// ApplyResult reports the effective revision and restart behavior.
type ApplyResult struct {
	Revision  string
	Restarted bool
	Warnings  []Diagnostic
}

// Capabilities records optional engine features.
type Capabilities struct {
	HotReload          bool
	ClashMode          bool
	DNSFlush           bool
	ConnectionControl  bool
	TrafficObservation bool
	ConfigDiagnostics  bool
}

// OutboundGroup is an engine-neutral view of a selector or URL-test group.
type OutboundGroup struct {
	Type     string
	Tag      string
	Selected string
	Members  []string
}

// ClashMode describes the current and available compatibility modes.
type ClashMode struct {
	Current   string
	Available []string
}

// Connection is an engine-neutral connection snapshot.
type Connection struct {
	ID        int64
	Inbound   string
	Network   string
	Source    string
	Target    string
	Outbound  string
	Rule      string
	RuleIndex int
	Process   string
	Upload    int64
	Download  int64
	StartedAt time.Time
}

// ConnectionFilter limits bulk connection operations.
type ConnectionFilter struct {
	IDs      []int64
	Outbound string
	Rule     string
	Process  string
}

// LogEntry is a structured, engine-neutral log record.
type LogEntry struct {
	Time    time.Time
	Level   string
	Source  string
	Message string
	Fields  map[string]any
}

// TrafficSnapshot contains cumulative and current traffic measurements.
type TrafficSnapshot struct {
	UploadTotal   int64
	DownloadTotal int64
	UploadRate    int64
	DownloadRate  int64
	At            time.Time
}

// TrafficConnection combines a connection with real-time rates.
type TrafficConnection struct {
	Connection   Connection
	UploadRate   int64
	DownloadRate int64
}

// EventType classifies asynchronous kernel and application events.
type EventType string

const (
	EventKernelState      EventType = "kernel.state"
	EventKernelLog        EventType = "kernel.log"
	EventKernelTraffic    EventType = "kernel.traffic"
	EventKernelConnection EventType = "kernel.connection"
	EventConfigApplied    EventType = "config.applied"
	EventControlChanged   EventType = "control.changed"
)

// Event is delivered through a bounded, non-blocking event bus.
type Event struct {
	Type     EventType
	Sequence uint64
	Time     time.Time
	Payload  any
}

// StateChanged is the payload of EventKernelState.
type StateChanged struct {
	State     State
	Revision  string
	StartedAt time.Time
	LastError *Error
}

// ConfigApplied is the payload of EventConfigApplied.
type ConfigApplied struct {
	Revision         string
	PreviousRevision string
	Source           string
	Restarted        bool
	RolledBack       bool
	Error            *Error
}

// ConfigRevision is a compact entry in configuration history.
type ConfigRevision struct {
	Revision  string
	Source    string
	CreatedAt time.Time
	Summary   string
}

// NodeSpec is a control-plane node intent. It deliberately contains no engine
// configuration structures.
type NodeSpec struct {
	ID       string
	Tag      string
	Protocol string
	Server   string
	Port     uint16
	Metadata map[string]string
}

// EngineDescriptor describes a loaded engine without exposing implementation types.
type EngineDescriptor struct {
	Kind         EngineKind
	Version      string
	Capabilities Capabilities
}
