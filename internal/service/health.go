package service

import "context"

// HealthService 提供健康检查用例逻辑。
type HealthService struct {
	ready func() error
}

// NewHealthService 构造健康检查用例服务。
func NewHealthService(ready func() error) *HealthService {
	return &HealthService{ready: ready}
}

// Liveness 存活检查始终通过。
func (s *HealthService) Liveness(_ context.Context) (map[string]string, error) {
	return map[string]string{"status": "ok"}, nil
}

// Readiness 就绪检查依赖底层依赖可用性。
func (s *HealthService) Readiness(_ context.Context) (map[string]string, error) {
	if s.ready != nil {
		if err := s.ready(); err != nil {
			return nil, Errorf(503, "not_ready", "service is not ready")
		}
	}
	return map[string]string{"status": "ready"}, nil
}
