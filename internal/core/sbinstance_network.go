package core

import (
	"errors"
	"fmt"
	"sync"

	tun "github.com/sagernet/sing-tun"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/common/x/list"
)

var errNetworkCallbackLayout = errors.New("unsupported sing-box network monitor callback layout")

type networkStartupGate struct {
	mu             sync.Mutex
	callback       tun.DefaultInterfaceUpdateCallback
	initialize     tun.DefaultInterfaceUpdateCallback
	started        bool
	closed         bool
	pending        bool
	interfaceValue *control.Interface
	flags          int
}

func newNetworkStartupGate(monitor tun.DefaultInterfaceMonitor) (*networkStartupGate, error) {
	gate := &networkStartupGate{}
	if monitor == nil {
		return gate, nil
	}
	// 1.14 原生监视器把 NetworkManager 的通知注册为首个回调。
	// 仅在 Box.New 完成而监视器尚未启动时修改公开令牌，保留移除令牌的语义。
	if fmt.Sprintf("%T", monitor) != "*tun.defaultInterfaceMonitor" {
		return nil, fmt.Errorf("%w: monitor %T", errNetworkCallbackLayout, monitor)
	}
	marker := monitor.RegisterCallback(nil)
	if marker == nil {
		return nil, fmt.Errorf("%w: missing callback token", errNetworkCallbackLayout)
	}
	err := gate.attach(marker)
	monitor.UnregisterCallback(marker)
	if err != nil {
		return nil, err
	}
	return gate, nil
}

func (g *networkStartupGate) attach(marker *list.Element[tun.DefaultInterfaceUpdateCallback]) error {
	first := marker.Prev()
	if first == nil || marker.Next() != nil {
		return fmt.Errorf("%w: missing initial network callback", errNetworkCallbackLayout)
	}
	for first.Prev() != nil {
		first = first.Prev()
	}
	if first.Value == nil {
		return fmt.Errorf("%w: empty initial network callback", errNetworkCallbackLayout)
	}
	g.callback = first.Value
	first.Value = g.notify
	return nil
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
	g.closed = true
	g.pending = false
	g.interfaceValue = nil
}
