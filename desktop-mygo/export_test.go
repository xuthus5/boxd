package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

func TestLogExportFormatting(t *testing.T) {
	text := formatLogExport([]core.LogEntry{{
		Timestamp: time.Date(2026, 11, 12, 3, 20, 0, 0, time.UTC),
		Level:     "warn", Message: "slow refresh",
	}})
	if !strings.Contains(text, "warn\tslow refresh") || !strings.HasPrefix(text, "2026-11-12T") {
		t.Fatalf("log export = %q", text)
	}
	if got := formatLogExport(nil); got != "" {
		t.Fatalf("empty log export = %q", got)
	}
}

func TestConfigHistoryExportFormatting(t *testing.T) {
	text, err := formatConfigHistoryExport([]model.ConfigApplyEvent{{ID: "snap-1", Status: "applied"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `"count": 1`) || !strings.Contains(text, `"snap-1"`) {
		t.Fatalf("history export = %q", text)
	}
	name := exportFilename("boxd-logs", "log")
	if !strings.HasPrefix(name, "boxd-logs-") || !strings.HasSuffix(name, ".log") {
		t.Fatalf("export filename = %q", name)
	}
}
