import {
  getPolicyPath,
  isJsonObject,
  setPolicyPath,
  type JsonObject,
} from "@/features/policy/policy-form-model"

type CleanupKind =
  | "string" | "boolean" | "number" | "list" | "number-list" | "json-object" | "json-array"
  | "query-type" | "rcode" | "routing-mark" | "value"
type CleanupField = readonly [path: string, kind?: CleanupKind]
export type CleanupRegistry = Record<string, readonly CleanupField[]>

const domainResolverFields: readonly CleanupField[] = [
  ["domain_resolver.timeout"], ["domain_resolver.disable_optimistic_cache", "boolean"],
  ["domain_resolver.server"], ["domain_resolver.strategy"], ["domain_resolver.disable_cache", "boolean"],
  ["domain_resolver.rewrite_ttl", "number"], ["domain_resolver.client_subnet"],
]

const dialerFields: readonly CleanupField[] = [
  ["detour"], ["bind_interface"], ["inet4_bind_address"], ["inet6_bind_address"],
  ["bind_address_no_port", "boolean"], ["protect_path"], ["routing_mark", "routing-mark"],
  ["reuse_addr", "boolean"], ["netns"], ["connect_timeout"], ["tcp_fast_open", "boolean"],
  ["tcp_multi_path", "boolean"], ["disable_tcp_keep_alive", "boolean"], ["tcp_keep_alive"],
  ["tcp_keep_alive_interval"], ["udp_fragment", "boolean"], ...domainResolverFields,
  ["network_strategy"], ["network_type", "list"], ["fallback_network_type", "list"], ["fallback_delay"],
  ["domain_strategy"],
]

const remoteFields: readonly CleanupField[] = [["server"], ["server_port", "number"], ...dialerFields]
const tlsFields: readonly CleanupField[] = [
  ["tls.engine"], ["tls.handshake_timeout"], ["tls.spoof"], ["tls.spoof_method"],
  ["tls.enabled", "boolean"], ["tls.disable_sni", "boolean"], ["tls.server_name"],
  ["tls.insecure", "boolean"], ["tls.alpn", "list"], ["tls.min_version"], ["tls.max_version"],
  ["tls.cipher_suites", "list"], ["tls.curve_preferences", "list"], ["tls.certificate", "list"],
  ["tls.certificate_path"], ["tls.certificate_public_key_sha256", "list"],
  ["tls.client_certificate", "list"], ["tls.client_certificate_path"], ["tls.client_key", "list"],
  ["tls.client_key_path"], ["tls.fragment", "boolean"], ["tls.fragment_fallback_delay"],
  ["tls.record_fragment", "boolean"], ["tls.kernel_tx", "boolean"], ["tls.kernel_rx", "boolean"],
  ["tls.ech.enabled", "boolean"], ["tls.ech.config", "list"], ["tls.ech.config_path"],
  ["tls.ech.query_server_name"], ["tls.ech.pq_signature_schemes_enabled", "boolean"],
  ["tls.ech.dynamic_record_sizing_disabled", "boolean"], ["tls.utls.enabled", "boolean"],
  ["tls.utls.fingerprint"], ["tls.reality.enabled", "boolean"], ["tls.reality.public_key"],
  ["tls.reality.short_id"],
]

const localFields: readonly CleanupField[] = [...dialerFields, ["prefer_go", "boolean"], ["neighbor_domain", "list"]]
const endpointFields: readonly CleanupField[] = [["endpoint"], ["accept_default_resolvers", "boolean"], ["accept_search_domain", "boolean"]]
const httpsFields: readonly CleanupField[] = [
  ...remoteFields, ...tlsFields, ["path"], ["method"], ["headers", "json-object"],
]

export const dnsServerCleanupFields: CleanupRegistry = {
  local: localFields,
  hosts: [["path", "list"], ["predefined", "json-object"]],
  udp: remoteFields,
  tcp: remoteFields,
  tls: [...remoteFields, ...tlsFields],
  quic: [...remoteFields, ...tlsFields],
  https: httpsFields,
  h3: httpsFields,
  dhcp: [...localFields, ["interface"]],
  mdns: [...localFields, ["interface", "list"]],
  tailscale: endpointFields, openvpn: endpointFields, openconnect: endpointFields,
  resolved: [["service"], ["accept_default_resolvers", "boolean"]],
  fakeip: [["inet4_range"], ["inet6_range"]],
}

const defaultRuleFields: readonly CleanupField[] = [
  ["query_client_subnet", "list"], ["query_dnssec", "boolean"], ["match_response", "value"],
  ["response_rcode", "rcode"], ["response_answer", "list"], ["response_ns", "list"], ["response_extra", "list"],
  ["package_name_regex", "list"], ["source_mac_address", "list"], ["source_hostname", "list"], ["preferred_by", "list"],
  ["inbound", "list"], ["ip_version", "number"], ["query_type", "query-type"], ["network", "list"],
  ["auth_user", "list"], ["protocol", "list"], ["domain", "list"], ["domain_suffix", "list"],
  ["domain_keyword", "list"], ["domain_regex", "list"], ["geosite", "list"],
  ["source_geoip", "list"], ["geoip", "list"], ["ip_cidr", "list"], ["ip_is_private", "boolean"],
  ["ip_accept_any", "boolean"], ["source_ip_cidr", "list"], ["source_ip_is_private", "boolean"],
  ["source_port", "number-list"], ["source_port_range", "list"], ["port", "number-list"],
  ["port_range", "list"], ["process_name", "list"], ["process_path", "list"],
  ["process_path_regex", "list"], ["package_name", "list"], ["user", "list"], ["user_id", "number-list"],
  ["outbound", "list"], ["clash_mode"], ["network_type", "list"], ["network_is_expensive", "boolean"],
  ["network_is_constrained", "boolean"], ["wifi_ssid", "list"], ["wifi_bssid", "list"],
  ["interface_address", "json-object"], ["network_interface_address", "json-object"],
  ["default_interface_address", "list"], ["rule_set", "list"],
  ["rule_set_ip_cidr_match_source", "boolean"], ["rule_set_ip_cidr_accept_empty", "boolean"],
  ["rule_set_ipcidr_match_source", "boolean"], ["invert", "boolean"],
]

export const dnsRuleCleanupFields: CleanupRegistry = {
  default: defaultRuleFields,
  logical: [["mode"], ["rules", "json-array"], ["invert", "boolean"]],
}

const routeOptions: readonly CleanupField[] = [["race", "boolean"], ["timeout"], ["disable_cache", "boolean"],
  ["disable_optimistic_cache", "boolean"], ["rewrite_ttl", "number"], ["client_subnet"], ["remove_client_subnet", "boolean"]]

export const dnsActionCleanupFields: CleanupRegistry = {
  route: [["server"], ["speculative", "boolean"], ...routeOptions],
  evaluate: [["server"], ["tag"], ["speculative", "boolean"], ...routeOptions],
  respond: [["race", "boolean"]],
  "route-options": routeOptions,
  reject: [["race", "boolean"], ["method"], ["no_drop", "boolean"]],
  predefined: [["race", "boolean"], ["rcode", "rcode"], ["answer", "list"], ["ns", "list"], ["extra", "list"]],
}

function validInteger(value: unknown, maximum = Number.MAX_SAFE_INTEGER): boolean {
  return typeof value === "number" && Number.isFinite(value) && Number.isInteger(value) && value >= 0 && value <= maximum
}

function matchesField(value: unknown, field: CleanupField): boolean {
  const kind = field[1] ?? "string"
  if (kind === "value") return true
  if (kind === "query-type") {
    const valid = (item: unknown) => typeof item === "string" || validInteger(item, 0xFFFF)
    return valid(value) || Array.isArray(value) && value.every(valid)
  }
  if (kind === "rcode") return typeof value === "string" || validInteger(value, 0xFFF)
  if (kind === "routing-mark") return typeof value === "string" || validInteger(value, 0xFFFFFFFF)
  if (kind === "boolean") return typeof value === "boolean"
  if (kind === "number") return validInteger(value)
  if (kind === "list") return typeof value === "string" || Array.isArray(value) && value.every((item) => typeof item === "string")
  if (kind === "number-list") return validInteger(value) || Array.isArray(value) && value.every((item) => validInteger(item))
  if (kind === "json-object") return value !== null && typeof value === "object" && !Array.isArray(value)
  if (kind === "json-array") return Array.isArray(value)
  return typeof value === "string"
}

function uniqueFields(registry: CleanupRegistry): CleanupField[] {
  const fields = new Map<string, CleanupField>()
  for (const group of Object.values(registry)) for (const field of group) if (!fields.has(field[0])) fields.set(field[0], field)
  return [...fields.values()]
}

interface CleanupTransition {
  current: string
  target: string
  share?: boolean
}

function compatibleFields(registry: CleanupRegistry, change: CleanupTransition): CleanupField[] {
  if (change.share === false) return []
  const target = registry[change.target] ?? []
  if (!Object.hasOwn(registry, change.current)) return [...target]
  return target.filter((field) => registry[change.current].some((source) => source[0] === field[0] && source[1] === field[1]))
}

function supportsDomainResolver(registry: CleanupRegistry, type: string): boolean {
  return (registry[type] ?? []).some((field) => field[0].startsWith("domain_resolver."))
}

function prepareDomainResolverRoot(
  object: JsonObject,
  registry: CleanupRegistry,
  change: CleanupTransition,
): JsonObject {
  if (!Object.keys(registry).some((type) => supportsDomainResolver(registry, type))) return object
  const value = object.domain_resolver
  if (value === undefined || isJsonObject(value)) return object
  if (typeof value === "string" && supportsDomainResolver(registry, change.target)) return object
  return setPolicyPath(object, "domain_resolver", undefined)
}

export function transitionCleanupFields(
  object: JsonObject,
  registry: CleanupRegistry,
  change: CleanupTransition,
): JsonObject {
  const compatible = new Map(compatibleFields(registry, change).map((field) => [field[0], field]))
  const prepared = prepareDomainResolverRoot(object, registry, change)
  return uniqueFields(registry).reduce((next, field) => {
    const value = getPolicyPath(next, field[0])
    if (value === undefined) return next
    const target = compatible.get(field[0])
    return target && matchesField(value, target) ? next : setPolicyPath(next, field[0], undefined)
  }, prepared)
}
