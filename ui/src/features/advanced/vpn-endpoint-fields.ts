import { udpNATFields } from "@/features/config/kernel114-fields"
import type { JsonObject, PolicyFieldSpec } from "@/features/policy/policy-form-model"

const vpnTypes = ["openvpn-client", "openvpn-server", "openconnect"]
const openVPNTypes = ["openvpn-client", "openvpn-server"]

export const vpnEndpointFields: readonly PolicyFieldSpec[] = [
  { path: "system", label: "system", kind: "boolean", when: { path: "type", is: vpnTypes } },
  { path: "name", label: "interfaceName", when: { path: "type", is: vpnTypes } },
  { path: "mtu", label: "mtu", kind: "number", when: { path: "type", is: vpnTypes } },
  { path: "udp_timeout", label: "udpTimeout", when: { path: "type", is: ["openvpn-client", "openconnect"] } },
  ...udpNATFields.map((field) => ({ ...field, when: { path: "type", is: vpnTypes } })),
  { path: "mode", label: "vpnMode", kind: "select", options: ["tls", "static_key"], when: { path: "type", is: openVPNTypes } },
  { path: "server", label: "server", when: { path: "type", is: ["openvpn-client", "openconnect"] } },
  { path: "server_port", label: "serverPort", kind: "number", when: { path: "type", is: "openvpn-client" } },
  { path: "servers", label: "vpnServers", kind: "json-array", when: { path: "type", is: "openvpn-client" } },
  { path: "remote_random", label: "remoteRandom", kind: "boolean", when: { path: "type", is: "openvpn-client" } },
  { path: "network", label: "vpnNetwork", kind: "select", options: ["udp", "udp4", "udp6", "tcp", "tcp4", "tcp6"], when: { path: "type", is: "openvpn-client" } },
  { path: "listen", label: "listenAddress", when: { path: "type", is: "openvpn-server" } },
  { path: "listen_port", label: "listenPort", kind: "number", when: { path: "type", is: "openvpn-server" } },
  { path: "network", label: "vpnNetwork", kind: "select", options: ["udp", "tcp"], when: { path: "type", is: "openvpn-server" } },
  { path: "users", label: "users", kind: "json-array", when: { path: "type", is: "openvpn-server" } },
  { path: "push", label: "vpnPush", kind: "json-object", when: { path: "type", is: "openvpn-server" } },
  { path: "max_clients", label: "maxClients", kind: "number", when: { path: "type", is: "openvpn-server" } },
  { path: "duplicate_cn", label: "duplicateCN", kind: "boolean", when: { path: "type", is: "openvpn-server" } },
  { path: "address", label: "address", kind: "list", when: { path: "type", is: openVPNTypes } },
  { path: "peer_address", label: "peerAddress", when: { path: "type", is: openVPNTypes } },
  { path: "peer_address_ipv6", label: "peerAddressIPv6", when: { path: "type", is: openVPNTypes } },
  { path: "topology", label: "vpnTopology", kind: "select", options: ["net30", "p2p", "subnet"], when: { path: "type", is: openVPNTypes } },
  { path: "static_key", label: "staticKey", kind: "list", when: [{ path: "type", is: openVPNTypes }, { path: "mode", is: "static_key" }] },
  { path: "static_key_path", label: "staticKeyPath", when: [{ path: "type", is: openVPNTypes }, { path: "mode", is: "static_key" }] },
  { path: "key_direction", label: "keyDirection", kind: "select", options: ["server", "client"], when: [{ path: "type", is: openVPNTypes }, { path: "mode", is: "static_key" }] },
  { path: "cipher", label: "vpnCipher", when: { path: "type", is: openVPNTypes } },
  { path: "data_ciphers", label: "dataCiphers", kind: "list", when: { path: "type", is: openVPNTypes } },
  { path: "data_ciphers_fallback", label: "dataCiphersFallback", when: { path: "type", is: openVPNTypes } },
  { path: "auth", label: "vpnAuth", when: { path: "type", is: openVPNTypes } },
  { path: "tls", label: "vpnTLS", kind: "json-object", when: { path: "type", is: vpnTypes } },
  { path: "username", label: "username", when: { path: "type", is: ["openvpn-client", "openconnect"] } },
  { path: "password", label: "password", when: { path: "type", is: ["openvpn-client", "openconnect"] } },
  { path: "route_no_pull", label: "routeNoPull", kind: "boolean", when: { path: "type", is: "openvpn-client" } },
  { path: "routes", label: "vpnRoutes", kind: "list", when: { path: "type", is: "openvpn-client" } },
  { path: "pull_filters", label: "pullFilters", kind: "json-array", when: { path: "type", is: "openvpn-client" } },
  { path: "flavor", label: "vpnFlavor", kind: "select", options: ["anyconnect", "gp", "fortinet", "f5", "pulse", "nc"], when: { path: "type", is: "openconnect" } },
  { path: "auth_group", label: "authGroup", when: { path: "type", is: "openconnect" } },
  { path: "cookie", label: "vpnCookie", when: { path: "type", is: "openconnect" } },
  { path: "token", label: "vpnToken", kind: "json-object", when: { path: "type", is: "openconnect" } },
  { path: "form_entries", label: "formEntries", kind: "json-array", when: { path: "type", is: "openconnect" } },
  { path: "no_udp", label: "disableUDP", kind: "boolean", when: { path: "type", is: "openconnect" } },
  { path: "ipv6_disabled", label: "disableIPv6", kind: "boolean", when: { path: "type", is: "openconnect" } },
  { path: "reconnect_timeout", label: "reconnectTimeout", when: { path: "type", is: "openconnect" } },
]

function hasText(value: unknown) {
  return typeof value === "string" && Boolean(value.trim())
}

export function isVPNEndpointReady(item: JsonObject): boolean {
  if (item.type === "openconnect") return hasText(item.server)
  if (item.type === "openvpn-client") return hasText(item.server)
    || Array.isArray(item.servers) && item.servers.length > 0 && item.servers.every((entry) => entry && typeof entry === "object" && !Array.isArray(entry) && hasText(entry.server))
  const addresses = Array.isArray(item.address) ? item.address : [item.address]
  return addresses.some(hasText) && typeof item.listen_port === "number" && item.listen_port > 0 && item.listen_port <= 65535
}
