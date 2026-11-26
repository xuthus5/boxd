package singbox

import (
	"testing"

	boxlog "github.com/sagernet/sing-box/log"

	"github.com/xuthus5/boxd/kernel"
)

type testLogSink struct {
	entry kernel.LogEntry
	seen  bool
}

func (sink *testLogSink) WriteLog(entry kernel.LogEntry) {
	sink.entry = entry
	sink.seen = true
}

type testEventSink struct {
	event kernel.Event
	seen  bool
}

func (sink *testEventSink) Publish(event kernel.Event) {
	sink.event = event
	sink.seen = true
}

func TestPlatformLogWriterStoresPlainStructuredMessages(t *testing.T) {
	logs := &testLogSink{}
	events := &testEventSink{}
	writer := newPlatformLogWriter(logs, events)
	writer.WriteMessage(boxlog.LevelTrace, "\x1b[37mTRACE\x1b[0m[0364] outbound/vless[42 12ms]: connected")

	if !logs.seen || !events.seen {
		t.Fatalf("writer did not publish both outputs: logs=%v events=%v", logs.seen, events.seen)
	}
	if logs.entry.Level != "trace" || logs.entry.Source != "sing-box" {
		t.Fatalf("entry metadata = %+v", logs.entry)
	}
	if logs.entry.Message != "outbound/vless[42 12ms]: connected" {
		t.Fatalf("entry message = %q", logs.entry.Message)
	}
	if events.event.Payload.(kernel.LogEntry).Message != logs.entry.Message {
		t.Fatalf("event message = %q", events.event.Payload.(kernel.LogEntry).Message)
	}
}

func TestPlatformLogWriterKeepsMessagesWithoutRecognizedPrefix(t *testing.T) {
	logs := &testLogSink{}
	writer := newPlatformLogWriter(logs, nil)
	writer.WriteMessage(boxlog.LevelInfo, "kernel initialized")
	if logs.entry.Message != "kernel initialized" {
		t.Fatalf("entry message = %q", logs.entry.Message)
	}
}

func TestPlatformLogWriterCleansFormattingFragments(t *testing.T) {
	logs := &testLogSink{}
	writer := newPlatformLogWriter(logs, nil)
	writer.WriteMessage(boxlog.LevelTrace, "[37mTRACE[0m[0364] outbound: connected")
	if logs.entry.Message != "outbound: connected" {
		t.Fatalf("entry message = %q", logs.entry.Message)
	}
}
