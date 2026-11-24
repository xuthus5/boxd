package httpapi

import (
	"net/http"
	"time"
)

func (s *Server) registerSnapshotRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/snapshot", s.handleSnapshot)
	mux.HandleFunc("GET /api/v1/capabilities", s.handleCapabilities)
	mux.HandleFunc("GET /api/v1/scheduler", s.handleScheduler)
	mux.HandleFunc("GET /api/v1/config/history", s.handleConfigHistory)
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.buildSnapshot(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	descriptor, err := s.app.Setup().Capabilities(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, capabilitiesView{Kind: descriptor.Kind, Version: descriptor.Version, Items: descriptor.Capabilities})
}

func (s *Server) handleScheduler(w http.ResponseWriter, r *http.Request) {
	status, err := s.app.Scheduler().Status(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleConfigHistory(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20, 200)
	history, err := s.app.Config().History(r.Context(), limit)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": history, "at": time.Now().UTC()})
}
