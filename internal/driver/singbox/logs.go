package singbox

import (
	"strings"
	"time"

	boxlog "github.com/sagernet/sing-box/log"

	"github.com/xuthus5/boxd/internal/logtext"
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
		Message: cleanPlatformMessage(level, message),
	}
	if w.logs != nil {
		w.logs.WriteLog(entry)
	}
	if w.events != nil {
		w.events.Publish(kernel.Event{Type: kernel.EventKernelLog, Payload: entry})
	}
}

func cleanPlatformMessage(level boxlog.Level, message string) string {
	message = logtext.StripTerminalFormatting(message)
	label := strings.ToUpper(boxlog.FormatLevel(level))
	prefix := label + "["
	if !strings.HasPrefix(message, prefix) {
		return message
	}
	remainder := message[len(prefix):]
	end := strings.IndexByte(remainder, ']')
	if end < 1 {
		return message
	}
	for _, char := range remainder[:end] {
		if char < '0' || char > '9' {
			return message
		}
	}
	return strings.TrimPrefix(remainder[end+1:], " ")
}
