package main

import (
	"context"
	"strconv"

	"github.com/xuthus5/boxd/internal/service"
)

func (a *application) askCloseConnection(conn connectionView) {
	if a.backend == nil || a.model.busy {
		return
	}
	a.deleteForm = deleteFormState{
		open: true, kind: "connection", id: strconv.FormatInt(conn.ID, 10), title: "Close connection?",
		message: "The active connection to " + conn.Target + " will be terminated.", confirm: "Close",
	}
}

func (a *application) askCloseConnections(connections []connectionView, filtered bool) {
	if a.backend == nil || a.model.busy {
		return
	}
	title, message := "Close all connections?", "All active connections will be terminated."
	if filtered {
		title = "Close filtered connections?"
		message = "Every connection matching the current filter will be terminated."
	}
	ids := make([]string, 0, len(connections))
	if filtered {
		for _, conn := range connections {
			ids = append(ids, strconv.FormatInt(conn.ID, 10))
		}
	}
	a.deleteForm = deleteFormState{open: true, kind: "connections", ids: ids, title: title, message: message, confirm: "Close"}
}

func (a *application) closeConnections(filters service.ConnectionCloseFilters) {
	a.runActionWithCallback("Closing connections", func(ctx context.Context) error {
		_, err := a.backend.CloseConnections(ctx, filters)
		return err
	}, "Connections closed", func() { a.requestRefresh() })
}
