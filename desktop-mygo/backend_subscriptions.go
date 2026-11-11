package main

import (
	"context"
	"fmt"

	"github.com/xuthus5/boxd/internal/model"
	"github.com/xuthus5/boxd/internal/service"
)

func (b *runtimeBackend) RefreshAllSubscriptions(ctx context.Context) (service.RefreshAllResult, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return service.RefreshAllResult{}, err
	}
	return svc.Subscriptions().RefreshAll(ctx)
}

func (b *runtimeBackend) CreateSubscription(ctx context.Context, input service.SubscriptionInput) (*model.Subscription, error) {
	svc, err := b.serviceSet()
	if err != nil {
		return nil, err
	}
	sub, err := svc.Subscriptions().Create(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := syncNodes(ctx, svc); err != nil {
		rollbackErr := svc.Subscriptions().Delete(ctx, sub.ID)
		if rollbackErr != nil {
			return nil, fmt.Errorf("synchronize subscription: %w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}
	return sub, nil
}

func (b *runtimeBackend) UpdateSubscription(ctx context.Context, id string, input service.SubscriptionInput) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	previous, err := svc.Subscriptions().Get(ctx, id)
	if err != nil {
		return err
	}
	if err := svc.Subscriptions().Update(ctx, id, input); err != nil {
		return err
	}
	if err := syncNodes(ctx, svc); err != nil {
		rollbackErr := svc.Subscriptions().Update(ctx, id, subscriptionInput(*previous))
		if rollbackErr != nil {
			return fmt.Errorf("synchronize subscription: %w; rollback failed: %v", err, rollbackErr)
		}
		return err
	}
	return nil
}

func (b *runtimeBackend) DeleteSubscription(ctx context.Context, id string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	if _, err := svc.Subscriptions().Get(ctx, id); err != nil {
		return err
	}
	if err := svc.Subscriptions().Delete(ctx, id); err != nil {
		return err
	}
	return syncNodes(ctx, svc)
}

func (b *runtimeBackend) SelectSubscription(ctx context.Context, id, tag string) error {
	svc, err := b.serviceSet()
	if err != nil {
		return err
	}
	return svc.Subscriptions().SetSelected(ctx, id, tag)
}

func subscriptionInput(sub model.Subscription) service.SubscriptionInput {
	return service.SubscriptionInput{
		Name: sub.Name, URL: sub.URL, IntervalMin: sub.IntervalMin, URLTest: sub.URLTest,
	}
}
