package api

import "net/http"

type HealthHandler struct {
	ready func() error
}

func NewHealthHandler(ready func() error) *HealthHandler {
	return &HealthHandler{ready: ready}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.ready != nil {
		if err := h.ready(); err != nil {
			writeJSONErrorCode(w, http.StatusServiceUnavailable, "not_ready", "service is not ready")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
