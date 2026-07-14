package api

import (
	"encoding/json"
	"net/http"

	"github.com/xuthus5/boxui/internal/core"
	"github.com/xuthus5/boxui/internal/model"
)

type ImportHandler struct {
	nodeManager *core.NodeManager
	subManager  *core.SubscriptionManager
	configPath  string
}

func NewImportHandler(nodeManager *core.NodeManager, subManager *core.SubscriptionManager, configPath string) *ImportHandler {
	return &ImportHandler{nodeManager: nodeManager, subManager: subManager, configPath: configPath}
}

func (h *ImportHandler) syncConfig() {
	_ = syncOutboundsToConfig(h.nodeManager, h.subManager, h.configPath)
}

func (h *ImportHandler) ImportLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Link string `json:"link"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid request body")
		return
	}

	if req.Link == "" {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "link is required")
		return
	}

	result, err := core.ParseProxyLink(req.Link)
	if err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ImportHandler) SaveNode(w http.ResponseWriter, r *http.Request) {
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

	outbound := model.Outbound{
		Tag:    req.Tag,
		Type:   req.Type,
		Server: req.Server,
		Port:   req.Port,
		Raw:    req.Config,
	}

	if err := h.nodeManager.Add(outbound); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, "failed to save node")
		return
	}

	h.syncConfig()

	writeJSON(w, http.StatusOK, nil)
}
