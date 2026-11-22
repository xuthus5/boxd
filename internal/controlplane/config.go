package controlplane

import (
	"context"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// ConfigService owns configuration use cases without exposing storage details.
type ConfigService struct {
	manager kernel.Manager
	store   kernel.ConfigStore
	profile *ProfileService
}

var _ control.ConfigAPI = (*ConfigService)(nil)

// NewConfigService constructs a configuration use-case service.
func NewConfigService(manager kernel.Manager, store kernel.ConfigStore, profile *ProfileService) *ConfigService {
	return &ConfigService{manager: manager, store: store, profile: profile}
}

func (s *ConfigService) Read(ctx context.Context) (kernel.ConfigDocument, error) {
	return s.manager.ReadConfig(ctx)
}

func (s *ConfigService) Validate(ctx context.Context, document kernel.ConfigDocument) (kernel.ValidationResult, error) {
	return s.manager.ValidateConfig(ctx, document)
}

func (s *ConfigService) Apply(ctx context.Context, document kernel.ConfigDocument, options kernel.ApplyOptions) (kernel.ApplyResult, error) {
	return s.manager.ApplyConfig(ctx, document, options)
}

func (s *ConfigService) Reconcile(ctx context.Context, options kernel.ApplyOptions) (kernel.ApplyResult, error) {
	if s.profile == nil {
		return kernel.ApplyResult{}, kernel.NewError(kernel.ErrCodeInternal, "config.reconcile", "profile service is unavailable")
	}
	return s.profile.Reconcile(ctx, options)
}

func (s *ConfigService) History(ctx context.Context, limit int) ([]kernel.ConfigRevision, error) {
	return s.store.History(ctx, limit)
}
