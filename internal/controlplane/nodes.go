package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const (
	maxImportNodes      = 500
	defaultDelayURL     = "https://www.gstatic.com/generate_204"
	defaultTestLimit    = 50
	maxTestLimit        = 500
	maxBatchNodeTests   = 100
	maxTestWorkers      = 8
	defaultProbeTimeout = 10 * time.Second
)

// NodeService owns node CRUD, bulk import, and latency measurements.
type NodeService struct {
	profile *ProfileService
	manager kernel.Manager
	results control.TestResultRepository
	codec   control.NodeCodec
	prober  control.NodeProber
	audit   control.AuditRecorder
	now     func() time.Time
}

func (s *NodeService) SetAuditor(recorder control.AuditRecorder) { s.audit = recorder }

var _ control.NodeAPI = (*NodeService)(nil)

func NewNodeService(profile *ProfileService, manager kernel.Manager, results control.TestResultRepository, codec control.NodeCodec, prober control.NodeProber) *NodeService {
	return &NodeService{profile: profile, manager: manager, results: results, codec: codec, prober: prober, now: time.Now}
}

func (s *NodeService) List(ctx context.Context, subscriptionID string) ([]control.Node, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	subscriptionID = strings.TrimSpace(subscriptionID)
	result := make([]control.Node, 0, len(state.Nodes))
	for _, node := range state.Nodes {
		if subscriptionID == "" || node.SubscriptionID == subscriptionID {
			result = append(result, node)
		}
	}
	return result, nil
}

func (s *NodeService) Get(ctx context.Context, id string) (control.Node, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.Node{}, err
	}
	for _, node := range state.Nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return control.Node{}, notFound("node", id)
}

func (s *NodeService) Import(ctx context.Context, nodes []control.Node) (control.NodeImportResult, error) {
	return s.importNodes(ctx, nodes, "")
}

func (s *NodeService) importNodes(ctx context.Context, nodes []control.Node, groupID string) (control.NodeImportResult, error) {
	if len(nodes) == 0 || len(nodes) > maxImportNodes {
		return control.NodeImportResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "nodes.import", fmt.Sprintf("import requires between 1 and %d nodes", maxImportNodes))
	}
	imported := make([]control.Node, len(nodes))
	copy(imported, nodes)
	ids := make(map[string]struct{}, len(imported))
	for index := range imported {
		node := &imported[index]
		node.ID = strings.TrimSpace(node.ID)
		if node.ID == "" {
			node.ID = scopedStableID("node", node.SubscriptionID, strings.ToLower(node.Protocol), node.Server, node.Port, node.Name)
		}
		if _, exists := ids[node.ID]; exists {
			return control.NodeImportResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "nodes.import", fmt.Sprintf("duplicate node ID %q in import", node.ID))
		}
		ids[node.ID] = struct{}{}
	}

	result := control.NodeImportResult{}
	state, err := s.profile.update(ctx, "nodes.import", func(state *control.DesiredState) error {
		groupIndex := -1
		if groupID != "" {
			for index := range state.Groups {
				if state.Groups[index].ID == groupID {
					groupIndex = index
					break
				}
			}
			if groupIndex < 0 {
				return notFound("group", groupID)
			}
		}
		indexByID := make(map[string]int, len(state.Nodes))
		for index, node := range state.Nodes {
			indexByID[node.ID] = index
		}
		for _, node := range imported {
			if index, exists := indexByID[node.ID]; exists {
				state.Nodes[index] = node
				result.Updated++
				continue
			}
			indexByID[node.ID] = len(state.Nodes)
			state.Nodes = append(state.Nodes, node)
			result.Added++
		}
		if groupIndex >= 0 {
			group := &state.Groups[groupIndex]
			existing := make(map[string]struct{}, len(group.Members))
			for _, member := range group.Members {
				existing[member] = struct{}{}
			}
			for _, node := range imported {
				if _, exists := existing[node.ID]; exists {
					continue
				}
				group.Members = append(group.Members, node.ID)
				existing[node.ID] = struct{}{}
			}
			if group.Kind == control.GroupKindSelector && group.Selected == "" && len(group.Members) > 0 {
				group.Selected = group.Members[0]
			}
		}
		return nil
	})
	if err != nil {
		return control.NodeImportResult{}, err
	}
	for _, node := range state.Nodes {
		if _, exists := ids[node.ID]; exists {
			result.Nodes = append(result.Nodes, node)
		}
	}
	return result, nil
}

func (s *NodeService) ImportLinks(ctx context.Context, links []string) (control.NodeImportResult, error) {
	return s.ImportLinksIntoGroup(ctx, links, "")
}

func (s *NodeService) ImportLinksIntoGroup(ctx context.Context, links []string, groupID string) (control.NodeImportResult, error) {
	if s.codec == nil {
		return control.NodeImportResult{}, capabilityUnavailable("nodes.import_links", "node link codec")
	}
	if len(links) == 0 {
		return control.NodeImportResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "nodes.import_links", "at least one node link is required")
	}
	nodes := make([]control.Node, 0, len(links))
	for _, link := range links {
		node, err := s.codec.Decode(ctx, link)
		if err != nil {
			return control.NodeImportResult{}, err
		}
		nodes = append(nodes, node)
	}
	return s.importNodes(ctx, nodes, strings.TrimSpace(groupID))
}

func (s *NodeService) DecodeLink(ctx context.Context, link string) (control.Node, error) {
	if s.codec == nil {
		return control.Node{}, capabilityUnavailable("nodes.decode_link", "node link codec")
	}
	return s.codec.Decode(ctx, link)
}

func (s *NodeService) EncodeLink(ctx context.Context, node control.Node) (string, error) {
	if s.codec == nil {
		return "", capabilityUnavailable("nodes.encode_link", "node link codec")
	}
	return s.codec.Encode(ctx, node)
}

func (s *NodeService) Save(ctx context.Context, node control.Node) (control.Node, error) {
	node.ID = strings.TrimSpace(node.ID)
	if node.ID == "" {
		id, err := s.profile.nextID("node")
		if err != nil {
			return control.Node{}, kernel.WrapError(kernel.ErrCodeInternal, "nodes.save", "generate node ID", err)
		}
		node.ID = id
	}
	state, err := s.profile.update(ctx, "nodes.save", func(state *control.DesiredState) error {
		for index := range state.Nodes {
			if state.Nodes[index].ID == node.ID {
				state.Nodes[index] = node
				return nil
			}
		}
		state.Nodes = append(state.Nodes, node)
		return nil
	})
	if err != nil {
		return control.Node{}, err
	}
	for _, saved := range state.Nodes {
		if saved.ID == node.ID {
			return saved, nil
		}
	}
	return control.Node{}, notFound("node", node.ID)
}

func (s *NodeService) Delete(ctx context.Context, id string) error {
	_, err := s.profile.update(ctx, "nodes.delete", func(state *control.DesiredState) error {
		for index := range state.Nodes {
			if state.Nodes[index].ID != id {
				continue
			}
			state.Nodes = append(state.Nodes[:index], state.Nodes[index+1:]...)
			return nil
		}
		return notFound("node", id)
	})
	return err
}

func (s *NodeService) Delay(ctx context.Context, id, url string) (control.NodeDelay, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return control.NodeDelay{}, err
	}
	url = strings.TrimSpace(url)
	if url == "" {
		url = defaultDelayURL
	}
	testedAt := s.now().UTC()
	latency, delayErr := s.manager.OutboundDelay(ctx, control.NodeTag(id), url)
	result := control.NodeTestResult{
		NodeID:   id,
		Kind:     "delay",
		URL:      url,
		Latency:  latency,
		TestedAt: testedAt,
	}
	if delayErr != nil {
		result.Error = delayErr.Error()
	}
	persistErr := s.persistTestResult(ctx, result)
	if delayErr != nil {
		return control.NodeDelay{}, errors.Join(delayErr, persistErr)
	}
	if persistErr != nil {
		return control.NodeDelay{}, persistErr
	}
	recordAudit(ctx, s.audit, "nodes.delay", map[string]any{"node_id": id, "url": url})
	return control.NodeDelay{NodeID: id, URL: url, Latency: latency, TestedAt: testedAt}, nil
}

// Test measures one or more nodes with a selected probe. Individual probe
// failures are returned per node so a batch can still report successful peers.
func (s *NodeService) Test(ctx context.Context, request control.NodeTestRequest) ([]control.NodeTestOutcome, error) {
	kind := control.NodeTestKind(strings.ToLower(strings.TrimSpace(string(request.Kind))))
	if kind == "" {
		kind = control.NodeTestKindURL
	}
	switch kind {
	case control.NodeTestKindURL, control.NodeTestKindTCP, control.NodeTestKindICMP:
	default:
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "nodes.test", "unsupported node test method "+string(kind)).
			WithHint("use url, tcp, or icmp")
	}
	if kind != control.NodeTestKindURL && s.prober == nil {
		return nil, capabilityUnavailable("nodes.test", "host network prober")
	}
	ids := make([]string, 0, len(request.NodeIDs))
	seen := make(map[string]struct{}, len(request.NodeIDs))
	for _, id := range request.NodeIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "nodes.test", "at least one node ID is required")
	}
	if len(ids) > maxBatchNodeTests {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "nodes.test", fmt.Sprintf("at most %d nodes can be tested at once", maxBatchNodeTests))
	}

	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	nodeByID := make(map[string]control.Node, len(state.Nodes))
	for _, node := range state.Nodes {
		nodeByID[node.ID] = node
	}
	nodes := make([]control.Node, 0, len(ids))
	for _, id := range ids {
		node, found := nodeByID[id]
		if !found {
			return nil, notFound("node", id)
		}
		nodes = append(nodes, node)
	}

	url := strings.TrimSpace(request.URL)
	if kind == control.NodeTestKindURL && url == "" {
		url = defaultDelayURL
	}
	outcomes := make([]control.NodeTestOutcome, len(nodes))
	jobs := make(chan int)
	var waitGroup sync.WaitGroup
	workerCount := min(maxTestWorkers, len(nodes))
	for worker := 0; worker < workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for index := range jobs {
				outcomes[index] = s.testOne(ctx, nodes[index], kind, url)
			}
		}()
	}
	for index := range nodes {
		jobs <- index
	}
	close(jobs)
	waitGroup.Wait()

	recordAudit(ctx, s.audit, "nodes.test", map[string]any{"kind": string(kind), "count": len(outcomes)})
	return outcomes, nil
}

func (s *NodeService) testOne(ctx context.Context, node control.Node, kind control.NodeTestKind, url string) control.NodeTestOutcome {
	probeContext, cancel := context.WithTimeout(ctx, defaultProbeTimeout)
	defer cancel()

	target := node.Server
	switch kind {
	case control.NodeTestKindURL:
		target = url
	case control.NodeTestKindTCP:
		target = net.JoinHostPort(node.Server, fmt.Sprintf("%d", node.Port))
	}
	testedAt := s.now().UTC()
	var latency time.Duration
	var probeErr error
	switch kind {
	case control.NodeTestKindURL:
		latency, probeErr = s.manager.OutboundDelay(probeContext, control.NodeTag(node.ID), target)
	case control.NodeTestKindTCP:
		latency, probeErr = s.prober.TCP(probeContext, node.Server, node.Port)
	case control.NodeTestKindICMP:
		latency, probeErr = s.prober.ICMP(probeContext, node.Server)
	}

	result := control.NodeTestOutcome{NodeID: node.ID, Kind: kind, Target: target, Latency: latency, TestedAt: testedAt}
	history := control.NodeTestResult{NodeID: node.ID, Kind: string(kind), URL: target, Latency: latency, TestedAt: testedAt}
	if probeErr != nil {
		result.Error = probeErr.Error()
		history.Error = result.Error
	}
	if persistErr := s.persistTestResult(ctx, history); persistErr != nil {
		result.Error = errors.Join(probeErr, persistErr).Error()
	}
	return result
}

func (s *NodeService) TestResults(ctx context.Context, nodeID string, limit int) ([]control.NodeTestResult, error) {
	if s.results == nil {
		return nil, capabilityUnavailable("nodes.test_results", "test result repository")
	}
	if _, err := s.Get(ctx, nodeID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultTestLimit
	}
	if limit > maxTestLimit {
		limit = maxTestLimit
	}
	return s.results.List(ctx, nodeID, limit)
}

func (s *NodeService) persistTestResult(ctx context.Context, result control.NodeTestResult) error {
	if s.results == nil {
		return nil
	}
	if err := s.results.Save(ctx, result); err != nil {
		return wrapStoreError("nodes.test_result", "save node test result", err)
	}
	return nil
}
