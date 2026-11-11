package main

import (
	"strconv"

	"github.com/xuthus5/boxd/internal/service"
)

func (a *application) confirmDelete() {
	form := a.deleteForm
	a.deleteForm = deleteFormState{}
	switch form.kind {
	case "node":
		a.deleteNode(form.id)
	case "subscription":
		a.deleteSubscription(form.id)
	case "connection":
		if id, err := strconv.ParseInt(form.id, 10, 64); err == nil {
			a.closeConnection(id)
		}
	case "connections":
		filters := service.ConnectionCloseFilters{IDs: form.ids}
		a.closeConnections(filters)
	case "config-restore":
		a.restoreConfigSnapshot(form.id)
	case "config-section-item":
		index := ""
		if len(form.ids) > 0 {
			index = form.ids[0]
		}
		a.deleteConfigSectionItem(form.id, index)
	}
}
