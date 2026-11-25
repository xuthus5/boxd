package httpapi

import (
	"net/http"
	"time"

	"github.com/xuthus5/boxd/control"
)

type ruleSetPatch struct {
	Enabled *bool `json:"enabled,omitempty"`
}

func (s *Server) registerRuleSetRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/rulesets", s.handleRuleSetList)
	mux.HandleFunc("POST /api/v1/rulesets", s.handleRuleSetCreate)
	mux.HandleFunc("PATCH /api/v1/rulesets/{name}", s.handleRuleSetPatch)
	mux.HandleFunc("DELETE /api/v1/rulesets/{name}", s.handleRuleSetDelete)
	mux.HandleFunc("POST /api/v1/rulesets/{name}/refresh", s.handleRuleSetRefresh)
	mux.HandleFunc("POST /api/v1/rulesets/refresh", s.handleRuleSetRefreshAll)
}

func (s *Server) handleRuleSetList(w http.ResponseWriter, r *http.Request) {
	views, err := s.ruleSetViews(r)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleRuleSetCreate(w http.ResponseWriter, r *http.Request) {
	var definition control.RuleSetDefinition
	if err := decodeJSON(w, r, &definition); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.RuleSets().Save(r.Context(), definition)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "rulesets", "rule set saved", map[string]any{"name": saved.Name})
	writeJSON(w, http.StatusCreated, makeRuleSetViews([]control.RuleSet{saved}, time.Now().UTC())[0])
}

func (s *Server) handleRuleSetPatch(w http.ResponseWriter, r *http.Request) {
	var patch ruleSetPatch
	if err := decodeJSON(w, r, &patch); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.RuleSets().Patch(r.Context(), r.PathValue("name"), control.RuleSetPatch{Enabled: patch.Enabled})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "rulesets", "rule set updated", map[string]any{"name": saved.Name, "enabled": saved.Enabled})
	writeJSON(w, http.StatusOK, makeRuleSetViews([]control.RuleSet{saved}, time.Now().UTC())[0])
}

func (s *Server) handleRuleSetDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.app.RuleSets().Delete(r.Context(), name); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "rulesets", "rule set deleted", map[string]any{"name": name})
	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func (s *Server) handleRuleSetRefresh(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	refreshed, err := s.app.RuleSets().Refresh(r.Context(), name)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "rulesets", "rule set refreshed", map[string]any{"name": refreshed.Name})
	writeJSON(w, http.StatusOK, makeRuleSetViews([]control.RuleSet{refreshed}, time.Now().UTC())[0])
}

func (s *Server) handleRuleSetRefreshAll(w http.ResponseWriter, r *http.Request) {
	refreshed, err := s.app.RuleSets().RefreshAll(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "rulesets", "all rule sets refreshed", map[string]any{"count": len(refreshed)})
	writeJSON(w, http.StatusOK, makeRuleSetViews(refreshed, time.Now().UTC()))
}

func (s *Server) ruleSetViews(r *http.Request) ([]ruleSetView, error) {
	ruleSets, err := s.app.RuleSets().List(r.Context())
	if err != nil {
		return nil, err
	}
	return makeRuleSetViews(ruleSets, time.Now().UTC()), nil
}
