package controlplane

import (
	"context"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// SettingsService implements preference reads and writes.
type SettingsService struct {
	repository control.SettingsRepository
	audit      control.AuditRecorder
	now        func() time.Time
}

func (s *SettingsService) SetAuditor(recorder control.AuditRecorder) { s.audit = recorder }

var _ control.SettingsAPI = (*SettingsService)(nil)

// NewSettingsService constructs a settings use-case service.
func NewSettingsService(repository control.SettingsRepository) *SettingsService {
	return &SettingsService{repository: repository, now: time.Now}
}

func (s *SettingsService) List(ctx context.Context) ([]control.Setting, error) {
	return s.repository.List(ctx)
}

func (s *SettingsService) Get(ctx context.Context, key string) (control.Setting, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return control.Setting{}, false, kernel.NewError(kernel.ErrCodeInvalidConfig, "settings.get", "setting key is required")
	}
	return s.repository.Get(ctx, key)
}

func (s *SettingsService) Set(ctx context.Context, key, value string) (control.Setting, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return control.Setting{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "settings.set", "setting key is required")
	}
	setting := control.Setting{Key: key, Value: value, UpdatedAt: s.now().UTC()}
	if err := s.repository.Upsert(ctx, setting); err != nil {
		return control.Setting{}, err
	}
	recordAudit(ctx, s.audit, "settings.set", map[string]any{"key": setting.Key})
	return setting, nil
}
