package core

import (
	"net/url"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func checkDNSEndpointBootstrapCycle(report *model.ConfigDiagnostics, topology bootstrapTopology, entry diagnosticEntry) {
	server := dnsServerObject(topology.cfg, entry.path)
	tag := stringValue(server["endpoint"])
	if topology.dependencyGraph.reaches(outboundBootstrapNode(tag), dnsBootstrapNode(entry.tag)) {
		addDiagnostic(report, "dns_dependency_cycle", model.ConfigDiagnosticSeverityError, entry.path+".endpoint", tag, "")
	}
}

func openConnectServerUsesDomain(server string) bool {
	if server == "" {
		return false
	}
	if !strings.Contains(server, "://") {
		server = "https://" + server
	}
	parsed, err := url.Parse(server)
	return err == nil && isDomainName(parsed.Hostname())
}

func openVPNServerUsesDomain(object map[string]any) bool {
	if isDomainName(stringValue(object["server"])) {
		return true
	}
	for _, remote := range diagnosticObjects(object["servers"], "") {
		if isDomainName(stringValue(remote.object["server"])) {
			return true
		}
	}
	return false
}

func diagnosticDomainRemotePath(entry diagnosticObject) string {
	server := strings.TrimSpace(stringValue(entry.object["server"]))
	switch stringValue(entry.object["type"]) {
	case "openconnect":
		if openConnectServerUsesDomain(server) {
			return entry.path + ".server"
		}
		return ""
	case "openvpn-client":
		for _, remote := range diagnosticObjects(entry.object["servers"], entry.path+".servers") {
			if isDomainName(stringValue(remote.object["server"])) {
				return remote.path + ".server"
			}
		}
	}
	if isDomainName(server) {
		return entry.path + ".server"
	}
	return ""
}
