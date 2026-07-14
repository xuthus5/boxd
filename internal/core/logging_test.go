package core

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestNewAppLogHandler(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLogWriter(10)
	h := NewAppLogHandler(&buf, lw, slog.LevelInfo)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestAppLogHandlerEnabled(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLogWriter(10)
	h := NewAppLogHandler(&buf, lw, slog.LevelInfo)

	tests := []struct {
		level slog.Level
		want  bool
	}{
		{slog.LevelDebug, false},
		{slog.LevelInfo, true},
		{slog.LevelWarn, true},
		{slog.LevelError, true},
	}

	for _, tt := range tests {
		got := h.Enabled(context.Background(), tt.level)
		if got != tt.want {
			t.Errorf("Enabled(%s) = %v, want %v", tt.level, got, tt.want)
		}
	}
}

func TestAppLogHandlerHandle(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLogWriter(10)
	h := NewAppLogHandler(&buf, lw, slog.LevelInfo)

	logger := slog.New(h)
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("output should contain 'INFO', got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("output should contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("output should contain 'key=value', got: %s", output)
	}

	// 验证日志也写入了 LogWriter
	recent := lw.Recent()
	if len(recent) != 1 {
		t.Fatalf("expected 1 entry in LogWriter, got %d", len(recent))
	}
	if recent[0].Message != "INFO test message key=value" {
		t.Errorf("LogWriter entry message = %q, want 'test message'", recent[0].Message)
	}
	if recent[0].Level != "info" {
		t.Errorf("LogWriter entry level = %q, want 'info'", recent[0].Level)
	}
}

func TestAppLogHandlerHandleError(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLogWriter(10)
	h := NewAppLogHandler(&buf, lw, slog.LevelWarn)

	logger := slog.New(h)
	logger.Warn("warning msg")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("output should contain 'WARN', got: %s", output)
	}

	recent := lw.Recent()
	if len(recent) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recent))
	}
	if recent[0].Level != "warn" {
		t.Errorf("level = %q, want 'warn'", recent[0].Level)
	}
}

func TestAppLogHandlerHandleDebug(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLogWriter(10)
	h := NewAppLogHandler(&buf, lw, slog.LevelDebug)

	logger := slog.New(h)
	logger.Debug("debug msg")

	recent := lw.Recent()
	if len(recent) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recent))
	}
	if recent[0].Level != "debug" {
		t.Errorf("level = %q, want 'debug'", recent[0].Level)
	}
}

func TestAppLogHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLogWriter(10)
	h := NewAppLogHandler(&buf, lw, slog.LevelInfo)

	child := h.WithAttrs([]slog.Attr{slog.String("module", "test")})
	if child == nil {
		t.Fatal("expected non-nil child handler")
	}

	logger := slog.New(child)
	logger.Info("with attrs msg")

	output := buf.String()
	if !strings.Contains(output, "module=test") {
		t.Errorf("output should contain 'module=test', got: %s", output)
	}
}

func TestAppLogHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	lw := NewLogWriter(10)
	h := NewAppLogHandler(&buf, lw, slog.LevelInfo)

	child := h.WithGroup("mygroup")
	if child == nil {
		t.Fatal("expected non-nil child handler")
	}
}

func TestAppLogHandlerNilLogWriter(t *testing.T) {
	var buf bytes.Buffer
	h := NewAppLogHandler(&buf, nil, slog.LevelInfo)

	logger := slog.New(h)
	logger.Info("no logwriter")

	if !strings.Contains(buf.String(), "no logwriter") {
		t.Errorf("output should contain message, got: %s", buf.String())
	}
}

func TestLevelFromSlog(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelError, "error"},
		{slog.LevelWarn, "warn"},
		{slog.LevelInfo, "info"},
		{slog.LevelDebug, "debug"},
	}

	for _, tt := range tests {
		got := levelFromSlog(tt.level)
		if got != tt.want {
			t.Errorf("levelFromSlog(%s) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestFormatRecord(t *testing.T) {
	var buf bytes.Buffer
	h := NewAppLogHandler(&buf, nil, slog.LevelInfo)
	r := slog.NewRecord(time.Now(), slog.LevelError, "error msg", 0)
	r.AddAttrs(slog.String("attr1", "val1"))

	result := h.formatRecord(r)
	if !strings.Contains(result, "ERROR") {
		t.Errorf("should contain 'ERROR', got: %s", result)
	}
	if !strings.Contains(result, "error msg") {
		t.Errorf("should contain 'error msg', got: %s", result)
	}
	if !strings.Contains(result, "attr1=val1") {
		t.Errorf("should contain 'attr1=val1', got: %s", result)
	}
	if !strings.HasSuffix(result, "\n") {
		t.Errorf("should end with newline, got: %q", result)
	}
}
