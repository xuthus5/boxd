package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type subscriptionSelectorRequest struct {
	Tag string `json:"tag"`
}

// SetSelector PUT /api/subscriptions/{id}/selector 设置 selector 模式下的出口节点。
func (h *SubscriptionHandler) SetSelector(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req subscriptionSelectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}
	subscription := h.manager.Get(id)
	if subscription == nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorSubscriptionNotFound, "subscription not found")
		return
	}
	if err := core.ValidateSubscriptionSelection(*subscription, req.Tag); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}

	updated, err := h.manager.SetSelected(id, req.Tag)
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	if h.nodeMgr != nil {
		if err := syncOutboundsToConfig(h.nodeMgr, h.manager, h.configPath); err != nil {
			restoreErr := h.restoreSelection(id, subscription.Selected)
			slog.Error("subscription selection config sync failed", "id", id, "err", err, "restore_err", restoreErr)
			message := subscriptionSyncErrorMessage(err)
			if restoreErr != nil {
				message += "; restore failed: " + restoreErr.Error()
			}
			writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorSubscriptionSync, message)
			return
		}
	}
	if err := applyRuntimeSelection(h.instance, *updated); err != nil {
		rollbackErr := h.restoreSelection(id, subscription.Selected)
		slog.Error("subscription runtime selection failed", "id", id, "tag", req.Tag, "err", err, "rollback_err", rollbackErr)
		code, message := model.ErrorInternal, fmt.Sprintf("runtime selection failed: %v", err)
		if rollbackErr != nil {
			code, message = model.ErrorSubscriptionSync, message+"; rollback failed: "+rollbackErr.Error()
		}
		writeJSONErrorCode(w, http.StatusInternalServerError, code, message)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *SubscriptionHandler) restoreSelection(id, selected string) error {
	_, selectErr := h.manager.SetSelected(id, selected)
	var syncErr error
	if h.nodeMgr != nil {
		syncErr = syncOutboundsToConfig(h.nodeMgr, h.manager, h.configPath)
	}
	return errors.Join(selectErr, syncErr)
}

func applyRuntimeSelection(instance restartableInstance, subscription model.Subscription) error {
	selector, ok := instance.(subscriptionOutboundSelector)
	if !ok {
		return nil
	}
	target := core.ResolveSubscriptionSelectorTarget(
		core.SubscriptionSelectableTags(subscription),
		subscription.Selected,
	)
	if target == "" {
		return nil
	}
	err := selector.SelectOutbound(subscription.Name, target)
	if errors.Is(err, core.ErrNotRunning) || errors.Is(err, core.ErrNotSelectable) {
		return nil
	}
	return err
}
