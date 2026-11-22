package controlplane

import (
	"context"
	"fmt"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

type SetupService struct {
	profile *ProfileService
	manager kernel.Manager
}

var _ control.SetupAPI = (*SetupService)(nil)

func NewSetupService(profile *ProfileService, manager kernel.Manager) *SetupService {
	return &SetupService{profile: profile, manager: manager}
}

func (s *SetupService) EnsureDefaults(ctx context.Context) (control.DesiredState, error) {
	if s.profile == nil {
		return control.DesiredState{}, kernel.NewError(kernel.ErrCodeInternal, "setup.ensure_defaults", "profile service is unavailable")
	}
	return s.profile.EnsureDefaults(ctx)
}

func (s *SetupService) Capabilities(context.Context) (kernel.EngineDescriptor, error) {
	return kernel.EngineDescriptor{
		Kind:         s.manager.Kind(),
		Version:      s.manager.Version(),
		Capabilities: s.manager.Capabilities(),
	}, nil
}

type InboundService struct {
	profile *ProfileService
}

var _ control.InboundAPI = (*InboundService)(nil)

func NewInboundService(profile *ProfileService) *InboundService {
	return &InboundService{profile: profile}
}

func (s *InboundService) List(ctx context.Context) ([]control.InboundProfile, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return append([]control.InboundProfile(nil), state.Inbounds...), nil
}

func (s *InboundService) Get(ctx context.Context, id string) (control.InboundProfile, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.InboundProfile{}, err
	}
	for _, inbound := range state.Inbounds {
		if inbound.ID == id {
			return inbound, nil
		}
	}
	return control.InboundProfile{}, notFound("inbound", id)
}

func (s *InboundService) Save(ctx context.Context, profile control.InboundProfile) (control.InboundProfile, error) {
	if profile.ID == "" {
		id, err := s.profile.nextID("inbound")
		if err != nil {
			return control.InboundProfile{}, kernel.WrapError(kernel.ErrCodeInternal, "inbound.save", "generate inbound ID", err)
		}
		profile.ID = id
	}
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Inbounds {
			if state.Inbounds[index].ID == profile.ID {
				state.Inbounds[index] = profile
				return nil
			}
		}
		state.Inbounds = append(state.Inbounds, profile)
		return nil
	})
	if err != nil {
		return control.InboundProfile{}, err
	}
	for _, saved := range state.Inbounds {
		if saved.ID == profile.ID {
			return saved, nil
		}
	}
	return control.InboundProfile{}, notFound("inbound", profile.ID)
}

func (s *InboundService) Delete(ctx context.Context, id string) error {
	_, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Inbounds {
			if state.Inbounds[index].ID != id {
				continue
			}
			state.Inbounds = append(state.Inbounds[:index], state.Inbounds[index+1:]...)
			return nil
		}
		return notFound("inbound", id)
	})
	return err
}

type EgressService struct {
	profile *ProfileService
	manager kernel.Manager
}

var _ control.EgressAPI = (*EgressService)(nil)

func NewEgressService(profile *ProfileService, manager kernel.Manager) *EgressService {
	return &EgressService{profile: profile, manager: manager}
}

func (s *EgressService) Catalog(ctx context.Context) (control.EgressCatalog, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.EgressCatalog{}, err
	}
	return control.EgressCatalog{
		Nodes:  append([]control.Node(nil), state.Nodes...),
		Groups: append([]control.Group(nil), state.Groups...),
		Chains: append([]control.ProxyChain(nil), state.Chains...),
	}, nil
}

func (s *EgressService) SaveNode(ctx context.Context, node control.Node) (control.Node, error) {
	if node.ID == "" {
		id, err := s.profile.nextID("node")
		if err != nil {
			return control.Node{}, kernel.WrapError(kernel.ErrCodeInternal, "egress.save_node", "generate node ID", err)
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

func (s *EgressService) DeleteNode(ctx context.Context, id string) error {
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

func (s *EgressService) SaveGroup(ctx context.Context, group control.Group) (control.Group, error) {
	if group.ID == "" {
		id, err := s.profile.nextID("group")
		if err != nil {
			return control.Group{}, kernel.WrapError(kernel.ErrCodeInternal, "egress.save_group", "generate group ID", err)
		}
		group.ID = id
	}
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Groups {
			if state.Groups[index].ID == group.ID {
				state.Groups[index] = group
				return nil
			}
		}
		state.Groups = append(state.Groups, group)
		return nil
	})
	if err != nil {
		return control.Group{}, err
	}
	for _, saved := range state.Groups {
		if saved.ID == group.ID {
			return saved, nil
		}
	}
	return control.Group{}, notFound("group", group.ID)
}

func (s *EgressService) DeleteGroup(ctx context.Context, id string) error {
	_, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Groups {
			if state.Groups[index].ID != id {
				continue
			}
			state.Groups = append(state.Groups[:index], state.Groups[index+1:]...)
			return nil
		}
		return notFound("group", id)
	})
	return err
}

func (s *EgressService) SaveChain(ctx context.Context, chain control.ProxyChain) (control.ProxyChain, error) {
	if chain.ID == "" {
		id, err := s.profile.nextID("chain")
		if err != nil {
			return control.ProxyChain{}, kernel.WrapError(kernel.ErrCodeInternal, "egress.save_chain", "generate chain ID", err)
		}
		chain.ID = id
	}
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Chains {
			if state.Chains[index].ID == chain.ID {
				state.Chains[index] = chain
				return nil
			}
		}
		state.Chains = append(state.Chains, chain)
		return nil
	})
	if err != nil {
		return control.ProxyChain{}, err
	}
	for _, saved := range state.Chains {
		if saved.ID == chain.ID {
			return saved, nil
		}
	}
	return control.ProxyChain{}, notFound("chain", chain.ID)
}

func (s *EgressService) DeleteChain(ctx context.Context, id string) error {
	_, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Chains {
			if state.Chains[index].ID != id {
				continue
			}
			state.Chains = append(state.Chains[:index], state.Chains[index+1:]...)
			return nil
		}
		return notFound("chain", id)
	})
	return err
}

func (s *EgressService) SelectGroupMember(ctx context.Context, groupID, nodeID string) error {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return err
	}
	foundGroup := false
	for _, group := range state.Groups {
		if group.ID != groupID {
			continue
		}
		foundGroup = true
		if group.Kind != control.GroupKindSelector {
			return kernel.NewError(kernel.ErrCodeInvalidConfig, "egress.select_group_member", fmt.Sprintf("group %q is not a selector", groupID))
		}
		member := false
		for _, id := range group.Members {
			if id == nodeID {
				member = true
				break
			}
		}
		if !member {
			return kernel.NewError(kernel.ErrCodeNotFound, "egress.select_group_member", fmt.Sprintf("node %q is not a member of group %q", nodeID, groupID))
		}
		break
	}
	if !foundGroup {
		return notFound("group", groupID)
	}
	return s.manager.SelectOutbound(ctx, control.GroupTag(groupID), control.NodeTag(nodeID))
}

type RouteService struct {
	profile *ProfileService
}

var _ control.RouteAPI = (*RouteService)(nil)

func NewRouteService(profile *ProfileService) *RouteService {
	return &RouteService{profile: profile}
}

func (s *RouteService) Get(ctx context.Context) (control.RouteProfile, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.RouteProfile{}, err
	}
	return state.Routes, nil
}

func (s *RouteService) SetMode(ctx context.Context, mode control.RouteMode) (control.RouteProfile, error) {
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		state.Routes.Mode = mode
		return nil
	})
	if err != nil {
		return control.RouteProfile{}, err
	}
	return state.Routes, nil
}

func (s *RouteService) SetDefaultEgress(ctx context.Context, egress control.EgressRef) (control.RouteProfile, error) {
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		state.Routes.DefaultEgress = egress
		return nil
	})
	if err != nil {
		return control.RouteProfile{}, err
	}
	return state.Routes, nil
}

func (s *RouteService) SetPresets(ctx context.Context, presets control.RoutePresets) (control.RouteProfile, error) {
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		state.Routes.Presets = presets
		return nil
	})
	if err != nil {
		return control.RouteProfile{}, err
	}
	return state.Routes, nil
}

func (s *RouteService) SaveRule(ctx context.Context, rule control.RouteRule) (control.RouteRule, error) {
	if rule.ID == "" {
		id, err := s.profile.nextID("rule")
		if err != nil {
			return control.RouteRule{}, kernel.WrapError(kernel.ErrCodeInternal, "routes.save_rule", "generate rule ID", err)
		}
		rule.ID = id
	}
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Routes.Rules {
			if state.Routes.Rules[index].ID == rule.ID {
				state.Routes.Rules[index] = rule
				return nil
			}
		}
		state.Routes.Rules = append(state.Routes.Rules, rule)
		return nil
	})
	if err != nil {
		return control.RouteRule{}, err
	}
	for _, saved := range state.Routes.Rules {
		if saved.ID == rule.ID {
			return saved, nil
		}
	}
	return control.RouteRule{}, notFound("route rule", rule.ID)
}

func (s *RouteService) DeleteRule(ctx context.Context, id string) error {
	_, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		for index := range state.Routes.Rules {
			if state.Routes.Rules[index].ID != id {
				continue
			}
			state.Routes.Rules = append(state.Routes.Rules[:index], state.Routes.Rules[index+1:]...)
			return nil
		}
		return notFound("route rule", id)
	})
	return err
}

func (s *RouteService) ReplaceRules(ctx context.Context, rules []control.RouteRule) ([]control.RouteRule, error) {
	for index := range rules {
		if rules[index].ID != "" {
			continue
		}
		id, err := s.profile.nextID("rule")
		if err != nil {
			return nil, kernel.WrapError(kernel.ErrCodeInternal, "routes.replace_rules", "generate rule ID", err)
		}
		rules[index].ID = id
	}
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		state.Routes.Rules = append([]control.RouteRule(nil), rules...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return append([]control.RouteRule(nil), state.Routes.Rules...), nil
}

type DNSService struct {
	profile *ProfileService
	manager kernel.Manager
}

var _ control.DNSAPI = (*DNSService)(nil)

func NewDNSService(profile *ProfileService, manager kernel.Manager) *DNSService {
	return &DNSService{profile: profile, manager: manager}
}

func (s *DNSService) Get(ctx context.Context) (control.DNSProfile, error) {
	state, err := s.profile.Snapshot(ctx)
	if err != nil {
		return control.DNSProfile{}, err
	}
	return state.DNS, nil
}

func (s *DNSService) Update(ctx context.Context, profile control.DNSProfile) (control.DNSProfile, error) {
	state, err := s.profile.update(ctx, func(state *control.DesiredState) error {
		state.DNS = profile
		return nil
	})
	if err != nil {
		return control.DNSProfile{}, err
	}
	return state.DNS, nil
}

func (s *DNSService) Flush(ctx context.Context) error {
	return s.manager.FlushDNS(ctx)
}

type ObservabilityService struct {
	manager kernel.Manager
}

var _ control.ObservabilityAPI = (*ObservabilityService)(nil)

func NewObservabilityService(manager kernel.Manager) *ObservabilityService {
	return &ObservabilityService{manager: manager}
}

func (s *ObservabilityService) RecentLogs(limit int) []kernel.LogEntry {
	return s.manager.RecentLogs(limit)
}

func (s *ObservabilityService) Traffic(context.Context) (kernel.TrafficSnapshot, error) {
	return s.manager.TrafficSnapshot(), nil
}

func (s *ObservabilityService) Connections(ctx context.Context) ([]kernel.Connection, error) {
	return s.manager.ListConnections(ctx)
}

func (s *ObservabilityService) CloseConnection(ctx context.Context, id int64) error {
	return s.manager.CloseConnection(ctx, id)
}

func (s *ObservabilityService) CloseConnections(ctx context.Context, filter kernel.ConnectionFilter) (int, error) {
	return s.manager.CloseConnections(ctx, filter)
}

func (s *ObservabilityService) Subscribe(ctx context.Context, types ...kernel.EventType) (<-chan kernel.Event, error) {
	return s.manager.Subscribe(ctx, types...)
}

func notFound(kind, id string) error {
	return kernel.NewError(kernel.ErrCodeNotFound, "control."+kind, fmt.Sprintf("%s %q was not found", kind, id)).
		WithHint("reload the current state and retry")
}
