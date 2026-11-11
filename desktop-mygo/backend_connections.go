package main

import (
	"context"

	"github.com/xuthus5/boxd/internal/service"
)

func (b *runtimeBackend) CloseConnections(ctx context.Context, filters service.ConnectionCloseFilters) (map[string]any, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return nil, err
	}
	return svc.Stats().CloseConnections(ctx, filters)
}
