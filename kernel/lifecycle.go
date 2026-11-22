package kernel

import "context"

// Lifecycle controls the engine process or embedded runtime.
type Lifecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	Status() Status
}
