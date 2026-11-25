package httpapi

import (
	"net/http"
	"time"

	"github.com/xuthus5/boxd/control"
)

type subscriptionPatch struct {
	Name            *string `json:"name,omitempty"`
	URL             *string `json:"url,omitempty"`
	IntervalSeconds *int    `json:"interval_seconds,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
}

type subscriptionRefreshView struct {
	Subscription subscriptionView `json:"subscription"`
	Added        int              `json:"added"`
	Updated      int              `json:"updated"`
	Removed      int              `json:"removed"`
	NotModified  bool             `json:"not_modified"`
}

func (s *Server) registerSubscriptionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/subscriptions", s.handleSubscriptionList)
	mux.HandleFunc("POST /api/v1/subscriptions", s.handleSubscriptionCreate)
	mux.HandleFunc("GET /api/v1/subscriptions/{id}", s.handleSubscriptionGet)
	mux.HandleFunc("PATCH /api/v1/subscriptions/{id}", s.handleSubscriptionPatch)
	mux.HandleFunc("DELETE /api/v1/subscriptions/{id}", s.handleSubscriptionDelete)
	mux.HandleFunc("POST /api/v1/subscriptions/{id}/refresh", s.handleSubscriptionRefresh)
	mux.HandleFunc("POST /api/v1/subscriptions/refresh", s.handleSubscriptionRefreshAll)
}

func (s *Server) handleSubscriptionList(w http.ResponseWriter, r *http.Request) {
	views, err := s.subscriptionViews(r)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleSubscriptionGet(w http.ResponseWriter, r *http.Request) {
	subscription, err := s.app.Subscriptions().Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeSubscriptionViews([]control.Subscription{subscription}, catalog.Nodes, time.Now().UTC())[0])
}

func (s *Server) handleSubscriptionCreate(w http.ResponseWriter, r *http.Request) {
	var subscription control.Subscription
	if err := decodeJSON(w, r, &subscription); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Subscriptions().Save(r.Context(), subscription)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "subscriptions", "subscription saved", map[string]any{"id": saved.ID})
	writeJSON(w, http.StatusCreated, makeSubscriptionViews([]control.Subscription{saved}, nil, time.Now().UTC())[0])
}

func (s *Server) handleSubscriptionPatch(w http.ResponseWriter, r *http.Request) {
	var patch subscriptionPatch
	if err := decodeJSON(w, r, &patch); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Subscriptions().Patch(r.Context(), r.PathValue("id"), control.SubscriptionPatch{
		Name: patch.Name, URL: patch.URL, IntervalSeconds: patch.IntervalSeconds, Enabled: patch.Enabled,
	})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "subscriptions", "subscription updated", map[string]any{"id": saved.ID, "enabled": saved.Enabled})
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeSubscriptionViews([]control.Subscription{saved}, catalog.Nodes, time.Now().UTC())[0])
}

func (s *Server) handleSubscriptionDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.app.Subscriptions().Delete(r.Context(), id); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "subscriptions", "subscription deleted", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (s *Server) handleSubscriptionRefresh(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.Subscriptions().Refresh(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "subscriptions", "subscription refreshed", map[string]any{
		"id": result.Subscription.ID, "added": result.Added, "updated": result.Updated, "removed": result.Removed,
	})
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, subscriptionRefreshView{
		Subscription: makeSubscriptionViews([]control.Subscription{result.Subscription}, catalog.Nodes, time.Now().UTC())[0],
		Added:        result.Added, Updated: result.Updated, Removed: result.Removed, NotModified: result.NotModified,
	})
}

func (s *Server) handleSubscriptionRefreshAll(w http.ResponseWriter, r *http.Request) {
	results, err := s.app.Subscriptions().RefreshAll(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	views := make([]subscriptionRefreshView, 0, len(results))
	for _, result := range results {
		views = append(views, subscriptionRefreshView{
			Subscription: makeSubscriptionViews([]control.Subscription{result.Subscription}, catalog.Nodes, time.Now().UTC())[0],
			Added:        result.Added, Updated: result.Updated, Removed: result.Removed, NotModified: result.NotModified,
		})
	}
	s.logs.add("info", "subscriptions", "all subscriptions refreshed", map[string]any{"count": len(views)})
	writeJSON(w, http.StatusOK, map[string]any{"results": views})
}

func (s *Server) subscriptionViews(r *http.Request) ([]subscriptionView, error) {
	subscriptions, err := s.app.Subscriptions().List(r.Context())
	if err != nil {
		return nil, err
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		return nil, err
	}
	return makeSubscriptionViews(subscriptions, catalog.Nodes, time.Now().UTC()), nil
}
