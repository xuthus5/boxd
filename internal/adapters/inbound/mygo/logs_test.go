package mygo

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xuthus5/boxd/control"
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

func TestLogsPageUsesAvailableWidth(t *testing.T) {
	for _, width := range []int{1920, 2560} {
		t.Run(fmt.Sprintf("%dpx", width), func(t *testing.T) {
			controller := newViewTestController(ControllerOptions{})
			controller.page = pageLogs
			tester := ui.NewTester(controller.View, width, 1000)
			bounds, ok := tester.Find("日志内容")
			if !ok {
				t.Fatalf("log content did not render; texts=%q", tester.Texts())
			}
			if right := bounds.X + bounds.W; right < float32(width)-60 {
				t.Fatalf("log content ended at %.1fpx in a %dpx window", right, width)
			}
		})
	}
}

func TestLogsPageKeepsBottomRowAboveViewportEdge(t *testing.T) {
	controller := newViewTestController(ControllerOptions{})
	controller.page = pageLogs
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	controller.snapshot.Activity = make([]control.ActivityLogEntry, 80)
	for i := range controller.snapshot.Activity {
		controller.snapshot.Activity[i] = control.ActivityLogEntry{
			ID:      uint64(i + 1),
			Time:    base.Add(time.Duration(i) * time.Second),
			Level:   "info",
			Source:  "desktop",
			Message: fmt.Sprintf("log row %03d", i),
		}
	}
	tester := ui.NewTester(controller.View, 960, 640)
	viewport, ok := tester.Find("日志内容")
	if !ok {
		t.Fatalf("log content did not render; texts=%q", tester.Texts())
	}
	tester.Scroll(viewport.X+viewport.W/2, viewport.Y+viewport.H/2, 0, 100000)
	tester.Frame()

	viewport, ok = tester.Find("日志内容")
	if !ok {
		t.Fatalf("log content disappeared after scrolling; texts=%q", tester.Texts())
	}
	oldest := controller.snapshot.Activity[0]
	label := oldest.Time.Local().Format("2006-01-02 15:04:05.000") + "  INFO  " + oldest.Message
	row, ok := tester.Find(label)
	if !ok {
		t.Fatalf("oldest log row was not rendered after scrolling; texts=%q", tester.Texts())
	}
	if row.H < 26 {
		t.Fatalf("oldest log row was clipped to %.1fpx", row.H)
	}
	if gap := viewport.Y + viewport.H - (row.Y + row.H); gap < 20 {
		t.Fatalf("oldest log row ended %.1fpx from the viewport edge, want at least 20px", gap)
	}
}
