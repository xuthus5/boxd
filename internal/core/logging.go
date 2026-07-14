package core

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// AppLogHandler 将 BoxUI 自身的结构化日志同时输出到 io.Writer 和 LogWriter，
// 使管理面板日志页面能同时展示 sing-box 内核日志与 BoxUI 自身日志。
type AppLogHandler struct {
	mu     sync.Mutex
	w      io.Writer
	logger *LogWriter
	attrs  []slog.Attr
	level  slog.Level
}

// NewAppLogHandler 创建一个同时写入 io.Writer 和 LogWriter 的 slog handler。
func NewAppLogHandler(w io.Writer, lw *LogWriter, level slog.Level) *AppLogHandler {
	return &AppLogHandler{
		w:      w,
		logger: lw,
		level:  level,
	}
}

func (h *AppLogHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level
}

func (h *AppLogHandler) Handle(_ context.Context, r slog.Record) error {
	line := h.formatRecord(r)
	h.mu.Lock()
	_, err := io.WriteString(h.w, line)
	h.mu.Unlock()
	if err != nil {
		return err
	}

	if h.logger != nil {
		// 去除末尾换行，将完整结构化消息传入 LogWriter
		msg := strings.TrimRight(line, "\n")
		h.logger.WriteAppEntry(levelFromSlog(r.Level), msg)
	}

	return nil
}

func (h *AppLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &AppLogHandler{
		w:      h.w,
		logger: h.logger,
		attrs:  append(h.attrs[:len(h.attrs):len(h.attrs)], attrs...),
		level:  h.level,
	}
}

func (h *AppLogHandler) WithGroup(name string) slog.Handler {
	return h
}

// formatRecord 格式化日志记录，包含 handler 预设 attrs 和 record 内 attrs。
func (h *AppLogHandler) formatRecord(r slog.Record) string {
	var buf []byte
	buf = append(buf, r.Level.String()...)
	buf = append(buf, ' ')
	buf = append(buf, r.Message...)

	// 先输出 handler 预设 attrs
	for _, a := range h.attrs {
		buf = append(buf, ' ')
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
	}

	// 再输出 record 内 attrs
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
		return true
	})

	buf = append(buf, '\n')
	return string(buf)
}

func levelFromSlog(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "error"
	case l >= slog.LevelWarn:
		return "warn"
	case l >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}
