package core

import (
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type outboundDependencyEdge struct {
	target string
	path   string
}

type outboundDependencyGraph struct {
	order []string
	edges map[string][]outboundDependencyEdge
}

type outboundTopologyInspector struct {
	report *model.ConfigDiagnostics
	cfg    map[string]any
	known  map[string]struct{}
	graph  outboundDependencyGraph
}

type outboundVisitState uint8

const (
	outboundUnvisited outboundVisitState = iota
	outboundVisiting
	outboundVisited
)

func checkOutboundTopology(
	report *model.ConfigDiagnostics,
	cfg map[string]any,
	entries []diagnosticEntry,
) {
	inspector := outboundTopologyInspector{
		report: report,
		cfg:    cfg,
		known:  tagSet(entries),
		graph: outboundDependencyGraph{
			edges: make(map[string][]outboundDependencyEdge),
		},
	}
	inspector.inspect(uniqueDiagnosticEntries(entries))
	inspector.graph.reportCycles(report)
}

func uniqueDiagnosticEntries(entries []diagnosticEntry) []diagnosticEntry {
	result := make([]diagnosticEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if _, exists := seen[entry.tag]; exists {
			continue
		}
		seen[entry.tag] = struct{}{}
		result = append(result, entry)
	}
	return result
}

func (i *outboundTopologyInspector) inspect(entries []diagnosticEntry) {
	for _, entry := range entries {
		i.graph.order = append(i.graph.order, entry.tag)
		object := objectAtPath(i.cfg, entry.path)
		i.addReference(entry.tag, entry.path+".detour", object["detour"])
		i.inspectGroup(entry, object)
	}
}

func (i *outboundTopologyInspector) addReference(source, path string, value any) {
	target := strings.TrimSpace(stringValue(value))
	if target == "" {
		return
	}
	if _, exists := i.known[target]; exists {
		i.graph.edges[source] = append(i.graph.edges[source], outboundDependencyEdge{target: target, path: path})
	}
}

func (i *outboundTopologyInspector) inspectGroup(entry diagnosticEntry, object map[string]any) {
	typeName := strings.TrimSpace(stringValue(object["type"]))
	if typeName != "selector" && typeName != "urltest" {
		return
	}
	members := i.addGroupMembers(entry, object["outbounds"])
	if typeName == "selector" {
		i.checkGroupDefault(entry.path, object["default"], members)
	}
}

func (i *outboundTopologyInspector) addGroupMembers(entry diagnosticEntry, value any) map[string]struct{} {
	items, _ := value.([]any)
	members := make(map[string]struct{}, len(items))
	for index, item := range items {
		target := strings.TrimSpace(stringValue(item))
		if target == "" {
			continue
		}
		members[target] = struct{}{}
		path := entry.path + ".outbounds[" + strconv.Itoa(index) + "]"
		i.addReference(entry.tag, path, target)
	}
	return members
}

func (i *outboundTopologyInspector) checkGroupDefault(path string, value any, members map[string]struct{}) {
	target := strings.TrimSpace(stringValue(value))
	if target == "" {
		return
	}
	if _, exists := i.known[target]; !exists {
		return
	}
	if _, exists := members[target]; !exists {
		addDiagnostic(i.report, "invalid_group_default", model.ConfigDiagnosticSeverityError, path+".default", target, "")
	}
}

func (g outboundDependencyGraph) reportCycles(report *model.ConfigDiagnostics) {
	states := make(map[string]outboundVisitState, len(g.order))
	for _, tag := range g.order {
		if states[tag] == outboundUnvisited {
			g.visit(report, tag, states)
		}
	}
}

func (g outboundDependencyGraph) visit(
	report *model.ConfigDiagnostics,
	tag string,
	states map[string]outboundVisitState,
) {
	states[tag] = outboundVisiting
	for _, edge := range g.edges[tag] {
		switch states[edge.target] {
		case outboundUnvisited:
			g.visit(report, edge.target, states)
		case outboundVisiting:
			addDiagnostic(report, "outbound_dependency_cycle", model.ConfigDiagnosticSeverityError, edge.path, edge.target, "")
		case outboundVisited:
		}
	}
	states[tag] = outboundVisited
}
