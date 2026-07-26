package core

import (
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type dnsDependencyEdge struct {
	target string
	path   string
}

type dnsDependencyGraph struct {
	order []string
	edges map[string][]dnsDependencyEdge
}

type dnsTopologyInspector struct {
	cfg   map[string]any
	known map[string]struct{}
	graph dnsDependencyGraph
}

type dnsFakeIPServer struct {
	entry diagnosticEntry
	path  string
}

type dnsVisitState uint8

const (
	dnsUnvisited dnsVisitState = iota
	dnsVisiting
	dnsVisited
)

func checkDNSTopology(
	report *model.ConfigDiagnostics,
	cfg map[string]any,
	entries []diagnosticEntry,
) {
	uniqueEntries := uniqueDiagnosticEntries(entries)
	inspector := dnsTopologyInspector{
		cfg:   cfg,
		known: tagSet(uniqueEntries),
		graph: dnsDependencyGraph{edges: make(map[string][]dnsDependencyEdge)},
	}
	inspector.inspect(uniqueEntries)
	inspector.graph.reportCycles(report)
	checkDNSDefault(report, cfg, entries, uniqueEntries)
	checkMultipleFakeIPDNSServers(report, cfg, entries)
}

func (i *dnsTopologyInspector) inspect(entries []diagnosticEntry) {
	for _, entry := range entries {
		i.graph.order = append(i.graph.order, entry.tag)
		server := dnsServerObject(i.cfg, entry.path)
		i.addDomainResolver(entry.tag, entry.path+".domain_resolver", server["domain_resolver"])
		i.addReference(entry.tag, entry.path+".address_resolver", server["address_resolver"])
	}
}

func (i *dnsTopologyInspector) addDomainResolver(source, path string, value any) {
	switch resolver := value.(type) {
	case string:
		i.addReference(source, path, resolver)
	case map[string]any:
		i.addReference(source, path+".server", resolver["server"])
	}
}

func (i *dnsTopologyInspector) addReference(source, path string, value any) {
	target := strings.TrimSpace(stringValue(value))
	if target == "" {
		return
	}
	if _, exists := i.known[target]; exists {
		i.graph.edges[source] = append(i.graph.edges[source], dnsDependencyEdge{target: target, path: path})
	}
}

func (g dnsDependencyGraph) reportCycles(report *model.ConfigDiagnostics) {
	states := make(map[string]dnsVisitState, len(g.order))
	for _, tag := range g.order {
		if states[tag] == dnsUnvisited {
			g.visit(report, tag, states)
		}
	}
}

func (g dnsDependencyGraph) visit(
	report *model.ConfigDiagnostics,
	tag string,
	states map[string]dnsVisitState,
) {
	states[tag] = dnsVisiting
	for _, edge := range g.edges[tag] {
		switch states[edge.target] {
		case dnsUnvisited:
			g.visit(report, edge.target, states)
		case dnsVisiting:
			addDiagnostic(report, "dns_dependency_cycle", model.ConfigDiagnosticSeverityError, edge.path, edge.target, "")
		case dnsVisited:
		}
	}
	states[tag] = dnsVisited
}

func checkDNSDefault(
	report *model.ConfigDiagnostics,
	cfg map[string]any,
	entries []diagnosticEntry,
	uniqueEntries []diagnosticEntry,
) {
	dns := objectValue(cfg["dns"])
	final := strings.TrimSpace(stringValue(dns["final"]))
	if final != "" {
		entry, found := diagnosticEntryByTag(uniqueEntries, final)
		if found {
			addInvalidDNSDefault(report, entry, dnsServerObject(cfg, entry.path), "dns.final")
		}
		return
	}
	if len(entries) == 0 || entries[0].path != "dns.servers[0]" {
		return
	}
	entry := entries[0]
	addInvalidDNSDefault(report, entry, dnsServerObject(cfg, entry.path), "")
}

func addInvalidDNSDefault(
	report *model.ConfigDiagnostics,
	entry diagnosticEntry,
	server map[string]any,
	finalPath string,
) {
	fakeIPPath, fakeIP := dnsFakeIPPath(entry.path, server)
	if !fakeIP {
		return
	}
	path := fakeIPPath
	if finalPath != "" {
		path = finalPath
	}
	addDiagnostic(report, "invalid_dns_default", model.ConfigDiagnosticSeverityError, path, entry.tag, "")
}

func checkMultipleFakeIPDNSServers(
	report *model.ConfigDiagnostics,
	cfg map[string]any,
	entries []diagnosticEntry,
) {
	fakeIPServers := make([]dnsFakeIPServer, 0, len(entries))
	for _, entry := range entries {
		path, fakeIP := dnsFakeIPPath(entry.path, dnsServerObject(cfg, entry.path))
		if fakeIP {
			fakeIPServers = append(fakeIPServers, dnsFakeIPServer{entry: entry, path: path})
		}
	}
	if len(fakeIPServers) < 2 {
		return
	}
	for _, server := range fakeIPServers[1:] {
		addDiagnostic(
			report,
			"multiple_fakeip_dns_servers",
			model.ConfigDiagnosticSeverityError,
			server.path,
			server.entry.tag,
			"",
		)
	}
}

func dnsFakeIPPath(path string, server map[string]any) (string, bool) {
	typeName := strings.TrimSpace(stringValue(server["type"]))
	if typeName == "fakeip" {
		return path + ".type", true
	}
	isLegacy := typeName == "" || typeName == "legacy"
	if isLegacy && strings.TrimSpace(stringValue(server["address"])) == "fakeip" {
		return path + ".address", true
	}
	return "", false
}

func diagnosticEntryByTag(entries []diagnosticEntry, tag string) (diagnosticEntry, bool) {
	for _, entry := range entries {
		if entry.tag == tag {
			return entry, true
		}
	}
	return diagnosticEntry{}, false
}

func dnsServerObject(cfg map[string]any, path string) map[string]any {
	const prefix = "dns.servers["
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, "]") {
		return nil
	}
	indexText := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "]")
	index, err := strconv.Atoi(indexText)
	if err != nil {
		return nil
	}
	dns := objectValue(cfg["dns"])
	servers, _ := dns["servers"].([]any)
	if index < 0 || index >= len(servers) {
		return nil
	}
	return objectValue(servers[index])
}
