package httpapi

import (
	"net/http"

	"github.com/xuthus5/boxd/control"
)

type routeModeRequest struct {
	Mode control.RouteMode `json:"mode"`
}

type routeEgressRequest struct {
	Kind control.EgressKind `json:"kind"`
	ID   string             `json:"id,omitempty"`
}

type routeRulePatch struct {
	Name     *string              `json:"name,omitempty"`
	Disabled *bool                `json:"disabled,omitempty"`
	Match    *control.RouteMatch  `json:"match,omitempty"`
	Action   *control.RouteAction `json:"action,omitempty"`
	Egress   *control.EgressRef   `json:"egress,omitempty"`
}

func (s *Server) registerRouteRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/routes", s.handleRouteGet)
	mux.HandleFunc("PATCH /api/v1/routes/mode", s.handleRouteMode)
	mux.HandleFunc("PATCH /api/v1/routes/default-egress", s.handleRouteDefaultEgress)
	mux.HandleFunc("PATCH /api/v1/routes/presets", s.handleRoutePresets)
	mux.HandleFunc("POST /api/v1/routes/rules", s.handleRouteRuleSave)
	mux.HandleFunc("PATCH /api/v1/routes/rules/{id}", s.handleRouteRulePatch)
	mux.HandleFunc("DELETE /api/v1/routes/rules/{id}", s.handleRouteRuleDelete)
	mux.HandleFunc("PUT /api/v1/routes/rules", s.handleRouteRulesReplace)
}

func (s *Server) handleRouteGet(w http.ResponseWriter, r *http.Request) {
	view, err := s.currentRouteView(r)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleRouteMode(w http.ResponseWriter, r *http.Request) {
	var request routeModeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	if _, err := s.app.Routes().SetMode(r.Context(), request.Mode); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "routes", "route mode changed", map[string]any{"mode": request.Mode})
	view, err := s.currentRouteView(r)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleRouteDefaultEgress(w http.ResponseWriter, r *http.Request) {
	var request routeEgressRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	ref := control.EgressRef{Kind: request.Kind, ID: request.ID}
	if _, err := s.app.Routes().SetDefaultEgress(r.Context(), ref); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "routes", "default egress changed", map[string]any{"kind": request.Kind, "id": request.ID})
	view, err := s.currentRouteView(r)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleRoutePresets(w http.ResponseWriter, r *http.Request) {
	var presets control.RoutePresets
	if err := decodeJSON(w, r, &presets); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	if _, err := s.app.Routes().SetPresets(r.Context(), presets); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "routes", "routing presets changed", map[string]any{
		"bypass_lan": presets.BypassLAN, "china_direct": presets.ChinaDirect, "ad_block": presets.AdBlock,
	})
	view, err := s.currentRouteView(r)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleRouteRuleSave(w http.ResponseWriter, r *http.Request) {
	var rule control.RouteRule
	if err := decodeJSON(w, r, &rule); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Routes().SaveRule(r.Context(), rule)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "routes", "route rule saved", map[string]any{"id": saved.ID})
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleRouteRulePatch(w http.ResponseWriter, r *http.Request) {
	var patch routeRulePatch
	if err := decodeJSON(w, r, &patch); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Routes().PatchRule(r.Context(), r.PathValue("id"), control.RouteRulePatch{
		Name: patch.Name, Disabled: patch.Disabled, Match: patch.Match, Action: patch.Action, Egress: patch.Egress,
	})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "routes", "route rule updated", map[string]any{"id": saved.ID, "disabled": saved.Disabled})
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleRouteRuleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.app.Routes().DeleteRule(r.Context(), id); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "routes", "route rule deleted", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (s *Server) handleRouteRulesReplace(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Rules []control.RouteRule `json:"rules"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	rules, err := s.app.Routes().ReplaceRules(r.Context(), request.Rules)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "routes", "route rules replaced", map[string]any{"count": len(rules)})
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) currentRouteView(r *http.Request) (routeView, error) {
	routes, err := s.app.Routes().Get(r.Context())
	if err != nil {
		return routeView{}, err
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		return routeView{}, err
	}
	return makeRouteViews(routes, catalog), nil
}
