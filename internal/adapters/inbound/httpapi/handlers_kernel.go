package httpapi

import (
	"net/http"
	"time"

	"github.com/xuthus5/boxd/kernel"
)

func (s *Server) registerKernelRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/kernel/status", s.handleKernelStatus)
	mux.HandleFunc("POST /api/v1/kernel/start", s.handleKernelStart)
	mux.HandleFunc("POST /api/v1/kernel/stop", s.handleKernelStop)
	mux.HandleFunc("POST /api/v1/kernel/restart", s.handleKernelRestart)
}

func (s *Server) handleKernelStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.app.Kernel().Status(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeKernelView(status, time.Now().UTC()))
}

func (s *Server) handleKernelStart(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Kernel().Start(r.Context()); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.afterKernelChange(w, r, "kernel started")
}

func (s *Server) handleKernelStop(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Kernel().Stop(r.Context()); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.afterKernelChange(w, r, "kernel stopped")
}

func (s *Server) handleKernelRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Kernel().Restart(r.Context()); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.afterKernelChange(w, r, "kernel restarted")
}

func (s *Server) afterKernelChange(w http.ResponseWriter, r *http.Request, message string) {
	status, err := s.app.Kernel().Status(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "kernel", message, map[string]any{"state": status.State})
	writeJSON(w, http.StatusOK, makeKernelView(status, time.Now().UTC()))
}

var _ = kernel.StateStopped
