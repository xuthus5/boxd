package controlplane

import (
	"context"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// KernelService adapts the kernel manager to the public kernel API.
type KernelService struct {
	manager kernel.Manager
	audit   control.AuditRecorder
}

func (s *KernelService) SetAuditor(recorder control.AuditRecorder) { s.audit = recorder }

var _ control.KernelAPI = (*KernelService)(nil)

// NewKernelService constructs a kernel use-case service.
func NewKernelService(manager kernel.Manager) *KernelService {
	return &KernelService{manager: manager}
}

func (s *KernelService) Status(context.Context) (kernel.Status, error) {
	return s.manager.Status(), nil
}

func (s *KernelService) Start(ctx context.Context) error {
	if err := s.manager.Start(ctx); err != nil {
		return err
	}
	recordAudit(ctx, s.audit, "kernel.start", nil)
	return nil
}

func (s *KernelService) Stop(ctx context.Context) error {
	if err := s.manager.Stop(ctx); err != nil {
		return err
	}
	recordAudit(ctx, s.audit, "kernel.stop", nil)
	return nil
}

func (s *KernelService) Restart(ctx context.Context) error {
	if err := s.manager.Restart(ctx); err != nil {
		return err
	}
	recordAudit(ctx, s.audit, "kernel.restart", nil)
	return nil
}
