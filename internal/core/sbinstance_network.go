package core

import (
	"errors"
	"fmt"
	"sync"

	tun "github.com/sagernet/sing-tun"
	"github.com/sagernet/sing/common/control"
)

var errNetworkCallbackLayout = errors.New("unsupported sing-box network monitor callback layout")

type networkStartupGate struct {
	mu             sync.Mutex
	callback       tun.DefaultInterfaceUpdateCallback
	initialize     tun.DefaultInterfaceUpdateCallback
	unregister     func()
	started        bool
	closed         bool
	pending        bool
	interfaceValue *control.Interface
	flags          int
}

// newNetworkStartupGate 在 Box.New 之后、内核 Start 之前挂载 boxd 的接口回调。
// 1.14.2 起内核改为在 Start(Initialize) 阶段才注册自己的接口回调，监视器在 New
// 之后为空，因此这里不再接管既有回调，而是自行注册一个，用于缓冲启动期通知并
// 在 boxd 启动完成后重放。
func newNetworkStartupGate(monitor tun.DefaultInterfaceMonitor) (*networkStartupGate, error) {
	gate := &networkStartupGate{}
	if monitor == nil {
		return gate, nil
	}
	if fmt.Sprintf("%T", monitor) != "*tun.defaultInterfaceMonitor" {
		return nil, fmt.Errorf("%w: monitor %T", errNetworkCallbackLayout, monitor)
	}
	element := monitor.RegisterCallback(gate.notify)
	gate.unregister = func() { monitor.UnregisterCallback(element) }
	return gate, nil
}

func (g *networkStartupGate) notify(networkInterface *control.Interface, flags int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return
	}
	if !g.started {
		if networkInterface == nil && g.initialize != nil {
			g.pending = false
			g.interfaceValue = nil
			g.initialize(nil, flags)
			return
		}
		g.pending = true
		g.interfaceValue = networkInterface
		g.flags = flags
		return
	}
	g.callback(networkInterface, flags)
}

func (g *networkStartupGate) start() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.started {
		return
	}
	g.started = true
	if g.pending {
		g.pending = false
		callback := g.initialize
		if callback == nil {
			callback = g.callback
		}
		callback(g.interfaceValue, g.flags)
		g.interfaceValue = nil
	}
}

func (g *networkStartupGate) close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return
	}
	g.closed = true
	g.pending = false
	g.interfaceValue = nil
	if g.unregister != nil {
		g.unregister()
	}
}
