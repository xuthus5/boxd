package core

import (
	"regexp"
	"sync"

	"github.com/sagernet/sing-box/log"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type LogWriter struct {
	mu     sync.RWMutex
	buffer []LogEntry
	maxBuf int
	subs   map[int]chan LogEntry
	nextID int
}

func NewLogWriter(maxBuf int) *LogWriter {
	return &LogWriter{
		buffer: make([]LogEntry, 0, maxBuf),
		maxBuf: maxBuf,
		subs:   make(map[int]chan LogEntry),
	}
}

func (w *LogWriter) WriteMessage(level log.Level, message string) {
	entry := LogEntry{
		Level:   levelToString(level),
		Message: ansiRe.ReplaceAllString(message, ""),
	}

	w.mu.Lock()
	w.buffer = append(w.buffer, entry)
	if len(w.buffer) > w.maxBuf {
		w.buffer = w.buffer[1:]
	}
	subs := make([]chan LogEntry, 0, len(w.subs))
	for _, ch := range w.subs {
		subs = append(subs, ch)
	}
	w.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- entry:
		default:
		}
	}
}

func (w *LogWriter) Subscribe() (chan LogEntry, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.nextID++
	ch := make(chan LogEntry, 64)
	w.subs[w.nextID] = ch
	return ch, w.nextID
}

func (w *LogWriter) Unsubscribe(id int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if ch, ok := w.subs[id]; ok {
		close(ch)
		delete(w.subs, id)
	}
}

func (w *LogWriter) Recent() []LogEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]LogEntry, len(w.buffer))
	copy(result, w.buffer)
	return result
}

func levelToString(level log.Level) string {
	switch level {
	case log.LevelPanic, log.LevelFatal, log.LevelError:
		return "error"
	case log.LevelWarn:
		return "warn"
	case log.LevelInfo:
		return "info"
	case log.LevelDebug, log.LevelTrace:
		return "debug"
	default:
		return "info"
	}
}

// WriteAppEntry 写入 BoxUI 自身的结构化日志条目，与 sing-box 内核日志共用同一缓冲区。
func (w *LogWriter) WriteAppEntry(level, message string) {
	entry := LogEntry{
		Level:   level,
		Message: message,
	}

	w.mu.Lock()
	w.buffer = append(w.buffer, entry)
	if len(w.buffer) > w.maxBuf {
		w.buffer = w.buffer[1:]
	}
	subs := make([]chan LogEntry, 0, len(w.subs))
	for _, ch := range w.subs {
		subs = append(subs, ch)
	}
	w.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- entry:
		default:
		}
	}
}
