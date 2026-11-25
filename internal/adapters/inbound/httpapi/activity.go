package httpapi

import (
	"context"

	"github.com/xuthus5/boxd/control"
)

// activitySink is a thin web-facing adapter over the control-plane activity
// log. It keeps handler call sites concise without owning storage or fan-out.
type activitySink struct {
	api control.ActivityLogAPI
}

func (s activitySink) add(level, source, message string, fields map[string]any) {
	if s.api == nil {
		return
	}
	s.api.Record(context.Background(), control.ActivityLogEntry{
		Level: level, Source: source, Message: message, Fields: fields,
	})
}

func (s activitySink) recent(limit int) []control.ActivityLogEntry {
	if s.api == nil {
		return nil
	}
	return s.api.Recent(limit)
}

func (s activitySink) subscribe(ctx context.Context) <-chan control.ActivityLogEntry {
	if s.api == nil {
		channel := make(chan control.ActivityLogEntry)
		close(channel)
		return channel
	}
	subscription, err := s.api.Subscribe(ctx)
	if err != nil {
		channel := make(chan control.ActivityLogEntry)
		close(channel)
		return channel
	}
	return subscription
}
