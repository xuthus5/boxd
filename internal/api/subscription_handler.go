package api

import (
	"encoding/json"
	"net/http"
	"strings"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type SubscriptionHandler struct {
	manager    *core.SubscriptionManager
	nodeMgr    *core.NodeManager
	configPath string
	instance   restartableInstance
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

func NewSubscriptionHandler(manager *core.SubscriptionManager, nodeMgr *core.NodeManager, configPath string, instance ...restartableInstance) *SubscriptionHandler {
	handler := &SubscriptionHandler{manager: manager, nodeMgr: nodeMgr, configPath: configPath}
	if len(instance) > 0 {
		handler.instance = instance[0]
	}
	return handler
}

func (h *SubscriptionHandler) syncConfig() error {
	if err := syncOutboundsToConfig(h.nodeMgr, h.manager, h.configPath); err != nil {
		return err
	}
	return restartAfterSync(h.instance)
}

func subscriptionSyncErrorMessage(err error) string {
	detail := strings.TrimSpace(err.Error())
	if detail == "" {
		return "subscription refreshed but configuration sync failed"
	}
	return "subscription refreshed but configuration sync failed: " + detail
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
	if err := core.ValidateSubscriptionURL(req.URL); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
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
	if req.URL != "" {
		if err := core.ValidateSubscriptionURL(req.URL); err != nil {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
			return
		}
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

	if err := h.syncConfig(); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorSubscriptionSync, subscriptionSyncErrorMessage(err))
		return
	}

	writeJSONWithMeta(w, http.StatusOK, nil, map[string]string{"action": "refreshing"})
}

func (h *SubscriptionHandler) RefreshAll(w http.ResponseWriter, r *http.Request) {
	failures := h.manager.RefreshAll()
	if failures == nil {
		failures = []core.SubscriptionRefreshFailure{}
	}

	syncErr := h.syncConfig()
	syncMessage := ""
	if syncErr != nil {
		syncMessage = subscriptionSyncErrorMessage(syncErr)
	}
	if len(failures) > 0 || syncErr != nil {
		errorCode := model.ErrorSubscriptionRefresh
		errorMessage := "some subscriptions failed to refresh"
		if syncErr != nil && len(failures) == 0 {
			errorCode = model.ErrorSubscriptionSync
			errorMessage = syncMessage
		}
		data := map[string]any{"failed": failures}
		if syncErr != nil {
			data["sync_error"] = syncMessage
		}
		writeJSONStatus(w, http.StatusOK, model.StatusPartial, data, &model.APIError{
			Code:    errorCode,
			Message: errorMessage,
		}, map[string]any{
			"failed_count": len(failures),
			"sync_failed":  syncErr != nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"failed": []any{}})
}
