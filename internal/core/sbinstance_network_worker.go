package core

import (
	"context"
	"sync"

	"github.com/sagernet/sing/common/control"
)

type networkUpdateEvent struct {
	interfaceValue *control.Interface
	flags          int
}

type networkUpdateWorker struct {
	ctx           context.Context
	cancel        context.CancelFunc
	handle        func(context.Context, *control.Interface, int)
	mu            sync.Mutex
	closed        bool
	pending       *networkUpdateEvent
	currentCancel context.CancelFunc
	wake          chan struct{}
	done          chan struct{}
}

func newNetworkUpdateWorker(ctx context.Context, handle func(context.Context, *control.Interface, int)) *networkUpdateWorker {
	ctx, cancel := context.WithCancel(ctx)
	worker := &networkUpdateWorker{
		ctx: ctx, cancel: cancel, handle: handle,
		wake: make(chan struct{}, 1), done: make(chan struct{}),
	}
	go worker.run()
	return worker
}

func (w *networkUpdateWorker) notify(networkInterface *control.Interface, flags int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.ctx.Err() != nil {
		return
	}
	w.pending = &networkUpdateEvent{interfaceValue: networkInterface, flags: flags}
	if w.currentCancel != nil {
		w.currentCancel()
	}
	// 通知不等待网络 I/O；连续变更合并到最新状态，唤醒信号不会丢失。
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *networkUpdateWorker) run() {
	defer close(w.done)
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.wake:
		}
		ctx, event, cancel := w.takePending()
		if ctx == nil {
			continue
		}
		w.handle(ctx, event.interfaceValue, event.flags)
		cancel()
		w.mu.Lock()
		w.currentCancel = nil
		w.mu.Unlock()
	}
}

func (w *networkUpdateWorker) takePending() (context.Context, *networkUpdateEvent, context.CancelFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.ctx.Err() != nil || w.pending == nil {
		return nil, nil, nil
	}
	event := w.pending
	w.pending = nil
	ctx, cancel := context.WithCancel(w.ctx)
	w.currentCancel = cancel
	return ctx, event, cancel
}

func (w *networkUpdateWorker) close() {
	w.mu.Lock()
	w.closed = true
	w.pending = nil
	w.mu.Unlock()
	w.cancel()
	<-w.done
}
