package httpapi

import (
	"net/http"

	"github.com/xuthus5/boxd/control"
)

type inboundPatch struct {
	Name    *string                `json:"name,omitempty"`
	Enabled *bool                  `json:"enabled,omitempty"`
	Listen  *control.ListenAddress `json:"listen,omitempty"`
	UDP     *bool                  `json:"udp,omitempty"`
}

func (s *Server) registerInboundRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/inbounds", s.handleInboundList)
	mux.HandleFunc("POST /api/v1/inbounds", s.handleInboundCreate)
	mux.HandleFunc("GET /api/v1/inbounds/{id}", s.handleInboundGet)
	mux.HandleFunc("PATCH /api/v1/inbounds/{id}", s.handleInboundPatch)
	mux.HandleFunc("DELETE /api/v1/inbounds/{id}", s.handleInboundDelete)
}

func (s *Server) handleInboundList(w http.ResponseWriter, r *http.Request) {
	inbounds, err := s.app.Inbounds().List(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	connections, _ := s.app.Observability().Connections(r.Context())
	writeJSON(w, http.StatusOK, makeInboundViews(inbounds, connections))
}

func (s *Server) handleInboundGet(w http.ResponseWriter, r *http.Request) {
	inbound, err := s.app.Inbounds().Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s.inboundView(r, inbound))
}

func (s *Server) handleInboundCreate(w http.ResponseWriter, r *http.Request) {
	var profile control.InboundProfile
	if err := decodeJSON(w, r, &profile); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Inbounds().Save(r.Context(), profile)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "inbound", "inbound saved", map[string]any{"id": saved.ID})
	writeJSON(w, http.StatusCreated, s.inboundView(r, saved))
}

func (s *Server) handleInboundPatch(w http.ResponseWriter, r *http.Request) {
	var patch inboundPatch
	if err := decodeJSON(w, r, &patch); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Inbounds().Patch(r.Context(), r.PathValue("id"), control.InboundPatch{
		Name: patch.Name, Enabled: patch.Enabled, Listen: patch.Listen, UDP: patch.UDP,
	})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "inbound", "inbound updated", map[string]any{"id": saved.ID, "enabled": saved.Enabled})
	writeJSON(w, http.StatusOK, s.inboundView(r, saved))
}

func (s *Server) handleInboundDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.app.Inbounds().Delete(r.Context(), id); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "inbound", "inbound deleted", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (s *Server) inboundView(r *http.Request, inbound control.InboundProfile) inboundView {
	connections, _ := s.app.Observability().Connections(r.Context())
	views := makeInboundViews([]control.InboundProfile{inbound}, connections)
	if len(views) == 0 {
		return inboundView{}
	}
	return views[0]
}
