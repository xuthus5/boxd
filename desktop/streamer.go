package main

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/xuthus5/boxd/internal/core"
)

// eventEmitter 抽象事件发射能力，便于测试注入。
type eventEmitter interface {
	Emit(name string, data ...any) bool
}

// appEventEmitter 适配 application.App 为 eventEmitter。
type appEventEmitter struct {
	app *application.App
}

func (a appEventEmitter) Emit(name string, data ...any) bool {
	if a.app == nil {
		return false
	}
	return a.app.Event.Emit(name, data...)
}

// EventStreamer 将内核/应用日志、流量与连接快照通过 Wails Events 推送给前端，
// 替代桌面模式下的 SSE 通道。
type EventStreamer struct {
	emitter   eventEmitter
	rt        *desktopRuntime
	kernelLog *core.LogWriter
	appLog    *core.LogWriter
	stopCh    chan struct{}
	stopOnce  sync.Once
}

// NewEventStreamer 构造事件流推送器。
func NewEventStreamer(app *application.App, rt *desktopRuntime) *EventStreamer {
	return newEventStreamer(appEventEmitter{app: app}, rt)
}

// newEventStreamer 用可注入发射器构造事件流推送器，便于测试。
func newEventStreamer(emitter eventEmitter, rt *desktopRuntime) *EventStreamer {
	var kernelLog, appLog *core.LogWriter
	if rt != nil && rt.svc != nil {
		kernelLog = rt.svc.Deps.KernelLogWriter
		appLog = rt.svc.Deps.AppLogWriter
	}
	return &EventStreamer{
		emitter:   emitter,
		rt:        rt,
		kernelLog: kernelLog,
		appLog:    appLog,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动日志订阅与流量采样推送。
func (s *EventStreamer) Start() {
	go s.pushLogs(KernelLogEventName, s.kernelLog)
	go s.pushLogs(AppLogEventName, s.appLog)
	go s.pushTrafficTicker()
}

// Stop 停止事件推送。
func (s *EventStreamer) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

// emit 通过事件发射器推送数据。
func (s *EventStreamer) emit(name string, data any) {
	if s.emitter == nil {
		return
	}
	s.emitter.Emit(name, data)
}

// pushLogs 订阅日志并推送新条目。
func (s *EventStreamer) pushLogs(eventName string, writer *core.LogWriter) {
	if writer == nil {
		return
	}
	recent, ch, id := writer.SnapshotAndSubscribe()
	defer writer.Unsubscribe(id)

	for _, entry := range recent {
		s.emit(eventName, entry)
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			// 心跳：保持前端连接状态为 open。
			s.emit(eventName, map[string]string{"type": "heartbeat"})
		case entry, open := <-ch:
			if !open {
				return
			}
			s.emit(eventName, entry)
		}
	}
}

// pushTrafficTicker 每秒推送流量与连接快照。
func (s *EventStreamer) pushTrafficTicker() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.emitTraffic()
			s.emitConnections()
		}
	}
}

func (s *EventStreamer) emitTraffic() {
	if s.rt == nil || s.rt.instance == nil || s.rt.instance.TrafficTracker() == nil {
		s.emit(TrafficEventName, map[string]any{
			"upload_bytes":   int64(0),
			"download_bytes": int64(0),
			"timestamp":      time.Now().Format(time.RFC3339),
		})
		return
	}
	up, down := s.rt.instance.TrafficTracker().Total()
	s.emit(TrafficEventName, map[string]any{
		"upload_bytes":   up,
		"download_bytes": down,
		"timestamp":      time.Now().Format(time.RFC3339),
	})
}

func (s *EventStreamer) emitConnections() {
	if s.rt == nil || s.rt.instance == nil || s.rt.instance.TrafficTracker() == nil {
		s.emit(ConnectionsEventName, map[string]any{
			"active_connections": 0,
			"list":               []core.TrafficConn{},
		})
		return
	}
	list := s.rt.instance.TrafficTracker().Connections()
	s.emit(ConnectionsEventName, map[string]any{
		"active_connections": len(list),
		"list":               list,
	})
}

// 事件名常量，前端与 Go 侧共用。
const (
	TrafficEventName     = "boxd:traffic"
	ConnectionsEventName = "boxd:connections"
	KernelLogEventName   = "boxd:kernel-log"
	AppLogEventName      = "boxd:app-log"
)
