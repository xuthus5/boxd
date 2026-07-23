package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	chi "github.com/go-chi/chi/v5"

	"github.com/xuthus5/boxd/internal/model"
)

// maxCloseConnectionIDs limits bulk close-by-id requests.
const maxCloseConnectionIDs = 500

// parseConnectionIDs parses a comma-separated connection id list.
func parseConnectionIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid connection id %q", part)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) > maxCloseConnectionIDs {
			return nil, fmt.Errorf("too many connection ids (max %d)", maxCloseConnectionIDs)
		}
	}
	return ids, nil
}

// CloseConnection DELETE /api/stats/connections/{id}
func (h *StatsHandler) CloseConnection(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "invalid connection id")
		return
	}
	if h.instance == nil {
		writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, "service not available")
		return
	}
	if !h.instance.CloseConnection(id) {
		writeJSONErrorCode(w, http.StatusNotFound, model.ErrorNotFound, "connection not found")
		return
	}
	writeJSONWithMeta(w, http.StatusOK, nil, map[string]int64{"closed_id": id})
}

// CloseAllConnections DELETE /api/stats/connections
// Optional query filters: outbound, rule, process, ids. Filters are mutually exclusive.
func (h *StatsHandler) CloseAllConnections(w http.ResponseWriter, r *http.Request) {
	if h.instance == nil {
		writeJSONErrorCode(w, http.StatusServiceUnavailable, model.ErrorUnavailable, "service not available")
		return
	}
	outbound := strings.TrimSpace(r.URL.Query().Get("outbound"))
	rule := strings.TrimSpace(r.URL.Query().Get("rule"))
	process := strings.TrimSpace(r.URL.Query().Get("process"))
	idsRaw := strings.TrimSpace(r.URL.Query().Get("ids"))
	filters := 0
	if outbound != "" {
		filters++
	}
	if rule != "" {
		filters++
	}
	if process != "" {
		filters++
	}
	if idsRaw != "" {
		filters++
	}
	if filters > 1 {
		writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "specify only one of outbound, rule, process, or ids")
		return
	}
	var count int
	switch {
	case outbound != "":
		count = h.instance.CloseConnectionsByOutbound(outbound)
	case rule != "":
		count = h.instance.CloseConnectionsByRule(rule)
	case process != "":
		count = h.instance.CloseConnectionsByProcess(process)
	case idsRaw != "":
		ids, err := parseConnectionIDs(idsRaw)
		if err != nil {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, err.Error())
			return
		}
		if len(ids) == 0 {
			writeJSONErrorCode(w, http.StatusBadRequest, model.ErrorInvalidRequest, "ids must include at least one connection id")
			return
		}
		count = h.instance.CloseConnectionsByIDs(ids)
	default:
		count = h.instance.CloseAllConnections()
	}
	writeJSON(w, http.StatusOK, map[string]any{"closed": count, "outbound": outbound, "rule": rule, "process": process, "ids": idsRaw})
}
