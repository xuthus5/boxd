package api

import (
	"net/http"

	"github.com/xuthus5/boxui/internal/model"
)

type ServiceHandler struct {
	instance serviceInstance
}

type serviceInstance interface {
	Start() error
	Stop() error
	Restart() error
	Status() model.ServiceStatus
}

func NewServiceHandler(instance serviceInstance) *ServiceHandler {
	return &ServiceHandler{instance: instance}
}

func (h *ServiceHandler) Start(w http.ResponseWriter, r *http.Request) {
	if err := h.instance.Start(); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	writeJSONWithMeta(w, http.StatusOK, nil, map[string]string{"action": "started"})
}

func (h *ServiceHandler) Stop(w http.ResponseWriter, r *http.Request) {
	if err := h.instance.Stop(); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	writeJSONWithMeta(w, http.StatusOK, nil, map[string]string{"action": "stopped"})
}

func (h *ServiceHandler) Restart(w http.ResponseWriter, r *http.Request) {
	if err := h.instance.Restart(); err != nil {
		writeJSONErrorCode(w, http.StatusInternalServerError, model.ErrorInternal, err.Error())
		return
	}
	writeJSONWithMeta(w, http.StatusOK, nil, map[string]string{"action": "restarted"})
}

func (h *ServiceHandler) Status(w http.ResponseWriter, r *http.Request) {
	info := h.instance.Status()

	writeJSON(w, http.StatusOK, info)
}
