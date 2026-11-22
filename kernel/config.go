package kernel

import (
	"context"
)

// ConfigController owns the opaque configuration applied to an engine.
type ConfigController interface {
	ReadConfig(ctx context.Context) (ConfigDocument, error)
	ValidateConfig(ctx context.Context, doc ConfigDocument) (ValidationResult, error)
	ApplyConfig(ctx context.Context, doc ConfigDocument, opts ApplyOptions) (ApplyResult, error)
}

// ConfigStore persists the current configuration and its immutable history.
type ConfigStore interface {
	Load(ctx context.Context) (ConfigDocument, error)
	SaveAtomic(ctx context.Context, doc ConfigDocument) error
	History(ctx context.Context, limit int) ([]ConfigRevision, error)
}

// ConfigRemover is an optional extension used to roll back the first persisted
// configuration when no previous document exists.
type ConfigRemover interface {
	DeleteCurrent(ctx context.Context) error
}
