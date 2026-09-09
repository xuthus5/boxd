package core

import (
	"context"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/service/pause"
)

type networkStartupUpdater interface {
	UpdateInterfaces() error
	UpdateWIFIState(context.Context)
	ResetNetwork(context.Context)
}

type networkStartupState struct {
	ctx     context.Context
	network networkStartupUpdater
	manager pause.Manager
	logger  log.ContextLogger
}

func (s networkStartupState) apply(networkInterface *control.Interface, _ int) {
	_ = s.refresh(networkInterface)
}

func (s networkStartupState) update(ctx context.Context, networkInterface *control.Interface, _ int) {
	s.ctx = ctx
	if s.refresh(networkInterface) {
		s.network.ResetNetwork(ctx)
	}
}

func (s networkStartupState) refresh(networkInterface *control.Interface) bool {
	if s.ctx.Err() != nil {
		return false
	}
	if networkInterface == nil {
		s.manager.NetworkPause()
		s.logger.Error("missing default interface")
		return false
	}
	s.manager.NetworkWake()
	s.logger.Info("updated default interface ", networkInterface.Name, ", index ", networkInterface.Index)
	s.network.UpdateWIFIState(s.ctx)
	if s.ctx.Err() != nil {
		return false
	}
	if err := s.network.UpdateInterfaces(); err != nil {
		s.logger.Warn("update network interfaces: ", err)
	}
	// 上游 UpdateInterfaces 会更新 IPv4/IPv6 接口并 defer updateNetworkEnvironment。
	return s.ctx.Err() == nil
}
