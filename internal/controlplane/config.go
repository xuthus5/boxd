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
	audit   control.AuditRecorder
}

func (s *ConfigService) SetAuditor(recorder control.AuditRecorder) { s.audit = recorder }

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

func (s *ConfigService) Diagnose(ctx context.Context, document kernel.ConfigDocument) ([]kernel.Diagnostic, error) {
	if diagnoser, ok := s.manager.(kernel.ConfigDiagnosticsFeature); ok {
		return diagnoser.DiagnoseConfig(ctx, document)
	}
	result, err := s.manager.ValidateConfig(ctx, document)
	if err != nil {
		return nil, err
	}
	return append([]kernel.Diagnostic(nil), result.Diagnostics...), nil
}

func (s *ConfigService) Apply(ctx context.Context, document kernel.ConfigDocument, options kernel.ApplyOptions) (kernel.ApplyResult, error) {
	result, err := s.manager.ApplyConfig(ctx, document, options)
	if err == nil {
		recordAudit(ctx, s.audit, "config.apply", map[string]any{"revision": result.Revision, "source": options.Source})
	}
	return result, err
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
