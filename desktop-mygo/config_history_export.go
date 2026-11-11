package main

import (
	"encoding/json"
	"time"

	"github.com/xuthus5/boxd/internal/model"
)

type configHistoryExport struct {
	ExportedAt time.Time                `json:"exported_at"`
	Count      int                      `json:"count"`
	Records    []model.ConfigApplyEvent `json:"records"`
}

func formatConfigHistoryExport(events []model.ConfigApplyEvent) (string, error) {
	body, err := json.MarshalIndent(configHistoryExport{
		ExportedAt: time.Now().UTC(), Count: len(events), Records: events,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(body) + "\n", nil
}

func (a *application) exportConfigHistory() {
	if len(a.configHistory.events) == 0 {
		return
	}
	text, err := formatConfigHistoryExport(a.configHistory.events)
	if err != nil {
		a.model.setActionError(err)
		a.invalidate()
		return
	}
	a.exportTextFile("Export configuration history", exportFilename("boxd-config-history", "json"), "json", text)
}
