package core

import (
	"context"
	"fmt"
	"net"

	M "github.com/sagernet/sing/common/metadata"
)

// OutboundGroups 返回当前所有出站分组（selector/urltest）及其当前选中与成员列表。
func (s *SBInstance) OutboundGroups() []OutboundGroupInfo {
	s.mu.Lock()
	box := s.box
	s.mu.Unlock()

	if box == nil {
		return nil
	}

	var groups []OutboundGroupInfo
	for _, ob := range box.Outbound().Outbounds() {
		g, ok := ob.(selectableOutbound)
		if !ok {
			continue
		}
		groups = append(groups, OutboundGroupInfo{
			Type: g.Type(),
			Tag:  g.Tag(),
			Now:  g.Now(),
			All:  g.All(),
		})
	}
	return groups
}

// SelectOutbound 切换某 selector 分组的当前出站。selector 内部会在启用 cache_file 时持久化选择。
func (s *SBInstance) SelectOutbound(groupTag, outTag string) error {
	s.mu.Lock()
	box := s.box
	running := s.running
	s.mu.Unlock()

	if !running || box == nil {
		return ErrNotRunning
	}

	ob, found := box.Outbound().Outbound(groupTag)
	if !found {
		return fmt.Errorf("%w: %q", ErrGroupNotFound, groupTag)
	}

	sel, ok := ob.(selectableOutbound)
	if !ok {
		return fmt.Errorf("%w: %q", ErrNotSelectable, groupTag)
	}

	matched := false
	all := sel.All()
	for _, t := range all {
		if t == outTag {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("%w: %q not in group %q", ErrTagNotInGroup, outTag, groupTag)
	}

	if !sel.SelectOutbound(outTag) {
		return fmt.Errorf("failed to select outbound %q in group %q", outTag, groupTag)
	}
	return nil
}

// URLTestDelays 对某 urltest 分组触发测速，返回各成员延迟（ms）。
// 仅当目标出站实现 adapter.URLTestGroup 接口时可用。
func (s *SBInstance) URLTestDelays(ctx context.Context, groupTag string) (map[string]uint16, error) {
	s.mu.Lock()
	box := s.box
	running := s.running
	s.mu.Unlock()

	if !running || box == nil {
		return nil, ErrNotRunning
	}

	ob, found := box.Outbound().Outbound(groupTag)
	if !found {
		return nil, fmt.Errorf("%w: %q", ErrGroupNotFound, groupTag)
	}

	type urlTester interface {
		URLTest(ctx context.Context) (map[string]uint16, error)
	}

	tester, ok := ob.(urlTester)
	if !ok {
		return nil, fmt.Errorf("%w: %q is not a urltest group", ErrNotSelectable, groupTag)
	}

	return tester.URLTest(ctx)
}

// CloseConnection 关闭指定 id 的活跃连接，返回是否找到并关闭。
func (s *SBInstance) CloseConnection(id int64) bool {
	s.mu.Lock()
	traffic := s.Traffic
	s.mu.Unlock()

	if traffic == nil {
		return false
	}
	return traffic.CloseConn(id)
}

// CloseAllConnections 关闭全部活跃连接，返回已关闭数量。
func (s *SBInstance) CloseAllConnections() int {
	s.mu.Lock()
	traffic := s.Traffic
	s.mu.Unlock()

	if traffic == nil {
		return 0
	}
	return traffic.CloseAllConns()
}

func (s *SBInstance) DialOutbound(ctx context.Context, tag, network, addr string) (net.Conn, error) {
	s.mu.Lock()
	if !s.running || s.box == nil {
		s.mu.Unlock()
		return nil, ErrNotRunning
	}
	s.mu.Unlock()

	outbound, found := s.box.Outbound().Outbound(tag)
	if !found {
		return nil, fmt.Errorf("outbound %q not found in running instance", tag)
	}
	return outbound.DialContext(ctx, network, M.ParseSocksaddr(addr))
}
