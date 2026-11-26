package mygo

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/kernel"
)

func TestMergedLogsStripTerminalFormatting(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.snapshot.KernelLogs = []kernel.LogEntry{{
		Time: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC), Level: "trace", Source: "sing-box",
		Message: "\x1b[37mTRACE\x1b[0m[0364] outbound: connected",
	}}
	logs := controller.snapshot.mergedLogs(false, true)
	if len(logs) != 1 {
		t.Fatalf("logs = %+v", logs)
	}
	if logs[0].Message != "TRACE[0364] outbound: connected" {
		t.Fatalf("message = %q", logs[0].Message)
	}
}

func TestLogsPageNeverRendersTerminalEscapeSequences(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.snapshot.KernelLogs = []kernel.LogEntry{{
		Time: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC), Level: "trace", Source: "sing-box",
		Message: "\x1b[37mTRACE\x1b[0m[0364] outbound: connected",
	}}
	controller.page = pageLogs
	tester := ui.NewTester(controller.View, 1320, 860)
	if !tester.HasText("outbound: connected") {
		t.Fatalf("log message missing; texts=%q", tester.Texts())
	}
	if tester.HasText("\x1b") || tester.HasText("[37m") || tester.HasText("[0m") {
		t.Fatalf("terminal escape sequence rendered; texts=%q", tester.Texts())
	}
}

func TestLogLevelColorsUseSemanticTones(t *testing.T) {
	p := lightPalette()
	tests := []struct {
		level string
		want  ui.Color
	}{
		{level: "trace", want: p.TextFaint},
		{level: "debug", want: p.Violet},
		{level: "info", want: p.Sky},
		{level: "warn", want: p.Warning},
		{level: "warning", want: p.Warning},
		{level: "error", want: p.Danger},
		{level: "fatal", want: p.Danger},
		{level: "panic", want: p.Danger},
		{level: "unknown", want: p.TextMuted},
	}
	for _, test := range tests {
		if got := logLevelColor(p, test.level); got != test.want {
			t.Errorf("logLevelColor(%q) = %v, want %v", test.level, got, test.want)
		}
	}
}
