package main

import (
	"strings"

	"github.com/xuthus5/boxd/internal/core"
)

func formatLogExport(entries []core.LogEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")+"\t"+
			firstText(entry.Level, "info")+"\t"+entry.Message)
	}
	return strings.Join(lines, "\n")
}

func (a *application) copyLogs() {
	logs := a.model.visibleLogs()
	if len(logs) == 0 {
		return
	}
	if err := mygoClipboardWrite(formatLogExport(logs)); err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	a.model.actionError = ""
	a.model.notice = "Log entries copied"
	a.invalidate()
}

func (a *application) exportLogs() {
	logs := a.model.visibleLogs()
	if len(logs) == 0 {
		return
	}
	a.exportTextFile("Export logs", exportFilename("boxd-logs", "log"), "log", formatLogExport(logs))
}
