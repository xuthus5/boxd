package main

import (
	"context"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/service"
)

func (b *runtimeBackend) PreviewSetup(ctx context.Context, request core.SetupRequest) (*core.SetupPlan, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return nil, err
	}
	return svc.Config().PreviewSetup(ctx, request)
}

func (b *runtimeBackend) ApplySetup(ctx context.Context, request core.SetupRequest) (service.SetupApplyResult, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return service.SetupApplyResult{}, err
	}
	return svc.Config().ApplySetup(ctx, request)
}
