package mygo

import (
	"strings"
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
	if logs[0].Message != "outbound: connected" {
		t.Fatalf("message = %q", logs[0].Message)
	}
}

func TestMergedLogsCleanUpstreamStrippedColorFragments(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.snapshot.KernelLogs = []kernel.LogEntry{{
		Time: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC), Level: "trace", Source: "sing-box",
		Message: "[37mTRACE[0m[0364] outbound: connected",
	}}
	logs := controller.snapshot.mergedLogs(false, true)
	if len(logs) != 1 || logs[0].Message != "outbound: connected" {
		t.Fatalf("logs = %+v", logs)
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
	if tester.HasText("TRACE[0364]") {
		t.Fatalf("duplicate structured level rendered; texts=%q", tester.Texts())
	}
	if tester.HasText("sing-box") {
		t.Fatalf("redundant source column rendered; texts=%q", tester.Texts())
	}
}

func TestCopiedLogMatchesVisibleColumns(t *testing.T) {
	line := mergedLog{
		Time: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC), Level: "trace",
		Source: "sing-box", Message: "outbound: connected",
	}
	text := formatLogForCopy(line)
	if !strings.Contains(text, "[TRACE] outbound: connected") {
		t.Fatalf("copied log = %q", text)
	}
	if strings.Contains(text, "sing-box") || strings.ContainsAny(text, "\x1b") {
		t.Fatalf("copied log contains hidden metadata: %q", text)
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
