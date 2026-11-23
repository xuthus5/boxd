package control

import (
	"context"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

// SettingsRepository persists application preferences. Implementations live in
// the outbound adapter layer.
type SettingsRepository interface {
	List(ctx context.Context) ([]Setting, error)
	Get(ctx context.Context, key string) (Setting, bool, error)
	Upsert(ctx context.Context, setting Setting) error
}

// ProfileRepository persists the managed desired state as one atomic snapshot.
// Implementations may normalize it internally, but callers only observe a
// consistent snapshot.
type ProfileRepository interface {
	Load(ctx context.Context) (DesiredState, error)
	Save(ctx context.Context, state DesiredState) error
}

// ConfigCompiler translates the engine-neutral desired state into an opaque
// kernel configuration document. Engine-specific syntax stays behind this port.
type ConfigCompiler interface {
	Compile(ctx context.Context, state DesiredState) (kernel.ConfigDocument, error)
}

// DownloadRequest describes a conditional HTTP download. Empty validators
// perform an unconditional request.
type DownloadRequest struct {
	URL          string
	ETag         string
	LastModified string
	Timeout      time.Duration
	MaxBytes     int64
}

// DownloadResult is a bounded response body plus cache validators. NotModified
// is true for a successful 304 response and Body is empty in that case.
type DownloadResult struct {
	URL          string
	Body         []byte
	ContentType  string
	ETag         string
	LastModified string
	NotModified  bool
}

// Downloader fetches remote subscriptions and rule sets. Implementations live
// in an outbound adapter and enforce timeouts and response-size limits.
type Downloader interface {
	Get(ctx context.Context, request DownloadRequest) (DownloadResult, error)
}

// NodeCodec converts engine-neutral nodes to and from share links. Link
// syntax is transport-neutral and must not leak engine configuration types.
type NodeCodec interface {
	Decode(ctx context.Context, link string) (Node, error)
	Encode(ctx context.Context, node Node) (string, error)
}

// SubscriptionParser converts a downloaded body into engine-neutral nodes.
type SubscriptionParser interface {
	Parse(ctx context.Context, result DownloadResult) ([]Node, error)
}

// RuleSetRepository persists refresh metadata independently from desired
// state so status updates do not trigger configuration reapplication.
type RuleSetRepository interface {
	List(ctx context.Context) ([]RuleSetStatus, error)
	Get(ctx context.Context, name string) (RuleSetStatus, bool, error)
	Upsert(ctx context.Context, status RuleSetStatus) error
	Delete(ctx context.Context, name string) error
}

// TestResultRepository persists bounded node latency measurements.
type TestResultRepository interface {
	Save(ctx context.Context, result NodeTestResult) error
	List(ctx context.Context, nodeID string, limit int) ([]NodeTestResult, error)
}

// SecurityCredential is a salted password verifier. Plaintext passwords never
// cross this port.
type SecurityCredential struct {
	Salt       []byte
	Hash       []byte
	Iterations int
	UpdatedAt  time.Time
}

// SecurityRepository persists the single local administrator credential.
type SecurityRepository interface {
	Load(ctx context.Context) (SecurityCredential, bool, error)
	Save(ctx context.Context, credential SecurityCredential) error
}

// AuditRepository persists immutable control-plane audit events.
type AuditRepository interface {
	Append(ctx context.Context, event AuditEvent) (int64, error)
	List(ctx context.Context, filter AuditFilter) ([]AuditEvent, error)
}
