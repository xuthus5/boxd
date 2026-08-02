package service

import (
	"context"

	"github.com/xuthus5/boxd/internal/model"
)

// serviceControlInstance 抽象内核控制能力，便于测试注入。
type serviceControlInstance interface {
	Start() error
	Stop() error
	Restart() error
	Status() model.ServiceStatus
}

// ServiceControl 提供内核启动/停止/重启/状态用例逻辑。
type ServiceControl struct{ instance serviceControlInstance }

// ServiceStatus 返回内核运行状态。
func (s *ServiceControl) ServiceStatus(_ context.Context) (model.ServiceStatus, error) {
	if s.instance == nil {
		return model.ServiceStatus{}, nil
	}
	return s.instance.Status(), nil
}

// ServiceStart 启动内核。
func (s *ServiceControl) ServiceStart(_ context.Context) error {
	if s.instance == nil {
		return Errorf(500, model.ErrorInternal, "service is not available")
	}
	if err := s.instance.Start(); err != nil {
		return Errorf(500, model.ErrorInternal, "%v", err)
	}
	return nil
}

// ServiceStop 停止内核。
func (s *ServiceControl) ServiceStop(_ context.Context) error {
	if s.instance == nil {
		return Errorf(500, model.ErrorInternal, "service is not available")
	}
	if err := s.instance.Stop(); err != nil {
		return Errorf(500, model.ErrorInternal, "%v", err)
	}
	return nil
}

// ServiceRestart 重启内核。
func (s *ServiceControl) ServiceRestart(_ context.Context) error {
	if s.instance == nil {
		return Errorf(500, model.ErrorInternal, "service is not available")
	}
	if err := s.instance.Restart(); err != nil {
		return Errorf(500, model.ErrorInternal, "%v", err)
	}
	return nil
}
