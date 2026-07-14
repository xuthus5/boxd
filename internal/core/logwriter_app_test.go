package core

import "testing"

func TestWriteAppEntry(t *testing.T) {
	lw := NewLogWriter(5)

	lw.WriteAppEntry("error", "app error message")
	lw.WriteAppEntry("info", "app info message")

	recent := lw.Recent()
	if len(recent) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(recent))
	}
	if recent[0].Message != "app error message" {
		t.Errorf("entry 0 message = %q, want 'app error message'", recent[0].Message)
	}
	if recent[0].Level != "error" {
		t.Errorf("entry 0 level = %q, want 'error'", recent[0].Level)
	}
	if recent[1].Message != "app info message" {
		t.Errorf("entry 1 message = %q, want 'app info message'", recent[1].Message)
	}
}

func TestWriteAppEntryBufferLimit(t *testing.T) {
	lw := NewLogWriter(2)
	lw.WriteAppEntry("info", "first")
	lw.WriteAppEntry("info", "second")
	lw.WriteAppEntry("info", "third")

	recent := lw.Recent()
	if len(recent) != 2 {
		t.Fatalf("expected 2 entries (buffer limit), got %d", len(recent))
	}
	if recent[0].Message != "second" {
		t.Errorf("entry 0 = %q, want 'second'", recent[0].Message)
	}
	if recent[1].Message != "third" {
		t.Errorf("entry 1 = %q, want 'third'", recent[1].Message)
	}
}

func TestWriteAppEntrySubscribe(t *testing.T) {
	lw := NewLogWriter(10)
	ch, id := lw.Subscribe()
	defer lw.Unsubscribe(id)

	lw.WriteAppEntry("warn", "subscribed msg")

	entry := <-ch
	if entry.Message != "subscribed msg" {
		t.Errorf("got %q, want 'subscribed msg'", entry.Message)
	}
	if entry.Level != "warn" {
		t.Errorf("level = %q, want 'warn'", entry.Level)
	}
}
