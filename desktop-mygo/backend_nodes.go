package main

import (
	"context"
	"errors"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

func (b *runtimeBackend) ImportNode(ctx context.Context, link string) (model.ImportResult, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return model.ImportResult{}, err
	}
	result, err := svc.Import().ParseLink(ctx, link)
	if err != nil {
		return model.ImportResult{}, err
	}
	err = svc.Import().SaveNode(ctx, service.NodeInput{
		Tag: result.Tag, Type: result.Type, Server: result.Server, Port: result.Port, Config: result.Config,
	})
	if err != nil {
		return model.ImportResult{}, err
	}
	return *result, nil
}

func (b *runtimeBackend) SaveNode(ctx context.Context, input service.NodeInput) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Import().SaveNode(ctx, input)
}

func (b *runtimeBackend) UpdateNode(ctx context.Context, oldTag string, input service.NodeInput) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	if svc.Deps.NodeManager == nil || svc.Deps.NodeManager.Get(oldTag) == nil {
		return errors.New("imported node not found")
	}
	if err := svc.Deps.NodeManager.Update(oldTag, nodeOutbound(input)); err != nil {
		return err
	}
	return syncNodes(ctx, svc)
}

func (b *runtimeBackend) DeleteNode(ctx context.Context, tag string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	if svc.Deps.NodeManager == nil || svc.Deps.NodeManager.Get(tag) == nil {
		return errors.New("imported node not found")
	}
	if err := svc.Deps.NodeManager.Delete(tag); err != nil {
		return err
	}
	return syncNodes(ctx, svc)
}

func (b *runtimeBackend) TestNode(ctx context.Context, req service.TestRequest) (model.TestResult, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return model.TestResult{}, err
	}
	return svc.Test().Run(ctx, req)
}

func (b *runtimeBackend) TestNodes(ctx context.Context, req service.TestBatchRequest) ([]model.TestResult, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return nil, err
	}
	return svc.Test().RunBatch(ctx, req)
}

func nodeOutbound(input service.NodeInput) model.Outbound {
	return model.Outbound{Tag: input.Tag, Type: input.Type, Server: input.Server, Port: input.Port, Raw: input.Config}
}

func syncNodes(ctx context.Context, svc *service.ServiceSet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return service.SyncOutboundsAndRestart(svc.Deps.NodeManager, svc.Deps.SubManager, svc.Deps.ConfigPath, svc.Deps.Instance)
}
