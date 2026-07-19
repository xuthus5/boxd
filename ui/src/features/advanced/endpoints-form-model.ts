import {
  isJsonObject,
  pruneInvisiblePolicyFields,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

export const endpointTypes = ["wireguard", "tailscale"] as const
export type EndpointType = (typeof endpointTypes)[number]

const isWireGuard = { path: "type", is: "wireguard" as const }
const isTailscale = { path: "type", is: "tailscale" as const }
const keepAliveOn = { path: "disable_tcp_keep_alive", falsy: true as const }
const resolverOn = { path: "domain_resolver.server" }
const exitNodeOn = { path: "exit_node" }
const systemIfOn = { path: "system_interface", is: true as const }

export const endpointIdentityFields = [
  { path: "type", label: "type", kind: "select", options: endpointTypes, required: true, section: "basic" },
  { path: "tag", label: "tag", required: true, section: "basic" },
] as const satisfies readonly PolicyFieldSpec[]

export const wireGuardFields = [
  { path: "system", label: "system", kind: "boolean", section: "wireguard", when: isWireGuard },
  { path: "name", label: "interfaceName", section: "wireguard", when: isWireGuard },
  { path: "mtu", label: "mtu", kind: "number", section: "wireguard", when: isWireGuard },
  { path: "address", label: "address", kind: "list", required: true, section: "wireguard", when: isWireGuard },
  { path: "private_key", label: "privateKey", required: true, section: "wireguard", when: isWireGuard },
  { path: "listen_port", label: "listenPort", kind: "number", section: "wireguard", when: isWireGuard },
  { path: "peers", label: "peers", kind: "json-array", section: "wireguard", when: isWireGuard },
  { path: "udp_timeout", label: "udpTimeout", section: "wireguard", when: isWireGuard },
  { path: "workers", label: "workers", kind: "number", section: "wireguard", when: isWireGuard },
] as const satisfies readonly PolicyFieldSpec[]

export const tailscaleFields = [
  { path: "state_directory", label: "stateDirectory", section: "tailscale", when: isTailscale },
  { path: "auth_key", label: "authKey", section: "tailscale", when: isTailscale },
  { path: "control_url", label: "controlURL", section: "tailscale", when: isTailscale },
  { path: "ephemeral", label: "ephemeral", kind: "boolean", section: "tailscale", when: isTailscale },
  { path: "hostname", label: "hostname", section: "tailscale", when: isTailscale },
  { path: "accept_routes", label: "acceptRoutes", kind: "boolean", section: "tailscale", when: isTailscale },
  { path: "exit_node", label: "exitNode", section: "tailscale", when: isTailscale },
  {
    path: "exit_node_allow_lan_access", label: "exitNodeAllowLANAccess", kind: "boolean", section: "tailscale",
    when: [isTailscale, exitNodeOn],
  },
  { path: "advertise_routes", label: "advertiseRoutes", kind: "list", section: "tailscale", when: isTailscale },
  { path: "advertise_exit_node", label: "advertiseExitNode", kind: "boolean", section: "tailscale", when: isTailscale },
  { path: "advertise_tags", label: "advertiseTags", kind: "list", section: "tailscale", when: isTailscale },
  { path: "relay_server_port", label: "relayServerPort", kind: "number", section: "tailscale", when: isTailscale },
  {
    path: "relay_server_static_endpoints", label: "relayServerStaticEndpoints", kind: "list", section: "tailscale",
    when: isTailscale,
  },
  { path: "system_interface", label: "systemInterface", kind: "boolean", section: "tailscale", when: isTailscale },
  {
    path: "system_interface_name", label: "systemInterfaceName", section: "tailscale",
    when: [isTailscale, systemIfOn],
  },
  {
    path: "system_interface_mtu", label: "systemInterfaceMTU", kind: "number", section: "tailscale",
    when: [isTailscale, systemIfOn],
  },
  { path: "udp_timeout", label: "udpTimeout", section: "tailscale", when: isTailscale },
] as const satisfies readonly PolicyFieldSpec[]

export const endpointDialerFields = [
  { path: "detour", label: "detour", kind: "ref", ref: "outbound", section: "dialer" },
  { path: "bind_interface", label: "bindInterface", kind: "network-interface", section: "dialer" },
  { path: "inet4_bind_address", label: "inet4BindAddress", section: "dialer" },
  { path: "inet6_bind_address", label: "inet6BindAddress", section: "dialer" },
  { path: "bind_address_no_port", label: "bindAddressNoPort", kind: "boolean", section: "dialer" },
  { path: "protect_path", label: "protectPath", section: "dialer" },
  { path: "routing_mark", label: "routingMark", section: "dialer" },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean", section: "dialer" },
  { path: "netns", label: "networkNamespace", section: "dialer" },
  { path: "connect_timeout", label: "connectTimeout", section: "dialer" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean", section: "dialer" },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean", section: "dialer" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean", section: "dialer" },
  { path: "tcp_keep_alive", label: "tcpKeepAlive", section: "dialer", when: keepAliveOn },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", section: "dialer", when: keepAliveOn },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean", section: "dialer" },
  { path: "domain_resolver.server", label: "domainResolverServer", kind: "ref", ref: "dns-server", section: "dialer" },
  {
    path: "domain_resolver.strategy", label: "domainResolverStrategy", kind: "select",
    options: ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"], section: "dialer", when: resolverOn,
  },
  {
    path: "domain_resolver.disable_cache", label: "domainResolverDisableCache", kind: "boolean",
    section: "dialer", when: resolverOn,
  },
  {
    path: "domain_resolver.rewrite_ttl", label: "domainResolverRewriteTTL", kind: "number",
    section: "dialer", when: resolverOn,
  },
  {
    path: "domain_resolver.client_subnet", label: "domainResolverClientSubnet",
    section: "dialer", when: resolverOn,
  },
  {
    path: "network_strategy", label: "networkStrategy", kind: "select",
    options: ["default", "fallback", "hybrid", "prefer_ipv4", "prefer_ipv6"], section: "dialer",
  },
  { path: "network_type", label: "networkType", kind: "list", section: "dialer" },
  { path: "fallback_network_type", label: "fallbackNetworkType", kind: "list", section: "dialer" },
  { path: "fallback_delay", label: "fallbackDelay", section: "dialer" },
] as const satisfies readonly PolicyFieldSpec[]

export const endpointFields = [
  ...endpointIdentityFields,
  ...wireGuardFields,
  ...tailscaleFields,
  ...endpointDialerFields,
] as const satisfies readonly PolicyFieldSpec[]

export function isEndpointObject(value: JsonValue | undefined): value is JsonObject {
  return isJsonObject(value)
}

export function isEndpointsStructureValid(value: JsonValue | null | undefined): value is JsonObject[] {
  return Array.isArray(value) && value.every(isJsonObject)
}

export function normalizeEndpoints(value: JsonValue | undefined): JsonObject[] {
  return isEndpointsStructureValid(value) ? value : []
}

export function inferEndpointType(item: JsonObject): EndpointType {
  return item.type === "tailscale" ? "tailscale" : "wireguard"
}

export function createEndpointDraft(type: EndpointType = "wireguard"): JsonObject {
  return type === "tailscale"
    ? { type: "tailscale", tag: "" }
    : { type: "wireguard", tag: "", address: [], private_key: "", peers: [] }
}

export function changeEndpointType(item: JsonObject, type: string): JsonObject {
  const nextType: EndpointType = type === "tailscale" ? "tailscale" : "wireguard"
  const base = createEndpointDraft(nextType)
  if (typeof item.tag === "string" && item.tag) base.tag = item.tag
  return prepareEndpointObject(base)
}

export function prepareEndpointObject(item: JsonObject): JsonObject {
  return pruneInvisiblePolicyFields(item, endpointFields)
}

export function prepareEndpoints(items: readonly JsonObject[]): JsonObject[] {
  return items.map((item) => prepareEndpointObject(item))
}

export function isEndpointReady(item: JsonObject): boolean {
  if (typeof item.tag !== "string" || !item.tag.trim()) return false
  const type = inferEndpointType(item)
  if (type === "wireguard") {
    const address = item.address
    const hasAddress = Array.isArray(address)
      ? address.some((entry) => typeof entry === "string" && entry.trim())
      : typeof address === "string" && Boolean(address.trim())
    return hasAddress && typeof item.private_key === "string" && Boolean(item.private_key.trim())
  }
  return true
}

export function summarizeEndpoint(item: JsonObject) {
  const type = inferEndpointType(item)
  if (type === "wireguard") {
    const peers = Array.isArray(item.peers) ? item.peers.length : 0
    const address = Array.isArray(item.address)
      ? item.address.filter((entry): entry is string => typeof entry === "string").join(", ")
      : typeof item.address === "string" ? item.address : ""
    return { type, detail: address || undefined, meta: peers }
  }
  const hostname = typeof item.hostname === "string" ? item.hostname : ""
  const exitNode = typeof item.exit_node === "string" ? item.exit_node : ""
  return { type, detail: hostname || exitNode || undefined, meta: 0 }
}
