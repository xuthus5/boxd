package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
)

type importLinksRequest struct {
	Links   []string `json:"links"`
	GroupID string   `json:"group_id,omitempty"`
}

type nodeImportView struct {
	Added        int        `json:"added"`
	Updated      int        `json:"updated"`
	Nodes        []nodeView `json:"nodes"`
	GroupWarning string     `json:"group_warning,omitempty"`
}

type nodeDelayView struct {
	NodeID    string    `json:"node_id"`
	URL       string    `json:"url"`
	LatencyMS int64     `json:"latency_ms"`
	TestedAt  time.Time `json:"tested_at"`
}

type nodeTestResultView struct {
	NodeID    string               `json:"node_id"`
	NodeName  string               `json:"node_name"`
	Kind      control.NodeTestKind `json:"kind"`
	Target    string               `json:"target"`
	LatencyMS int64                `json:"latency_ms"`
	Error     string               `json:"error,omitempty"`
	TestedAt  time.Time            `json:"tested_at"`
}

type decodeLinkRequest struct {
	Link string `json:"link"`
}

type selectMemberRequest struct {
	NodeID string `json:"node_id"`
}

func (s *Server) registerEgressRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/egress", s.handleEgressCatalog)
	mux.HandleFunc("GET /api/v1/nodes", s.handleNodeList)
	mux.HandleFunc("GET /api/v1/nodes/{id}", s.handleNodeGet)
	mux.HandleFunc("PUT /api/v1/nodes/{id}", s.handleNodeUpdate)
	mux.HandleFunc("POST /api/v1/nodes/import-links", s.handleNodeImportLinks)
	mux.HandleFunc("POST /api/v1/nodes/import", s.handleNodeImport)
	mux.HandleFunc("POST /api/v1/nodes/decode", s.handleNodeDecode)
	mux.HandleFunc("POST /api/v1/nodes/save", s.handleNodeSave)
	mux.HandleFunc("POST /api/v1/nodes/test", s.handleNodeTest)
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", s.handleNodeDelete)
	mux.HandleFunc("POST /api/v1/nodes/{id}/delay", s.handleNodeDelay)
	mux.HandleFunc("GET /api/v1/nodes/{id}/results", s.handleNodeResults)
	mux.HandleFunc("GET /api/v1/groups", s.handleGroupList)
	mux.HandleFunc("POST /api/v1/groups", s.handleGroupSave)
	mux.HandleFunc("DELETE /api/v1/groups/{id}", s.handleGroupDelete)
	mux.HandleFunc("POST /api/v1/groups/{id}/select", s.handleGroupSelect)
	mux.HandleFunc("GET /api/v1/chains", s.handleChainList)
	mux.HandleFunc("POST /api/v1/chains", s.handleChainSave)
	mux.HandleFunc("DELETE /api/v1/chains/{id}", s.handleChainDelete)
}

func (s *Server) handleEgressCatalog(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	view, err := s.catalogView(catalog, r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleNodeList(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.app.Nodes().List(r.Context(), r.URL.Query().Get("subscription_id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeNodeViews(nodes, catalogGroupMap(catalog.Groups), catalog.Groups))
}

func (s *Server) handleNodeGet(w http.ResponseWriter, r *http.Request) {
	node, err := s.app.Nodes().Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (s *Server) handleNodeImportLinks(w http.ResponseWriter, r *http.Request) {
	var request importLinksRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	request.Links = compactStrings(request.Links)
	result, err := s.app.Nodes().ImportLinksIntoGroup(r.Context(), request.Links, strings.TrimSpace(request.GroupID))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	view := nodeImportView{Added: result.Added, Updated: result.Updated}
	view.Nodes = makeNodeViews(result.Nodes, catalogGroupMap(catalog.Groups), catalog.Groups)
	s.logs.add("info", "nodes", "node links imported", map[string]any{"added": result.Added, "updated": result.Updated})
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleNodeImport(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Nodes []control.Node `json:"nodes"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	result, err := s.app.Nodes().Import(r.Context(), request.Nodes)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	view := nodeImportView{Added: result.Added, Updated: result.Updated}
	view.Nodes = makeNodeViews(result.Nodes, catalogGroupMap(catalog.Groups), catalog.Groups)
	s.logs.add("info", "nodes", "nodes imported", map[string]any{"added": result.Added, "updated": result.Updated})
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleNodeDecode(w http.ResponseWriter, r *http.Request) {
	var request decodeLinkRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	node, err := s.app.Nodes().DecodeLink(r.Context(), request.Link)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeNodeViews([]control.Node{node}, nil, nil)[0])
}

func (s *Server) handleNodeSave(w http.ResponseWriter, r *http.Request) {
	var node control.Node
	if err := decodeJSON(w, r, &node); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Nodes().Save(r.Context(), node)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeNodeViews([]control.Node{saved}, catalogGroupMap(catalog.Groups), catalog.Groups)[0])
}

func (s *Server) handleNodeUpdate(w http.ResponseWriter, r *http.Request) {
	var node control.Node
	if err := decodeJSON(w, r, &node); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	node.ID = r.PathValue("id")
	saved, err := s.app.Nodes().Save(r.Context(), node)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "nodes", "node updated", map[string]any{"id": saved.ID})
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeNodeViews([]control.Node{saved}, catalogGroupMap(catalog.Groups), catalog.Groups)[0])
}

func (s *Server) handleNodeDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.app.Nodes().Delete(r.Context(), id); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "nodes", "node deleted", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (s *Server) handleNodeDelay(w http.ResponseWriter, r *http.Request) {
	var request struct {
		URL string `json:"url,omitempty"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &request); err != nil {
			writeError(w, r, err.(*apiError))
			return
		}
	}
	delay, err := s.app.Nodes().Delay(r.Context(), r.PathValue("id"), request.URL)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, nodeDelayView{NodeID: delay.NodeID, URL: delay.URL, LatencyMS: delay.Latency.Milliseconds(), TestedAt: delay.TestedAt})
}

func (s *Server) handleNodeTest(w http.ResponseWriter, r *http.Request) {
	var request control.NodeTestRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	outcomes, err := s.app.Nodes().Test(r.Context(), request)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	nodes, err := s.app.Nodes().List(r.Context(), "")
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	names := make(map[string]string, len(nodes))
	for _, node := range nodes {
		names[node.ID] = node.Name
	}
	views := make([]nodeTestResultView, 0, len(outcomes))
	for _, outcome := range outcomes {
		views = append(views, nodeTestResultView{
			NodeID: outcome.NodeID, NodeName: names[outcome.NodeID], Kind: outcome.Kind, Target: outcome.Target,
			LatencyMS: outcome.Latency.Milliseconds(), Error: outcome.Error, TestedAt: outcome.TestedAt,
		})
	}
	s.logs.add("info", "nodes", "node reachability tested", map[string]any{"kind": request.Kind, "count": len(views)})
	writeJSON(w, http.StatusOK, map[string]any{"kind": request.Kind, "results": views})
}

func (s *Server) handleNodeResults(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 50, 500)
	results, err := s.app.Nodes().TestResults(r.Context(), r.PathValue("id"), limit)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleGroupList(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeGroupViews(catalog.Groups, catalog.Nodes))
}

func (s *Server) handleGroupSave(w http.ResponseWriter, r *http.Request) {
	var group control.Group
	if err := decodeJSON(w, r, &group); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Egress().SaveGroup(r.Context(), group)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "groups", "strategy group saved", map[string]any{"id": saved.ID})
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleGroupDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.app.Egress().DeleteGroup(r.Context(), id); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "groups", "strategy group deleted", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (s *Server) handleGroupSelect(w http.ResponseWriter, r *http.Request) {
	var request selectMemberRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	if err := s.app.Egress().SelectGroupMember(r.Context(), r.PathValue("id"), request.NodeID); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeGroupViews(catalog.Groups, catalog.Nodes))
}

func (s *Server) handleChainList(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.app.Egress().Catalog(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	routes, err := s.app.Routes().Get(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, makeChainViews(catalog.Chains, catalog.Nodes, routes))
}

func (s *Server) handleChainSave(w http.ResponseWriter, r *http.Request) {
	var chain control.ProxyChain
	if err := decodeJSON(w, r, &chain); err != nil {
		writeError(w, r, err.(*apiError))
		return
	}
	saved, err := s.app.Egress().SaveChain(r.Context(), chain)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "chains", "proxy chain saved", map[string]any{"id": saved.ID})
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleChainDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.app.Egress().DeleteChain(r.Context(), id); err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	s.logs.add("info", "chains", "proxy chain deleted", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (s *Server) catalogView(catalog control.EgressCatalog, ctx context.Context) (map[string]any, error) {
	routes, err := s.app.Routes().Get(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"nodes":  makeNodeViews(catalog.Nodes, catalogGroupMap(catalog.Groups), catalog.Groups),
		"groups": makeGroupViews(catalog.Groups, catalog.Nodes),
		"chains": makeChainViews(catalog.Chains, catalog.Nodes, routes),
	}, nil
}

func catalogGroupMap(groups []control.Group) map[string][]string {
	result := make(map[string][]string)
	for _, group := range groups {
		for _, member := range group.Members {
			result[member] = append(result[member], group.ID)
		}
	}
	return result
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
