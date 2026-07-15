import {
  getPolicyPath,
  isJsonObject,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"

const domainStrategies = ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"] as const
const networkStrategies = ["default", "fallback", "hybrid"] as const

export const routeGlobalFields = [
  { path: "final", label: "final" },
  { path: "find_process", label: "findProcess", kind: "boolean" },
  { path: "auto_detect_interface", label: "autoDetectInterface", kind: "boolean" },
  { path: "override_android_vpn", label: "overrideAndroidVPN", kind: "boolean" },
  { path: "default_interface", label: "defaultInterface" },
  { path: "default_mark", label: "defaultMark" },
  { path: "default_domain_resolver.server", label: "defaultDomainResolverServer" },
  { path: "default_domain_resolver.strategy", label: "defaultDomainResolverStrategy", kind: "select", options: domainStrategies },
  { path: "default_domain_resolver.disable_cache", label: "defaultDomainResolverDisableCache", kind: "boolean" },
  { path: "default_domain_resolver.rewrite_ttl", label: "defaultDomainResolverRewriteTTL", kind: "number" },
  { path: "default_domain_resolver.client_subnet", label: "defaultDomainResolverClientSubnet" },
  { path: "default_network_strategy", label: "defaultNetworkStrategy", kind: "select", options: networkStrategies },
  { path: "default_network_type", label: "defaultNetworkType", kind: "list" },
  { path: "default_fallback_network_type", label: "defaultFallbackNetworkType", kind: "list" },
  { path: "default_fallback_delay", label: "defaultFallbackDelay" },
] as const satisfies readonly PolicyFieldSpec[]

export const routeMatchFields = [
  { path: "type", label: "type", kind: "select", options: ["default", "logical"] },
  { path: "inbound", label: "inbound", kind: "list" },
  { path: "ip_version", label: "ipVersion", kind: "number" },
  { path: "network", label: "network", kind: "list" },
  { path: "auth_user", label: "authUser", kind: "list" },
  { path: "protocol", label: "protocol", kind: "list" },
  { path: "client", label: "client", kind: "list" },
  { path: "domain", label: "domain", kind: "list" },
  { path: "domain_suffix", label: "domainSuffix", kind: "list" },
  { path: "domain_keyword", label: "domainKeyword", kind: "list" },
  { path: "domain_regex", label: "domainRegex", kind: "list" },
  { path: "source_ip_cidr", label: "sourceIPCIDR", kind: "list" },
  { path: "source_ip_is_private", label: "sourceIPIsPrivate", kind: "boolean" },
  { path: "ip_cidr", label: "ipCIDR", kind: "list" },
  { path: "ip_is_private", label: "ipIsPrivate", kind: "boolean" },
  { path: "source_port", label: "sourcePort", kind: "number-list" },
  { path: "source_port_range", label: "sourcePortRange", kind: "list" },
  { path: "port", label: "port", kind: "number-list" },
  { path: "port_range", label: "portRange", kind: "list" },
  { path: "process_name", label: "processName", kind: "list" },
  { path: "process_path", label: "processPath", kind: "list" },
  { path: "process_path_regex", label: "processPathRegex", kind: "list" },
  { path: "package_name", label: "packageName", kind: "list" },
  { path: "user", label: "user", kind: "list" },
  { path: "user_id", label: "userID", kind: "number-list" },
  { path: "rule_set", label: "ruleSet", kind: "list" },
  { path: "rule_set_ip_cidr_match_source", label: "ruleSetIPCIDRMatchSource", kind: "boolean" },
  { path: "clash_mode", label: "clashMode" },
  { path: "network_type", label: "networkType", kind: "list" },
  { path: "network_is_expensive", label: "networkIsExpensive", kind: "boolean" },
  { path: "network_is_constrained", label: "networkIsConstrained", kind: "boolean" },
  { path: "wifi_ssid", label: "wifiSSID", kind: "list" },
  { path: "wifi_bssid", label: "wifiBSSID", kind: "list" },
  { path: "invert", label: "invert", kind: "boolean" },
] as const satisfies readonly PolicyFieldSpec[]

export const routeActions = [
  "route", "route-options", "direct", "bypass", "reject", "hijack-dns", "sniff", "resolve",
] as const

const routeOptionFields = [
  { path: "override_address", label: "overrideAddress" },
  { path: "override_port", label: "overridePort", kind: "number" },
  { path: "network_strategy", label: "networkStrategy", kind: "select", options: networkStrategies },
  { path: "fallback_delay", label: "fallbackDelay", kind: "number" },
  { path: "udp_disable_domain_unmapping", label: "udpDisableDomainUnmapping", kind: "boolean" },
  { path: "udp_connect", label: "udpConnect", kind: "boolean" },
  { path: "udp_timeout", label: "udpTimeout" },
  { path: "tls_fragment", label: "tlsFragment", kind: "boolean" },
  { path: "tls_fragment_fallback_delay", label: "tlsFragmentFallbackDelay" },
  { path: "tls_record_fragment", label: "tlsRecordFragment", kind: "boolean" },
] as const satisfies readonly PolicyFieldSpec[]

const directFields = [
  { path: "bind_interface", label: "bindInterface" },
  { path: "inet4_bind_address", label: "inet4BindAddress" },
  { path: "inet6_bind_address", label: "inet6BindAddress" },
  { path: "bind_address_no_port", label: "bindAddressNoPort", kind: "boolean" },
  { path: "protect_path", label: "protectPath" },
  { path: "routing_mark", label: "routingMark" },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean" },
  { path: "netns", label: "networkNamespace" },
  { path: "connect_timeout", label: "connectTimeout" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean" },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean" },
  { path: "tcp_keep_alive", label: "tcpKeepAlive" },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval" },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean" },
  { path: "domain_resolver.server", label: "domainResolverServer" },
  { path: "domain_resolver.strategy", label: "domainResolverStrategy", kind: "select", options: domainStrategies },
  { path: "domain_resolver.disable_cache", label: "domainResolverDisableCache", kind: "boolean" },
  { path: "domain_resolver.rewrite_ttl", label: "domainResolverRewriteTTL", kind: "number" },
  { path: "domain_resolver.client_subnet", label: "domainResolverClientSubnet" },
  { path: "network_strategy", label: "networkStrategy", kind: "select", options: networkStrategies },
  { path: "network_type", label: "networkType", kind: "list" },
  { path: "fallback_network_type", label: "fallbackNetworkType", kind: "list" },
  { path: "fallback_delay", label: "fallbackDelay" },
] as const satisfies readonly PolicyFieldSpec[]

const routeFields = [{ path: "outbound", label: "outbound" }, ...routeOptionFields]

export const routeActionFields: Record<string, readonly PolicyFieldSpec[]> = {
  route: routeFields,
  "route-options": routeOptionFields,
  direct: directFields,
  bypass: routeFields,
  reject: [
    { path: "method", label: "rejectMethod", kind: "select", options: ["default", "drop", "reply"] },
    { path: "no_drop", label: "rejectNoDrop", kind: "boolean" },
  ],
  "hijack-dns": [],
  sniff: [
    { path: "sniffer", label: "sniffer", kind: "list" },
    { path: "timeout", label: "sniffTimeout" },
  ],
  resolve: [
    { path: "server", label: "resolveServer" },
    { path: "strategy", label: "resolveStrategy", kind: "select", options: domainStrategies },
    { path: "disable_cache", label: "resolveDisableCache", kind: "boolean" },
    { path: "rewrite_ttl", label: "resolveRewriteTTL", kind: "number" },
    { path: "client_subnet", label: "resolveClientSubnet" },
  ],
}

export const ruleSetTypes = ["inline", "local", "remote"] as const

const logicalRuleFields: readonly PolicyFieldSpec[] = [
  { path: "mode", label: "logicalMode", kind: "select", options: ["and", "or"] },
  { path: "rules", label: "logicalRules", kind: "json-array" },
  { path: "invert", label: "invert", kind: "boolean" },
]
const defaultRuleFields = routeMatchFields.filter((field) => field.path !== "type")
const ruleTypeFields: Record<string, readonly PolicyFieldSpec[]> = { default: defaultRuleFields, logical: logicalRuleFields }
const ruleSetFields: Record<string, readonly PolicyFieldSpec[]> = {
  inline: [{ path: "rules", label: "rules", kind: "json-array" }],
  local: [{ path: "format", label: "format" }, { path: "path", label: "path" }],
  remote: [
    { path: "format", label: "format" }, { path: "url", label: "url" },
    { path: "download_detour", label: "downloadDetour" }, { path: "update_interval", label: "updateInterval" },
  ],
}

function uniqueFields(groups: readonly (readonly PolicyFieldSpec[])[]): PolicyFieldSpec[] {
  const fields = new Map<string, PolicyFieldSpec>()
  for (const group of groups) for (const field of group) if (!fields.has(field.path)) fields.set(field.path, field)
  return [...fields.values()]
}

function matchesField(value: unknown, field: PolicyFieldSpec): boolean {
  if (value === undefined) return true
  if (field.kind === "boolean") return typeof value === "boolean"
  if (field.kind === "number") return typeof value === "number" && Number.isFinite(value)
  if (field.kind === "list") return typeof value === "string" || Array.isArray(value) && value.every((item) => typeof item === "string")
  if (field.kind === "number-list") return typeof value === "number" && Number.isFinite(value)
    || Array.isArray(value) && value.every((item) => typeof item === "number" && Number.isFinite(item))
  if (field.kind === "json-object") return value !== null && typeof value === "object" && !Array.isArray(value)
  if (field.kind === "json-array") return Array.isArray(value)
  return typeof value === "string"
}

interface FieldTransition {
  source: readonly PolicyFieldSpec[]
  target: readonly PolicyFieldSpec[]
  sourceKnown: boolean
}

function fieldTransition(fields: Record<string, readonly PolicyFieldSpec[]>, source: string, target: string): FieldTransition {
  const sourceKnown = Object.hasOwn(fields, source)
  return { source: sourceKnown ? fields[source] : [], target: fields[target] ?? [], sourceKnown }
}

function compatibleFields(transition: FieldTransition): Map<string, PolicyFieldSpec> {
  const fields = transition.sourceKnown
    ? transition.target.filter((field) => transition.source.some((source) => source.path === field.path && source.kind === field.kind))
    : transition.target
  return new Map(fields.map((field) => [field.path, field]))
}

function transitionFields(object: JsonObject, known: readonly PolicyFieldSpec[], transition: FieldTransition): JsonObject {
  const compatible = compatibleFields(transition)
  return known.reduce((next, field) => {
    const target = compatible.get(field.path)
    return target && matchesField(getPolicyPath(next, field.path), target)
    ? next
    : setPolicyPath(next, field.path, undefined)
  }, object)
}

const knownRuleFields = uniqueFields(Object.values(ruleTypeFields))
const knownActionFields = uniqueFields(Object.values(routeActionFields))
const knownRuleSetFields = uniqueFields(Object.values(ruleSetFields))

export function routeRules(object: JsonObject): JsonObject[] {
  return Array.isArray(object.rules) ? object.rules.filter(isJsonObject) : []
}

export function routeRuleSets(object: JsonObject): JsonObject[] {
  return Array.isArray(object.rule_set) ? object.rule_set.filter(isJsonObject) : []
}

export function setRouteRules(object: JsonObject, rules: readonly JsonObject[]): JsonObject {
  return { ...object, rules: [...rules] }
}

export function setRouteRuleSets(object: JsonObject, ruleSets: readonly JsonObject[]): JsonObject {
  return { ...object, rule_set: [...ruleSets] }
}

export function changeRouteRuleType(rule: JsonObject, type: string): JsonObject {
  const current = String(rule.type ?? "default")
  if (current === type) return rule
  const next = transitionFields(rule, knownRuleFields, fieldTransition(ruleTypeFields, current, type))
  return setPolicyPath(next, "type", type === "default" ? undefined : type)
}

export function changeRouteAction(rule: JsonObject, action: string): JsonObject {
  const explicit = String(rule.action ?? "")
  if (explicit === action && explicit !== "") return rule
  const current = explicit || "route"
  const next = transitionFields(rule, knownActionFields, fieldTransition(routeActionFields, current, action))
  return setPolicyPath(next, "action", action || undefined)
}

export function changeRuleSetType(ruleSet: JsonObject, type: string): JsonObject {
  const current = String(ruleSet.type ?? "inline")
  if (current === type) return ruleSet
  const next = transitionFields(ruleSet, knownRuleSetFields, fieldTransition(ruleSetFields, current, type))
  return setPolicyPath(next, "type", type)
}

const summaryPaths = [
  "domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr", "source_ip_is_private",
  "ip_cidr", "ip_is_private", "source_port", "source_port_range", "port", "port_range", "process_name",
  "process_path", "process_path_regex", "package_name", "user", "user_id", "rule_set",
  "rule_set_ip_cidr_match_source", "inbound", "ip_version",
  "network", "auth_user", "protocol", "client", "clash_mode", "network_type", "network_is_expensive",
  "network_is_constrained", "wifi_ssid", "wifi_bssid", "invert",
]

export interface RouteRuleSummaryLabels {
  matchLabel: (path: string) => string
}

const defaultRouteRuleSummaryLabels: RouteRuleSummaryLabels = { matchLabel: (path) => path }

function summarizeValue(path: string, value: unknown, labels: RouteRuleSummaryLabels): string[] {
  if (Array.isArray(value)) return value.flatMap((item) => typeof item === "string" || typeof item === "number" ? [String(item)] : [])
  if (typeof value === "string" || typeof value === "number") return [String(value)]
  return value === true ? [labels.matchLabel(path)] : []
}

export function summarizeRouteRule(
  rule: JsonObject,
  labels = defaultRouteRuleSummaryLabels,
): { matches: string[]; action: string } {
  const matches = summaryPaths.flatMap((path) => summarizeValue(path, rule[path], labels))
  const action = String(rule.action ?? "route")
  return { matches, action: action === "route" && typeof rule.outbound === "string" ? rule.outbound : action }
}

export function summarizeRuleSet(ruleSet: JsonObject): { type: string; detail: string } {
  const baseType = typeof ruleSet.type === "string" && ruleSet.type ? ruleSet.type : "inline"
  const format = typeof ruleSet.format === "string" && ruleSet.format ? ` · ${ruleSet.format}` : ""
  const location = [ruleSet.url, ruleSet.path, ruleSet.tag].find((value) => typeof value === "string")
  return { type: `${baseType}${format}`, detail: typeof location === "string" ? location : "" }
}
