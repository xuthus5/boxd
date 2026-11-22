package kernel

import "context"

// ConnectionController exposes live connection inspection and control.
type ConnectionController interface {
	ListConnections(ctx context.Context) ([]Connection, error)
	CloseConnection(ctx context.Context, id int64) error
	CloseConnections(ctx context.Context, filter ConnectionFilter) (int, error)
}
