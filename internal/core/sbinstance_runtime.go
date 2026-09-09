package core

import (
	"context"
	"errors"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
)

type realBox struct {
	box        *box.Box
	gate       *networkStartupGate
	updates    *networkUpdateWorker
	cancel     context.CancelFunc
	pidFiles   namespacePIDFiles
	namespaces adapter.NetworkNamespaceManager
}

func newRealBox(options box.Options) (realBox, error) {
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	options.Context = ctx
	options.NetworkNamespaceHolderArgs = networkNamespaceHolderArgs()
	namespaces, pidFiles := takeNamespacePIDFiles(options.NetworkNamespaces)
	options.NetworkNamespaces = namespaces
	instance, err := box.New(options)
	if err != nil {
		cancel()
		return realBox{}, err
	}
	gate, err := newNetworkStartupGate(instance.Network().InterfaceMonitor())
	if err != nil {
		cancel()
		return realBox{}, errors.Join(err, instance.Close())
	}
	initialNetwork := networkStartupState{
		ctx: ctx, network: instance.Network(), manager: service.FromContext[pause.Manager](ctx),
		logger: instance.LogFactory().NewLogger("network"),
	}
	gate.initialize = initialNetwork.apply
	updates := newNetworkUpdateWorker(ctx, initialNetwork.update)
	gate.callback = updates.notify
	return realBox{
		box: instance, gate: gate, updates: updates, cancel: cancel, pidFiles: pidFiles,
		namespaces: service.FromContext[adapter.NetworkNamespaceManager](ctx),
	}, nil
}

func (b realBox) Start() error {
	if err := b.pidFiles.prepare(); err != nil {
		return errors.Join(err, b.Close())
	}
	if err := b.box.Start(); err != nil {
		b.gate.close()
		b.cancel()
		b.updates.close()
		return errors.Join(err, b.pidFiles.close())
	}
	if err := b.pidFiles.publish(b.namespaces); err != nil {
		return errors.Join(err, b.Close())
	}
	b.gate.start()
	return nil
}

func (b realBox) Close() error {
	b.cancel()
	b.gate.close()
	b.updates.close()
	return errors.Join(b.box.Close(), b.pidFiles.close())
}

func (b realBox) Router() boxRouter {
	return b.box.Router()
}

func (b realBox) Outbound() boxOutboundManager {
	return b.box.Outbound()
}
