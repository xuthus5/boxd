package singbox

import (
	"time"

	boxlog "github.com/sagernet/sing-box/log"

	"github.com/xuthus5/boxd/kernel"
)

type platformLogWriter struct {
	logs   kernel.LogSink
	events kernel.EventSink
}

var _ boxlog.PlatformWriter = (*platformLogWriter)(nil)

func newPlatformLogWriter(logs kernel.LogSink, events kernel.EventSink) *platformLogWriter {
	if logs == nil && events == nil {
		return nil
	}
	return &platformLogWriter{logs: logs, events: events}
}

func (w *platformLogWriter) WriteMessage(level boxlog.Level, message string) {
	if w == nil {
		return
	}
	entry := kernel.LogEntry{
		Time:    time.Now().UTC(),
		Level:   boxlog.FormatLevel(level),
		Source:  "sing-box",
		Message: message,
	}
	if w.logs != nil {
		w.logs.WriteLog(entry)
	}
	if w.events != nil {
		w.events.Publish(kernel.Event{Type: kernel.EventKernelLog, Payload: entry})
	}
}
