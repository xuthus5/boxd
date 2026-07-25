import {
  isJsonObject,
  pruneInvisiblePolicyFields,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"
import { createPolicyNumberTransform } from "@/features/policy/policy-number-transform"
import type { JsonValue } from "@/lib/api/types"

const enabledOn = { path: "enabled", is: true as const }
const resolverOn = { path: "domain_resolver.server" }
const keepAliveOn = { path: "disable_tcp_keep_alive", falsy: true as const }
const activeResolverOn = [enabledOn, resolverOn] as const
const activeKeepAliveOn = [enabledOn, keepAliveOn] as const

export const ntpBasicFields = [
  { path: "enabled", label: "enabled", kind: "boolean", section: "basic" },
  { path: "server", label: "server", section: "basic", when: enabledOn },
  { path: "server_port", label: "serverPort", kind: "number", section: "basic", when: enabledOn },
  { path: "interval", label: "interval", section: "basic", when: enabledOn },
  { path: "write_to_system", label: "writeToSystem", kind: "boolean", section: "basic", when: enabledOn },
] as const satisfies readonly PolicyFieldSpec[]

export const ntpDialerFields = [
  { path: "detour", label: "detour", kind: "ref", ref: "outbound", section: "dialer", when: enabledOn },
  { path: "bind_interface", label: "bindInterface", kind: "network-interface", section: "dialer", when: enabledOn },
  { path: "inet4_bind_address", label: "inet4BindAddress", section: "dialer", when: enabledOn },
  { path: "inet6_bind_address", label: "inet6BindAddress", section: "dialer", when: enabledOn },
  { path: "bind_address_no_port", label: "bindAddressNoPort", kind: "boolean", section: "dialer", when: enabledOn },
  { path: "protect_path", label: "protectPath", section: "dialer", when: enabledOn },
  { path: "routing_mark", label: "routingMark", kind: "text", section: "dialer", when: enabledOn },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean", section: "dialer", when: enabledOn },
  { path: "netns", label: "networkNamespace", section: "dialer", when: enabledOn },
  { path: "connect_timeout", label: "connectTimeout", section: "dialer", when: enabledOn },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean", section: "dialer", when: enabledOn },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean", section: "dialer", when: enabledOn },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean", section: "dialer", when: enabledOn },
  { path: "tcp_keep_alive", label: "tcpKeepAlive", section: "dialer", when: activeKeepAliveOn },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", section: "dialer", when: activeKeepAliveOn },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean", section: "dialer", when: enabledOn },
  { path: "domain_resolver.server", label: "domainResolverServer", kind: "ref", ref: "dns-server", section: "resolver", when: enabledOn },
  {
    path: "domain_resolver.strategy", label: "domainResolverStrategy", kind: "select",
    options: ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"], section: "resolver", when: activeResolverOn,
  },
  { path: "domain_resolver.disable_cache", label: "domainResolverDisableCache", kind: "boolean", section: "resolver", when: activeResolverOn },
  { path: "domain_resolver.rewrite_ttl", label: "domainResolverRewriteTTL", kind: "number", section: "resolver", when: activeResolverOn },
  { path: "domain_resolver.client_subnet", label: "domainResolverClientSubnet", section: "resolver", when: activeResolverOn },
  {
    path: "network_strategy", label: "networkStrategy", kind: "select",
    options: ["default", "fallback", "hybrid"], section: "network", when: enabledOn,
  },
  { path: "network_type", label: "networkType", kind: "list", section: "network", when: enabledOn },
  { path: "fallback_network_type", label: "fallbackNetworkType", kind: "list", section: "network", when: enabledOn },
  { path: "fallback_delay", label: "fallbackDelay", section: "network", when: enabledOn },
] as const satisfies readonly PolicyFieldSpec[]

export const ntpFields = [...ntpBasicFields, ...ntpDialerFields] as const satisfies readonly PolicyFieldSpec[]

export function isNTPStructureValid(value: JsonValue | null | undefined): value is JsonObject {
  const object = value ?? undefined
  if (!isJsonObject(object)) return false
  const resolver = object.domain_resolver
  return resolver === undefined || typeof resolver === "string" || isJsonObject(resolver)
}

export function normalizeNTPObject(value: JsonValue | undefined): JsonObject {
  if (!isJsonObject(value)) return {}
  if (typeof value.domain_resolver === "string") {
    const server = value.domain_resolver.trim()
    if (server) return { ...value, domain_resolver: { server } }
    const next = { ...value }
    delete next.domain_resolver
    return next
  }
  return value
}

export function prepareNTPObject(object: JsonObject): JsonObject {
  const prepared = pruneInvisiblePolicyFields(normalizeNTPObject(object), ntpFields)
  const resolver = prepared.domain_resolver
  if (isJsonObject(resolver) && (typeof resolver.server !== "string" || !resolver.server.trim())) {
    return setPolicyPath(prepared, "domain_resolver", undefined)
  }
  return prepared
}

export const transformNTPField = createPolicyNumberTransform({
  "server_port": { kind: "integer", maximum: 65535, fieldKind: "number" },
  "routing_mark": { kind: "mark", maximum: 0xFFFFFFFF, fieldKind: "text" },
  "domain_resolver.rewrite_ttl": { kind: "integer", maximum: 0xFFFFFFFF, fieldKind: "number" },
})
