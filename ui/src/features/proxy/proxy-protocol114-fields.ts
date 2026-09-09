import { quicFields } from "@/features/config/kernel114-fields"
import type { FieldSpec } from "@/features/proxy/proxy-form-model"

const snellShared: FieldSpec[] = [
  { path: "version", label: "version", kind: "number", section: "auth" },
  { path: "psk", label: "psk", section: "auth" },
  { path: "obfs_mode", label: "obfuscationType", kind: "select", options: ["none", "http", "tls"], section: "protocol", when: { path: "version", is: [4, 5] } },
  { path: "mode", label: "snellMode", kind: "select", options: ["default", "unshaped", "unsafe-raw"], section: "protocol", when: { path: "version", is: 6 } },
]

export const inbound114Protocols: Record<string, FieldSpec[]> = {
  snell: [...snellShared, { path: "users", label: "users", kind: "json-value", section: "auth" }],
  cloudflared: [
    { path: "token", label: "tunnelToken", section: "auth" },
    { path: "protocol", label: "tunnelProtocol", kind: "select", options: ["auto", "quic", "http2", "h2mux"], section: "protocol" },
    { path: "ha_connections", label: "haConnections", kind: "number", section: "protocol" },
    { path: "post_quantum", label: "postQuantum", kind: "boolean", section: "protocol" },
    { path: "edge_ip_version", label: "edgeIPVersion", kind: "number", section: "protocol" },
    { path: "datagram_version", label: "datagramVersion", kind: "select", options: ["v2", "v3"], section: "protocol" },
    { path: "grace_period", label: "gracePeriod", section: "protocol" },
    { path: "region", label: "region", section: "protocol" },
    { path: "control_dialer", label: "controlDialer", kind: "json-object", section: "protocol" },
    { path: "tunnel_dialer", label: "tunnelDialer", kind: "json-object", section: "protocol" },
  ],
}

export const outbound114Protocols: Record<string, FieldSpec[]> = {
  snell: [
    ...snellShared,
    { path: "userkey", label: "userKey", section: "auth" },
    { path: "reuse", label: "reuseConnection", kind: "boolean", section: "protocol" },
    { path: "obfs_host", label: "obfuscationHost", section: "protocol", when: { path: "version", is: 4 } },
    { path: "network", label: "network", kind: "network-multi", section: "protocol" },
  ],
  bridge: [
    { path: "interface", label: "bridgeInterface", kind: "network-interface", section: "protocol" },
    { path: "bridge_name", label: "bridgeName", section: "protocol" },
    { path: "iproute2_table_index", label: "ipRouteTableIndex", kind: "number", section: "protocol" },
    { path: "iproute2_rule_index", label: "ipRouteRuleIndex", kind: "number", section: "protocol" },
  ],
}

export const hysteria2114Fields: FieldSpec[] = [
  { path: "bbr_profile", label: "bbrProfile", kind: "select", options: ["standard", "conservative", "aggressive"], section: "protocol" },
  { path: "obfs.min_packet_size", label: "minPacketSize", kind: "number", section: "protocol", when: { path: "obfs.type", is: "gecko" } },
  { path: "obfs.max_packet_size", label: "maxPacketSize", kind: "number", section: "protocol", when: { path: "obfs.type", is: "gecko" } },
  { path: "realm", label: "hysteriaRealm", kind: "json-object", section: "protocol" },
]

export function quicProtocolFields(type: string): FieldSpec[] {
  return ["hysteria", "hysteria2", "tuic"].includes(type) ? quicFields : []
}
