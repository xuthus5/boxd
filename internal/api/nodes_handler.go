package api

import (
	"encoding/json"
	"errors"
	"net/http"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxd/internal/core"
	"github.com/xuthus5/boxd/internal/model"
)

type NodesHandler struct {
	nodeManager *core.NodeManager
	subManager  *core.SubscriptionManager
	configPath  string
	instance    restartableInstance
}

func NewNodesHandler(nodeManager *core.NodeManager, subManager *core.SubscriptionManager, configPath string, instance ...restartableInstance) *NodesHandler {
	handler := &NodesHandler{nodeManager: nodeManager, subManager: subManager, configPath: configPath}
	if len(instance) > 0 {
		handler.instance = instance[0]
	}
	return handler
}

func (h *NodesHandler) List(w http.ResponseWriter, r *http.Request) {
	type nodeEntry struct {
		Tag        string `json:"tag"`
		Type       string `json:"type"`
		Server     string `json:"server,omitempty"`
		Port       int    `json:"port,omitempty"`
		Source     string `json:"source"`
		SourceName string `json:"source_name,omitempty"`
	}

	var nodes []nodeEntry

	for _, n := range h.nodeManager.List() {
		nodes = append(nodes, nodeEntry{Tag: n.Tag, Type: n.Type, Server: n.Server, Port: n.Port, Source: "import"})
	}

	subscriptions, err := h.subManager.List()
	if err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to load subscriptions")
		return
	}
	for _, sub := range subscriptions {
		for _, ob := range sub.Outbounds {
			nodes = append(nodes, nodeEntry{Tag: ob.Tag, Type: ob.Type, Server: ob.Server, Port: ob.Port, Source: "subscription", SourceName: sub.Name})
		}
	}

	writeJSON(w, http.StatusOK, nodes)
}

func (h *NodesHandler) Get(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag is required")
		return
	}
	n := h.nodeManager.Get(tag)
	if n == nil {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNodeNotFound, "node not found")
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *NodesHandler) Update(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag is required")
		return
	}
	var req struct {
		Tag    string `json:"tag"`
		Type   string `json:"type"`
		Server string `json:"server"`
		Port   int    `json:"port"`
		Config any    `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}
	if req.Tag == "" || req.Type == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag and type are required")
		return
	}
	updated := model.Outbound{
		Tag:    req.Tag,
		Type:   req.Type,
		Server: req.Server,
		Port:   req.Port,
		Raw:    req.Config,
	}
	if err := h.nodeManager.Update(tag, updated); err != nil {
		switch {
		case errors.Is(err, core.ErrNodeNotFound):
			writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNodeNotFound, "node not found")
		case errors.Is(err, core.ErrNodeTagConflict):
			writeJSONErrorCode(w, http.StatusConflict, model.ErrorConflict, err.Error())
		default:
			writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorNodeUpdateFailed, "failed to update node")
		}
		return
	}
	if err := h.sync(); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorNodeUpdateFailed, "failed to synchronize node configuration: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *NodesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "tag is required")
		return
	}
	if err := h.nodeManager.Delete(tag); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to delete node")
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *NodesHandler) SyncToConfig(w http.ResponseWriter, r *http.Request) {
	if err := h.sync(); err != nil {
		if w != nil {
			writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		}
		return
	}
	if w != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"outbounds":     0,
			"selector_tags": 0,
			"restarted":     h.instance != nil,
		})
	}
}

func (h *NodesHandler) sync() error {
	return syncOutboundsAndRestart(h.nodeManager, h.subManager, h.configPath, h.instance)
}
