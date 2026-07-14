package core

import (
	"testing"

	"github.com/sagernet/sing-box/log"
)

func TestNewLogWriter(t *testing.T) {
	lw := NewLogWriter(100)
	if lw.maxBuf != 100 {
		t.Errorf("expected maxBuf 100, got %d", lw.maxBuf)
	}
}

func TestLogWriterWriteAndRecent(t *testing.T) {
	lw := NewLogWriter(10)
	lw.WriteMessage(log.LevelInfo, "test message")

	recent := lw.Recent()
	if len(recent) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recent))
	}
	if recent[0].Message != "test message" {
		t.Errorf("expected 'test message', got '%s'", recent[0].Message)
	}
	if recent[0].Level != "info" {
		t.Errorf("expected level 'info', got '%s'", recent[0].Level)
	}
}

func TestLogWriterBufferLimit(t *testing.T) {
	lw := NewLogWriter(3)
	for i := 0; i < 5; i++ {
		lw.WriteMessage(log.LevelInfo, "msg")
	}
	recent := lw.Recent()
	if len(recent) != 3 {
		t.Errorf("expected 3 entries (buffer limit), got %d", len(recent))
	}
}

func TestLogWriterSubscribe(t *testing.T) {
	lw := NewLogWriter(10)
	ch, id := lw.Subscribe()
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}

	lw.WriteMessage(log.LevelInfo, "sub msg")
	entry := <-ch
	if entry.Message != "sub msg" {
		t.Errorf("expected 'sub msg', got '%s'", entry.Message)
	}
}

func TestLogWriterUnsubscribe(t *testing.T) {
	lw := NewLogWriter(10)
	_, id := lw.Subscribe()
	lw.Unsubscribe(id)

	lw.WriteMessage(log.LevelInfo, "after unsubscribe")
	if len(lw.subs) != 0 {
		t.Errorf("expected 0 subscribers, got %d", len(lw.subs))
	}
}

func TestLogWriterMultipleSubscribers(t *testing.T) {
	lw := NewLogWriter(10)
	ch1, _ := lw.Subscribe()
	ch2, _ := lw.Subscribe()

	lw.WriteMessage(log.LevelInfo, "broadcast")

	entry1 := <-ch1
	entry2 := <-ch2
	if entry1.Message != "broadcast" || entry2.Message != "broadcast" {
		t.Error("broadcast failed")
	}
}

func TestLogWriterANSIStripping(t *testing.T) {
	lw := NewLogWriter(10)
	lw.WriteMessage(log.LevelInfo, "\x1b[31mcolored\x1b[0m")
	recent := lw.Recent()
	if recent[0].Message != "colored" {
		t.Errorf("expected 'colored' (no ANSI), got '%s'", recent[0].Message)
	}
}

func TestLevelToString(t *testing.T) {
	tests := []struct {
		level    log.Level
		expected string
	}{
		{log.LevelPanic, "error"},
		{log.LevelFatal, "error"},
		{log.LevelError, "error"},
		{log.LevelWarn, "warn"},
		{log.LevelInfo, "info"},
		{log.LevelDebug, "debug"},
		{log.LevelTrace, "debug"},
		{127, "info"},
	}
	for _, tt := range tests {
		result := levelToString(tt.level)
		if result != tt.expected {
			t.Errorf("levelToString(%d) = %s, want %s", tt.level, result, tt.expected)
		}
	}
}
