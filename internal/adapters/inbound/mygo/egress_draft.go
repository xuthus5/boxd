package mygo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

const maxChainHops = 4

// groupDraft is the editor-owned form state for a strategy group. It keeps
// stable node IDs while the presentation resolves them to user-facing names.
type groupDraft struct {
	ID       string
	Name     string
	Kind     control.GroupKind
	Members  []string
	Selected string
	URL      string
	Interval string
	Query    string
}

func (d groupDraft) controlGroup() control.Group {
	selected := strings.TrimSpace(d.Selected)
	if d.Kind != control.GroupKindSelector {
		selected = ""
	}
	interval, _ := strconv.Atoi(strings.TrimSpace(d.Interval))
	return control.Group{
		ID:              d.ID,
		Name:            strings.TrimSpace(d.Name),
		Kind:            d.Kind,
		Members:         append([]string(nil), d.Members...),
		Selected:        selected,
		URL:             strings.TrimSpace(d.URL),
		IntervalSeconds: interval,
	}
}

func (d *groupDraft) addMember(nodeID string) bool {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" || containsString(d.Members, nodeID) {
		return false
	}
	d.Members = append(d.Members, nodeID)
	d.ensureSelection()
	return true
}

func (d *groupDraft) addMembers(nodeIDs []string) int {
	added := 0
	for _, nodeID := range nodeIDs {
		if d.addMember(nodeID) {
			added++
		}
	}
	return added
}

func (d *groupDraft) removeMember(nodeID string) {
	for index, member := range d.Members {
		if member == nodeID {
			d.Members = append(d.Members[:index], d.Members[index+1:]...)
			break
		}
	}
	d.ensureSelection()
}

func (d *groupDraft) moveMember(index, delta int) {
	target := index + delta
	if index < 0 || index >= len(d.Members) || target < 0 || target >= len(d.Members) {
		return
	}
	d.Members[index], d.Members[target] = d.Members[target], d.Members[index]
}

func (d *groupDraft) reorderMembers(rows []int, to int) {
	d.Members = reorderStrings(d.Members, rows, to)
}

func (d *groupDraft) ensureSelection() {
	if d.Kind != control.GroupKindSelector {
		d.Selected = ""
		return
	}
	if !containsString(d.Members, d.Selected) {
		if len(d.Members) == 0 {
			d.Selected = ""
		} else {
			d.Selected = d.Members[0]
		}
	}
}

func (d groupDraft) issues(snapshot snapshot) []string {
	issues := make([]string, 0, 3)
	if strings.TrimSpace(d.Name) == "" {
		issues = append(issues, "请填写策略组名称")
	}
	if len(d.Members) == 0 {
		issues = append(issues, "请至少添加一个成员节点")
	}
	if d.Kind == control.GroupKindSelector && d.Selected != "" && !containsString(d.Members, d.Selected) {
		issues = append(issues, "当前出口必须是已选成员")
	}
	if d.Kind == control.GroupKindURLTest {
		if interval, err := strconv.Atoi(strings.TrimSpace(d.Interval)); err != nil || interval <= 0 {
			issues = append(issues, "测速间隔必须是大于 0 的整数")
		}
	}
	for _, member := range d.Members {
		if _, exists := snapshot.NodesByID[member]; !exists {
			issues = append(issues, fmt.Sprintf("成员节点 %s 已不存在，请移除后重试", member))
		}
	}
	return issues
}

type chainDraft struct {
	ID    string
	Name  string
	Hops  []control.ProxyHop
	Query string
}

func (d chainDraft) chain() control.ProxyChain {
	return control.ProxyChain{
		ID:   d.ID,
		Name: strings.TrimSpace(d.Name),
		Hops: append([]control.ProxyHop(nil), d.Hops...),
	}
}

func (d *chainDraft) addHop(kind control.EgressKind, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(d.Hops) >= maxChainHops {
		return false
	}
	for _, hop := range d.Hops {
		if hop.Kind == kind && hop.ID == id {
			return false
		}
	}
	d.Hops = append(d.Hops, control.ProxyHop{Kind: kind, ID: id})
	return true
}

func (d *chainDraft) removeHop(index int) {
	if index < 0 || index >= len(d.Hops) {
		return
	}
	d.Hops = append(d.Hops[:index], d.Hops[index+1:]...)
}

func (d *chainDraft) moveHop(index, delta int) {
	target := index + delta
	if index < 0 || index >= len(d.Hops) || target < 0 || target >= len(d.Hops) {
		return
	}
	d.Hops[index], d.Hops[target] = d.Hops[target], d.Hops[index]
}

func (d *chainDraft) reorderHops(rows []int, to int) {
	if len(rows) == 0 {
		return
	}
	moving := make([]control.ProxyHop, 0, len(rows))
	seen := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		if row < 0 || row >= len(d.Hops) {
			continue
		}
		if _, exists := seen[row]; exists {
			continue
		}
		seen[row] = struct{}{}
		moving = append(moving, d.Hops[row])
	}
	rest := make([]control.ProxyHop, 0, len(d.Hops)-len(moving))
	for index, hop := range d.Hops {
		if _, movingRow := seen[index]; !movingRow {
			rest = append(rest, hop)
		}
	}
	insert := to
	for row := range seen {
		if row < to {
			insert--
		}
	}
	insert = clampInt(insert, 0, len(rest))
	result := make([]control.ProxyHop, 0, len(d.Hops))
	result = append(result, rest[:insert]...)
	result = append(result, moving...)
	result = append(result, rest[insert:]...)
	d.Hops = result
}

func (d chainDraft) issues(snapshot snapshot) []string {
	issues := make([]string, 0, 3)
	if strings.TrimSpace(d.Name) == "" {
		issues = append(issues, "请填写链路名称")
	}
	if len(d.Hops) == 0 {
		issues = append(issues, "请至少添加一个跳点")
	}
	if len(d.Hops) > maxChainHops {
		issues = append(issues, fmt.Sprintf("链路最多包含 %d 个跳点", maxChainHops))
	}
	seen := make(map[string]struct{}, len(d.Hops))
	for index, hop := range d.Hops {
		key := hopKey(hop)
		if _, exists := seen[key]; exists {
			issues = append(issues, fmt.Sprintf("第 %d 个跳点与前面的跳点重复", index+1))
		}
		seen[key] = struct{}{}
		if !hopExists(snapshot, hop) {
			issues = append(issues, fmt.Sprintf("第 %d 个跳点已不存在", index+1))
		}
	}
	return issues
}

func hopExists(snapshot snapshot, hop control.ProxyHop) bool {
	switch hop.Kind {
	case control.EgressKindNode:
		_, exists := snapshot.NodesByID[hop.ID]
		return exists
	case control.EgressKindGroup:
		_, exists := snapshot.GroupsByID[hop.ID]
		return exists
	default:
		return false
	}
}

func hopKey(hop control.ProxyHop) string {
	return string(hop.Kind) + "\x00" + hop.ID
}

func (e editorState) issues(snapshot snapshot) []string {
	switch e.kind {
	case editorGroup:
		return e.group.issues(snapshot)
	case editorChain:
		return e.chain.issues(snapshot)
	default:
		return nil
	}
}

func validationDiagnosticsError(diagnostics []kernel.Diagnostic) error {
	parts := make([]string, 0, min(3, len(diagnostics)))
	for index, diagnostic := range diagnostics {
		if index == 3 {
			break
		}
		message := diagnostic.Message
		if diagnostic.Hint != "" {
			message += "；" + diagnostic.Hint
		}
		parts = append(parts, message)
	}
	if len(diagnostics) > len(parts) {
		parts = append(parts, fmt.Sprintf("另有 %d 个问题", len(diagnostics)-len(parts)))
	}
	return fmt.Errorf("校验未通过：%s", strings.Join(parts, "；"))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func reorderStrings(values []string, rows []int, to int) []string {
	if len(rows) == 0 {
		return append([]string(nil), values...)
	}
	moving := make([]string, 0, len(rows))
	seen := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		if row < 0 || row >= len(values) {
			continue
		}
		if _, exists := seen[row]; exists {
			continue
		}
		seen[row] = struct{}{}
		moving = append(moving, values[row])
	}
	rest := make([]string, 0, len(values)-len(moving))
	for index, value := range values {
		if _, movingRow := seen[index]; !movingRow {
			rest = append(rest, value)
		}
	}
	insert := to
	for row := range seen {
		if row < to {
			insert--
		}
	}
	insert = clampInt(insert, 0, len(rest))
	result := make([]string, 0, len(values))
	result = append(result, rest[:insert]...)
	result = append(result, moving...)
	result = append(result, rest[insert:]...)
	return result
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
