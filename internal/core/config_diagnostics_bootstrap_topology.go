package core

import (
	"net/url"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

const (
	bootstrapDNSNodePrefix      = "dns:"
	bootstrapDomainNodePrefix   = "domain:"
	bootstrapOutboundNodePrefix = "outbound:"
)

type bootstrapDependencyGraph map[string][]string

type bootstrapTopology struct {
	cfg             map[string]any
	outbounds       []diagnosticEntry
	dnsServers      []diagnosticEntry
	knownOutbounds  map[string]struct{}
	knownDNSServers map[string]struct{}
	routeResolver   string
	dnsResolver     string
	dependencyGraph bootstrapDependencyGraph
}

func newBootstrapTopology(
	cfg map[string]any,
	outbounds []diagnosticEntry,
	dnsServers []diagnosticEntry,
) bootstrapTopology {
	uniqueOutbounds := uniqueDiagnosticEntries(outbounds)
	uniqueDNSServers := uniqueDiagnosticEntries(dnsServers)
	topology := bootstrapTopology{
		cfg:             cfg,
		outbounds:       uniqueOutbounds,
		dnsServers:      uniqueDNSServers,
		knownOutbounds:  tagSet(uniqueOutbounds),
		knownDNSServers: tagSet(uniqueDNSServers),
		dependencyGraph: bootstrapDependencyGraph{},
	}
	topology.routeResolver = resolverWithSingleFallback(
		objectValue(cfg["route"])["default_domain_resolver"],
		uniqueDNSServers,
	)
	topology.dnsResolver = resolverWithSingleFallback(
		objectValue(cfg["dns"])["final"],
		uniqueDNSServers,
	)
	topology.build()
	return topology
}

func checkDNSOutboundBootstrapCycles(
	report *model.ConfigDiagnostics,
	topology bootstrapTopology,
) {
	for _, entry := range topology.dnsServers {
		server := dnsServerObject(topology.cfg, entry.path)
		detour := strings.TrimSpace(stringValue(server["detour"]))
		if _, exists := topology.knownOutbounds[detour]; !exists {
			continue
		}
		if topology.dependencyGraph.reaches(outboundBootstrapNode(detour), dnsBootstrapNode(entry.tag)) {
			addDiagnostic(
				report,
				"dns_dependency_cycle",
				model.ConfigDiagnosticSeverityError,
				entry.path+".detour",
				detour,
				"",
			)
		}
	}
}

func (t *bootstrapTopology) build() {
	t.addOutboundDependencies()
	t.addDNSDependencies()
}

func (t *bootstrapTopology) addOutboundDependencies() {
	for _, entry := range t.outbounds {
		object := objectAtPath(t.cfg, entry.path)
		source := outboundBootstrapNode(entry.tag)
		t.addOutboundReference(source, object["detour"])
		t.addGroupDependencies(entry, object)
		t.addConfiguredRemoteDependency(entry, object)
		t.addDomainDialDependencies(entry, object)
	}
}

func (t *bootstrapTopology) addGroupDependencies(
	entry diagnosticEntry,
	object map[string]any,
) {
	if entry.typeName != "selector" && entry.typeName != "urltest" {
		return
	}
	bootstrapSource := outboundBootstrapNode(entry.tag)
	domainSource := outboundDomainNode(entry.tag)
	for _, member := range stringValues(object["outbounds"]) {
		t.addOutboundReference(bootstrapSource, member)
		t.addDomainReference(domainSource, member)
	}
}

func (t *bootstrapTopology) addConfiguredRemoteDependency(
	entry diagnosticEntry,
	object map[string]any,
) {
	if !outboundHasDomainRemote(entry, object) {
		return
	}
	source := outboundBootstrapNode(entry.tag)
	if resolver := resolverTag(object["domain_resolver"]); resolver != "" {
		t.addDNSReference(source, resolver)
		return
	}
	detour := strings.TrimSpace(stringValue(object["detour"]))
	if detour != "" && !resolvesServerOnDetour(entry.typeName) {
		t.addDomainReference(source, detour)
		return
	}
	t.addDNSReference(source, t.routeResolver)
}

func (t *bootstrapTopology) addDomainDialDependencies(
	entry diagnosticEntry,
	object map[string]any,
) {
	source := outboundDomainNode(entry.tag)
	t.dependencyGraph.add(source, outboundBootstrapNode(entry.tag))
	switch strings.ToLower(entry.typeName) {
	case "direct":
		resolver := resolverTag(object["domain_resolver"])
		if resolver == "" {
			resolver = t.routeResolver
		}
		t.addDNSReference(source, resolver)
	case "tailscale", "wireguard":
		t.addDNSReference(source, t.dnsResolver)
	}
}

func (t *bootstrapTopology) addDNSDependencies() {
	for _, entry := range t.dnsServers {
		server := dnsServerObject(t.cfg, entry.path)
		source := dnsBootstrapNode(entry.tag)
		for _, value := range []any{server["domain_resolver"], server["address_resolver"]} {
			t.addDNSReference(source, resolverTag(value))
		}
		t.addOutboundReference(source, server["detour"])
	}
}

func (t *bootstrapTopology) addDNSReference(source, resolver string) {
	if _, exists := t.knownDNSServers[resolver]; exists {
		t.dependencyGraph.add(source, dnsBootstrapNode(resolver))
	}
}

func (t *bootstrapTopology) addOutboundReference(source string, value any) {
	target := strings.TrimSpace(stringValue(value))
	if _, exists := t.knownOutbounds[target]; exists {
		t.dependencyGraph.add(source, outboundBootstrapNode(target))
	}
}

func (t *bootstrapTopology) addDomainReference(source string, value any) {
	target := strings.TrimSpace(stringValue(value))
	if _, exists := t.knownOutbounds[target]; exists {
		t.dependencyGraph.add(source, outboundDomainNode(target))
	}
}

func outboundHasDomainRemote(entry diagnosticEntry, object map[string]any) bool {
	switch strings.ToLower(entry.typeName) {
	case "wireguard":
		return wireGuardPeerHasDomain(object["peers"])
	case "tailscale":
		return tailscaleControlUsesDomain(object["control_url"])
	default:
		return isDomainName(strings.TrimSpace(stringValue(object["server"])))
	}
}

func wireGuardPeerHasDomain(value any) bool {
	peers, _ := value.([]any)
	for _, peer := range peers {
		if isDomainName(strings.TrimSpace(stringValue(objectValue(peer)["address"]))) {
			return true
		}
	}
	return false
}

func tailscaleControlUsesDomain(value any) bool {
	controlURL := strings.TrimSpace(stringValue(value))
	if controlURL == "" {
		return true
	}
	parsed, err := url.Parse(controlURL)
	return err == nil && isDomainName(parsed.Hostname())
}

func resolvesServerOnDetour(typeName string) bool {
	switch strings.ToLower(typeName) {
	case "naive", "tailscale", "wireguard":
		return true
	default:
		return false
	}
}

func resolverWithSingleFallback(value any, dnsServers []diagnosticEntry) string {
	if resolver := resolverTag(value); resolver != "" {
		return resolver
	}
	if len(dnsServers) == 1 {
		return dnsServers[0].tag
	}
	return ""
}

func resolverTag(value any) string {
	switch resolver := value.(type) {
	case string:
		return strings.TrimSpace(resolver)
	case map[string]any:
		return strings.TrimSpace(stringValue(resolver["server"]))
	default:
		return ""
	}
}

func dnsBootstrapNode(tag string) string {
	return bootstrapDNSNodePrefix + tag
}

func outboundDomainNode(tag string) string {
	return bootstrapDomainNodePrefix + tag
}

func outboundBootstrapNode(tag string) string {
	return bootstrapOutboundNodePrefix + tag
}

func (g bootstrapDependencyGraph) add(source, target string) {
	if source == "" || target == "" {
		return
	}
	g[source] = append(g[source], target)
}

func (g bootstrapDependencyGraph) reaches(start, target string) bool {
	stack := []string{start}
	visited := make(map[string]struct{}, len(g))
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current == target {
			return true
		}
		if _, seen := visited[current]; seen {
			continue
		}
		visited[current] = struct{}{}
		stack = append(stack, g[current]...)
	}
	return false
}
