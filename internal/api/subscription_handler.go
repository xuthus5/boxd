package api

import (
	"encoding/json"
	"net/http"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

type SubscriptionHandler struct {
	manager    *core.SubscriptionManager
	nodeMgr    *core.NodeManager
	configPath string
}

type subscriptionRequest struct {
	Name        string                  `json:"name"`
	URL         string                  `json:"url"`
	IntervalMin int                     `json:"interval_min"`
	URLTest     *model.URLTestOverrides `json:"urltest"`
}

func (r subscriptionRequest) params() core.SubscriptionParams {
	return core.SubscriptionParams{
		Name:        r.Name,
		URL:         r.URL,
		IntervalMin: r.IntervalMin,
		URLTest:     r.URLTest,
	}
}

func NewSubscriptionHandler(manager *core.SubscriptionManager, nodeMgr *core.NodeManager, configPath string) *SubscriptionHandler {
	return &SubscriptionHandler{manager: manager, nodeMgr: nodeMgr, configPath: configPath}
}

func (h *SubscriptionHandler) syncConfig() {
	_ = syncOutboundsToConfig(h.nodeMgr, h.manager, h.configPath)
}

func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	subs, err := h.manager.List()
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to load subscriptions")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req subscriptionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.URL == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "name and url are required")
		return
	}

	if req.IntervalMin <= 0 {
		req.IntervalMin = 60
	}
	if err := core.ValidateURLTestOverrides(req.URLTest); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}

	sub, err := h.manager.Create(req.params())
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, sub)
}

func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sub := h.manager.Get(id)
	if sub == nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorSubscriptionNotFound, "subscription not found")
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

func (h *SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req subscriptionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}

	if err := core.ValidateURLTestOverrides(req.URLTest); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}

	if err := h.manager.Update(id, req.params()); err != nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorSubscriptionNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, nil)
}

func (h *SubscriptionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.manager.Delete(id); err != nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorSubscriptionNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, nil)
}

func (h *SubscriptionHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.manager.Refresh(id); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorSubscriptionRefresh, err.Error())
		return
	}

	h.syncConfig()

	writeJSONWithMeta(w, http.StatusOK, nil, map[string]string{"action": "refreshing"})
}

func (h *SubscriptionHandler) RefreshAll(w http.ResponseWriter, r *http.Request) {
	errs := h.manager.RefreshAll()

	h.syncConfig()

	if len(errs) > 0 {
		writeJSONStatus(w, http.StatusOK, model.StatusPartial, nil, &model.APIError{
			Code:    model.ErrorSubscriptionRefresh,
			Message: "some subscriptions failed to refresh",
		}, map[string]any{
			"failed_count": len(errs),
			"errors":       errs,
		})
		return
	}

	writeJSON(w, http.StatusOK, nil)
}
