package core

import "slices"

type outboundGroupSync struct {
	outbounds []any
	byTag     map[string]map[string]any
	indices   map[string]int
}

// SyncProxySelector 同步代理候选并保留有效选择；空组仅使用阻止出站。
// HTTP 与桌面端复用此逻辑，避免删除最后节点后产生悬空引用或直连回退。
func SyncProxySelector(outbounds []any, members []string) []any {
	sync := newOutboundGroupSync(outbounds)
	proxy := sync.byTag["proxy"]
	if proxy != nil && proxy["type"] != "selector" {
		sync.pruneMissingMembers()
		return sync.outbounds
	}
	entry := cloneMap(proxy)
	entry["type"] = "selector"
	entry["tag"] = "proxy"
	if len(members) == 0 {
		members = []string{sync.blockingTag()}
		delete(entry, "default")
	} else {
		members = sync.preserveProxyMembers(members, outboundGroupMembers(entry["outbounds"]))
		selected, _ := entry["default"].(string)
		if !slices.Contains(members, selected) {
			entry["default"] = members[0]
		}
	}
	entry["outbounds"] = members
	sync.replace(entry)
	sync.pruneMissingMembers()
	return sync.outbounds
}

func (s *outboundGroupSync) preserveProxyMembers(members, previous []string) []string {
	merged := slices.Clone(members)
	// 单个阻止出站是无节点占位；保留它会让磁盘缓存阻止首个新节点接通。
	if len(previous) == 1 && s.byTag[previous[0]]["type"] == "block" {
		return merged
	}
	seen := make(map[string]bool, len(members)+len(previous))
	for _, member := range members {
		seen[member] = true
	}
	// 运行时选项存储在独立缓存，不能只通过配置 default 推断用户选择。
	for _, member := range previous {
		if member == "proxy" || s.byTag[member] == nil || seen[member] {
			continue
		}
		merged = append(merged, member)
		seen[member] = true
	}
	return merged
}

func newOutboundGroupSync(outbounds []any) *outboundGroupSync {
	sync := &outboundGroupSync{
		outbounds: slices.Clone(outbounds),
		byTag:     make(map[string]map[string]any, len(outbounds)),
		indices:   make(map[string]int, len(outbounds)),
	}
	for index, item := range outbounds {
		entry, _ := item.(map[string]any)
		tag, _ := entry["tag"].(string)
		if tag != "" {
			sync.byTag[tag] = entry
			sync.indices[tag] = index
		}
	}
	return sync
}

func (s *outboundGroupSync) blockingTag() string {
	tag := "block"
	for s.byTag[tag] != nil {
		if s.byTag[tag]["type"] == "block" {
			return tag
		}
		tag += "-fallback"
	}
	s.replace(map[string]any{"type": "block", "tag": tag})
	return tag
}

func (s *outboundGroupSync) replace(entry map[string]any) {
	tag, _ := entry["tag"].(string)
	s.byTag[tag] = entry
	if index, exists := s.indices[tag]; exists {
		s.outbounds[index] = entry
		return
	}
	s.indices[tag] = len(s.outbounds)
	s.outbounds = append(s.outbounds, entry)
}

func (s *outboundGroupSync) pruneMissingMembers() {
	for _, item := range s.outbounds {
		entry, _ := item.(map[string]any)
		if entry["type"] != "selector" && entry["type"] != "urltest" {
			continue
		}
		members := outboundGroupMembers(entry["outbounds"])
		valid := make([]string, 0, len(members))
		seen := make(map[string]bool, len(members))
		for _, member := range members {
			if s.byTag[member] != nil && !seen[member] {
				valid = append(valid, member)
				seen[member] = true
			}
		}
		if len(valid) == 0 {
			valid = []string{s.blockingTag()}
		}
		updated := cloneMap(entry)
		updated["outbounds"] = valid
		if selected, ok := updated["default"].(string); ok && !slices.Contains(valid, selected) {
			delete(updated, "default")
		}
		s.replace(updated)
	}
}

func outboundGroupMembers(value any) []string {
	if members, ok := value.([]string); ok {
		return members
	}
	values, _ := value.([]any)
	members := make([]string, 0, len(values))
	for _, value := range values {
		if member, ok := value.(string); ok && member != "" {
			members = append(members, member)
		}
	}
	return members
}
