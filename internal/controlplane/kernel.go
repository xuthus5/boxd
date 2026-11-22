package controlplane

import (
	"context"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// KernelService adapts the kernel manager to the public kernel API.
type KernelService struct {
	manager kernel.Manager
}

var _ control.KernelAPI = (*KernelService)(nil)

// NewKernelService constructs a kernel use-case service.
func NewKernelService(manager kernel.Manager) *KernelService {
	return &KernelService{manager: manager}
}

func (s *KernelService) Status(context.Context) (kernel.Status, error) {
	return s.manager.Status(), nil
}

func (s *KernelService) Start(ctx context.Context) error {
	return s.manager.Start(ctx)
}

func (s *KernelService) Stop(ctx context.Context) error {
	return s.manager.Stop(ctx)
}

func (s *KernelService) Restart(ctx context.Context) error {
	return s.manager.Restart(ctx)
}
