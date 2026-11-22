package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const (
	defaultDelayURL  = "https://www.gstatic.com/generate_204"
	defaultTestLimit = 50
	maxTestLimit     = 500
)

// NodeService owns node CRUD, bulk import, and latency measurements.
type NodeService struct {
	profile *ProfileService
	manager kernel.Manager
	results control.TestResultRepository
	now     func() time.Time
}

var _ control.NodeAPI = (*NodeService)(nil)

func NewNodeService(profile *ProfileService, manager kernel.Manager, results control.TestResultRepository) *NodeService {
	return &NodeService{profile: profile, manager: manager, results: results, now: time.Now}
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
	if len(nodes) == 0 {
		return control.NodeImportResult{}, kernel.NewError(kernel.ErrCodeInvalidConfig, "nodes.import", "at least one node is required")
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
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
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

func (s *NodeService) Save(ctx context.Context, node control.Node) (control.Node, error) {
	node.ID = strings.TrimSpace(node.ID)
	if node.ID == "" {
		id, err := s.profile.nextID("node")
		if err != nil {
			return control.Node{}, kernel.WrapError(kernel.ErrCodeInternal, "nodes.save", "generate node ID", err)
		}
		node.ID = id
	}
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
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
	_, err := s.profile.update(ctx, func(state *control.DesiredState) error {
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
	return control.NodeDelay{NodeID: id, URL: url, Latency: latency, TestedAt: testedAt}, nil
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
