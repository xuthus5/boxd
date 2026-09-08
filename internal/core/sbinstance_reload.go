package core

// Reload 仅重载运行中的内核，停止态保持不变；状态检查与启停共用同一把锁。
func (s *SBInstance) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	if err := s.stopLocked(); err != nil {
		return err
	}
	return s.startLocked()
}

// ReloadConfig 优先使用保持运行状态的重载能力，兼容仅实现 Restart 的实例。
func ReloadConfig(instance interface{ Restart() error }) error {
	if instance == nil {
		return nil
	}
	if reloader, ok := instance.(interface{ Reload() error }); ok {
		return reloader.Reload()
	}
	return instance.Restart()
}
