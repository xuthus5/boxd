package service

import "github.com/xuthus5/boxd/internal/model"

func (a restartAdapter) Reload() error {
	if a.instance == nil {
		return nil
	}
	return a.instance.Reload()
}

func (a restartAdapter) Status() model.ServiceStatus {
	if a.instance == nil {
		return model.ServiceStatus{}
	}
	return a.instance.Status()
}
