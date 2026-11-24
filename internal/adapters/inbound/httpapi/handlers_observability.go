package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

func (s *Server) registerObservabilityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/observability/logs", s.handleLogs)
	mux.HandleFunc("GET /api/v1/observability/traffic", s.handleTraffic)
	mux.HandleFunc("GET /api/v1/observability/connections", s.handleConnections)
	mux.HandleFunc("DELETE /api/v1/observability/connections", s.handleCloseAllConnections)
	mux.HandleFunc("DELETE /api/v1/observability/connections/{id}", s.handleCloseConnection)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 200, 500)
	source := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
	response := logsView{}
	if source == "" || source == "all" || source == "app" {
		response.App = makeAppLogViews(s.logs.recent(limit))
	}
	if source == "" || source == "all" || source == "kernel" {
		response.Kernel = makeKernelLogViews(s.app.Observability().RecentLogs(limit))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleTraffic(w http.ResponseWriter, r *http.Request) {
	traffic, err := s.app.Observability().Traffic(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	status, err := s.app.Kernel().Status(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeTrafficView(traffic, time.Now().UTC(), status.State == kernel.StateRunning))
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	connections, err := s.app.Observability().Connections(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeConnectionViews(connections, time.Now().UTC()))
}

func (s *Server) handleCloseConnection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, r, newAPIError(http.StatusBadRequest, "invalid_connection_id", "connections.close", "connection ID is invalid", "reload the connection list"))
		return
	}
	if err := s.app.Observability().CloseConnection(r.Context(), id); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "connections", "connection closed", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]bool{"closed": true})
}

func (s *Server) handleCloseAllConnections(w http.ResponseWriter, r *http.Request) {
	count, err := s.app.Observability().CloseConnections(r.Context(), kernel.ConnectionFilter{})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("warn", "connections", "all connections closed", map[string]any{"count": count})
	writeJSON(w, http.StatusOK, map[string]int{"closed": count})
}

func parseLimit(raw string, fallback, maximum int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func (s *Server) registerEventRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events", s.handleEvents)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, r, newAPIError(http.StatusInternalServerError, "streaming_unavailable", "events.stream", "streaming is unavailable", "use a compatible HTTP server"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ctx := r.Context()
	kernelEvents, err := s.app.Observability().Subscribe(ctx,
		kernel.EventKernelState,
		kernel.EventKernelLog,
		kernel.EventConfigApplied,
		kernel.EventControlChanged,
	)
	if err != nil {
		return
	}
	appLogs := s.logs.subscribe(ctx)
	trafficTicker := time.NewTicker(2 * time.Second)
	keepaliveTicker := time.NewTicker(15 * time.Second)
	defer trafficTicker.Stop()
	defer keepaliveTicker.Stop()

	if !writeSSE(w, flusher, "ready", map[string]any{"server_time": time.Now().UTC(), "transport": "sse"}) {
		return
	}
	if !s.writeTrafficEvent(w, flusher, r) {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case entry, open := <-appLogs:
			if !open {
				appLogs = nil
				continue
			}
			if !writeSSE(w, flusher, "app.log", logView{ID: "app-" + strconv.FormatUint(entry.ID, 10), Time: entry.Time, Level: entry.Level, Source: entry.Source, Message: entry.Message, Fields: entry.Fields}) {
				return
			}
		case event, open := <-kernelEvents:
			if !open {
				kernelEvents = nil
				continue
			}
			if !s.writeKernelEvent(w, flusher, r, event) {
				return
			}
		case <-trafficTicker.C:
			if !s.writeTrafficEvent(w, flusher, r) {
				return
			}
		case <-keepaliveTicker.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
		if kernelEvents == nil && appLogs == nil {
			return
		}
	}
}

func (s *Server) writeKernelEvent(w http.ResponseWriter, flusher http.Flusher, r *http.Request, event kernel.Event) bool {
	switch event.Type {
	case kernel.EventKernelState:
		status, err := s.app.Kernel().Status(r.Context())
		if err != nil {
			return false
		}
		return writeSSE(w, flusher, "kernel.state", makeKernelView(status, time.Now().UTC()))
	case kernel.EventKernelLog:
		entry, ok := event.Payload.(kernel.LogEntry)
		if !ok {
			return true
		}
		return writeSSE(w, flusher, "kernel.log", logView{
			ID: "kernel-" + strconv.FormatUint(event.Sequence, 10), Time: entry.Time, Level: strings.ToLower(entry.Level),
			Source: entry.Source, Message: entry.Message, Fields: entry.Fields,
		})
	case kernel.EventConfigApplied:
		return writeSSE(w, flusher, "snapshot.invalidated", map[string]any{"reason": "config.applied", "at": event.Time})
	case kernel.EventControlChanged:
		return writeSSE(w, flusher, "snapshot.invalidated", map[string]any{"reason": "control.changed", "at": event.Time})
	default:
		return true
	}
}

func (s *Server) writeTrafficEvent(w http.ResponseWriter, flusher http.Flusher, r *http.Request) bool {
	traffic, err := s.app.Observability().Traffic(r.Context())
	if err != nil {
		return true
	}
	available := false
	if status, statusErr := s.app.Kernel().Status(r.Context()); statusErr == nil {
		available = status.State == kernel.StateRunning
	}
	connections, connErr := s.app.Observability().Connections(r.Context())
	connectionCount := 0
	if connErr == nil {
		connectionCount = len(connections)
	}
	return writeSSE(w, flusher, "traffic", map[string]any{
		"traffic": makeTrafficView(traffic, time.Now().UTC(), available), "connections": connectionCount,
	})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) bool {
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body); err != nil {
		return false
	}
	flusher.Flush()
	return true
}
