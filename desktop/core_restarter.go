package main

import "github.com/xuthus5/boxd/internal/core"

// coreRestarter 保留内核的显式重启和按运行状态重载能力。
type coreRestarter struct {
	instance *core.SBInstance
}

func (r coreRestarter) Restart() error {
	if r.instance == nil {
		return nil
	}
	return r.instance.Restart()
}

func (r coreRestarter) Reload() error {
	if r.instance == nil {
		return nil
	}
	return r.instance.Reload()
}
