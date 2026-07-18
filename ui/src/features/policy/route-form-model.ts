import {
  getPolicyPath,
  isJsonObject,
  pruneInvisiblePolicyFields,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"

const domainStrategies = ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"] as const
const networkStrategies = ["default", "fallback", "hybrid"] as const
const keepAliveOn = { path: "disable_tcp_keep_alive", falsy: true } as const
const tlsFragmentOn = { path: "tls_fragment", is: true } as const
const resolverOn = { path: "default_domain_resolver.server" } as const
const actionResolverOn = { path: "domain_resolver.server" } as const

export const routeGlobalFields = [
  { path: "final", label: "final", kind: "ref", ref: "outbound", section: "basic" },
  { path: "find_process", label: "findProcess", kind: "boolean", section: "basic" },
  { path: "auto_detect_interface", label: "autoDetectInterface", kind: "boolean", section: "interface" },
  { path: "override_android_vpn", label: "overrideAndroidVPN", kind: "boolean", section: "interface" },
  { path: "default_interface", label: "defaultInterface", kind: "network-interface", section: "interface" },
  { path: "default_mark", label: "defaultMark", section: "interface" },
  { path: "default_domain_resolver.server", label: "defaultDomainResolverServer", kind: "ref", ref: "dns-server", section: "dns" },
  { path: "default_domain_resolver.strategy", label: "defaultDomainResolverStrategy", kind: "select", options: domainStrategies, section: "dns", when: resolverOn },
  { path: "default_domain_resolver.disable_cache", label: "defaultDomainResolverDisableCache", kind: "boolean", section: "dns", when: resolverOn },
  { path: "default_domain_resolver.rewrite_ttl", label: "defaultDomainResolverRewriteTTL", kind: "number", section: "dns", when: resolverOn },
  { path: "default_domain_resolver.client_subnet", label: "defaultDomainResolverClientSubnet", section: "dns", when: resolverOn },
  { path: "default_network_strategy", label: "defaultNetworkStrategy", kind: "select", options: networkStrategies, section: "strategy" },
  { path: "default_network_type", label: "defaultNetworkType", kind: "list", section: "strategy" },
  { path: "default_fallback_network_type", label: "defaultFallbackNetworkType", kind: "list", section: "strategy" },
  { path: "default_fallback_delay", label: "defaultFallbackDelay", section: "strategy" },
] as const satisfies readonly PolicyFieldSpec[]

export const routeMatchFields = [
  { path: "type", label: "type", kind: "select", options: ["default", "logical"], section: "basic" },
  { path: "inbound", label: "inbound", kind: "ref-multi", ref: "inbound", section: "basic" },
  { path: "ip_version", label: "ipVersion", kind: "select", options: ["4", "6"], section: "basic" },
  { path: "network", label: "network", kind: "network-multi", section: "basic" },
  { path: "auth_user", label: "authUser", kind: "list", section: "basic" },
  { path: "protocol", label: "protocol", kind: "list", section: "basic" },
  { path: "client", label: "client", kind: "list", section: "basic" },
  { path: "domain", label: "domain", kind: "list", section: "domain" },
  { path: "domain_suffix", label: "domainSuffix", kind: "list", section: "domain" },
  { path: "domain_keyword", label: "domainKeyword", kind: "list", section: "domain" },
  { path: "domain_regex", label: "domainRegex", kind: "list", section: "domain" },
  { path: "source_ip_cidr", label: "sourceIPCIDR", kind: "list", section: "domain" },
  { path: "source_ip_is_private", label: "sourceIPIsPrivate", kind: "boolean", section: "domain" },
  { path: "ip_cidr", label: "ipCIDR", kind: "list", section: "domain" },
  { path: "ip_is_private", label: "ipIsPrivate", kind: "boolean", section: "domain" },
  { path: "source_port", label: "sourcePort", kind: "number-list", section: "process" },
  { path: "source_port_range", label: "sourcePortRange", kind: "list", section: "process" },
  { path: "port", label: "port", kind: "number-list", section: "process" },
  { path: "port_range", label: "portRange", kind: "list", section: "process" },
  { path: "process_name", label: "processName", kind: "list", section: "process" },
  { path: "process_path", label: "processPath", kind: "list", section: "process" },
  { path: "process_path_regex", label: "processPathRegex", kind: "list", section: "process" },
  { path: "package_name", label: "packageName", kind: "list", section: "process" },
  { path: "user", label: "user", kind: "list", section: "process" },
  { path: "user_id", label: "userID", kind: "number-list", section: "process" },
  { path: "rule_set", label: "ruleSet", kind: "ref-multi", ref: "rule-set", section: "environment" },
  { path: "rule_set_ip_cidr_match_source", label: "ruleSetIPCIDRMatchSource", kind: "boolean", section: "environment" },
  { path: "clash_mode", label: "clashMode", section: "environment" },
  { path: "network_type", label: "networkType", kind: "list", section: "environment" },
  { path: "network_is_expensive", label: "networkIsExpensive", kind: "boolean", section: "environment" },
  { path: "network_is_constrained", label: "networkIsConstrained", kind: "boolean", section: "environment" },
  { path: "wifi_ssid", label: "wifiSSID", kind: "list", section: "environment" },
  { path: "wifi_bssid", label: "wifiBSSID", kind: "list", section: "environment" },
  { path: "invert", label: "invert", kind: "boolean", section: "basic" },
] as const satisfies readonly PolicyFieldSpec[]

export const routeActions = [
  "route", "route-options", "direct", "bypass", "reject", "hijack-dns", "sniff", "resolve",
] as const

const routeOptionFields = [
  { path: "override_address", label: "overrideAddress", section: "action" },
  { path: "override_port", label: "overridePort", kind: "number", section: "action" },
  { path: "network_strategy", label: "networkStrategy", kind: "select", options: networkStrategies, section: "action" },
  { path: "fallback_delay", label: "fallbackDelay", kind: "number", section: "action" },
  { path: "udp_disable_domain_unmapping", label: "udpDisableDomainUnmapping", kind: "boolean", section: "action" },
  { path: "udp_connect", label: "udpConnect", kind: "boolean", section: "action" },
  { path: "udp_timeout", label: "udpTimeout", section: "action" },
  { path: "tls_fragment", label: "tlsFragment", kind: "boolean", section: "action" },
  { path: "tls_fragment_fallback_delay", label: "tlsFragmentFallbackDelay", section: "action", when: tlsFragmentOn },
  { path: "tls_record_fragment", label: "tlsRecordFragment", kind: "boolean", section: "action" },
] as const satisfies readonly PolicyFieldSpec[]

const directFields = [
  { path: "bind_interface", label: "bindInterface", kind: "network-interface", section: "action" },
  { path: "inet4_bind_address", label: "inet4BindAddress", section: "action" },
  { path: "inet6_bind_address", label: "inet6BindAddress", section: "action" },
  { path: "bind_address_no_port", label: "bindAddressNoPort", kind: "boolean", section: "action" },
  { path: "protect_path", label: "protectPath", section: "action" },
  { path: "routing_mark", label: "routingMark", section: "action" },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean", section: "action" },
  { path: "netns", label: "networkNamespace", section: "action" },
  { path: "connect_timeout", label: "connectTimeout", section: "action" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean", section: "action" },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean", section: "action" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean", section: "action" },
  { path: "tcp_keep_alive", label: "tcpKeepAlive", section: "action", when: keepAliveOn },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", section: "action", when: keepAliveOn },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean", section: "action" },
  { path: "domain_resolver.server", label: "domainResolverServer", kind: "ref", ref: "dns-server", section: "action" },
  { path: "domain_resolver.strategy", label: "domainResolverStrategy", kind: "select", options: domainStrategies, section: "action", when: actionResolverOn },
  { path: "domain_resolver.disable_cache", label: "domainResolverDisableCache", kind: "boolean", section: "action", when: actionResolverOn },
  { path: "domain_resolver.rewrite_ttl", label: "domainResolverRewriteTTL", kind: "number", section: "action", when: actionResolverOn },
  { path: "domain_resolver.client_subnet", label: "domainResolverClientSubnet", section: "action", when: actionResolverOn },
  { path: "network_strategy", label: "networkStrategy", kind: "select", options: networkStrategies, section: "action" },
  { path: "network_type", label: "networkType", kind: "list", section: "action" },
  { path: "fallback_network_type", label: "fallbackNetworkType", kind: "list", section: "action" },
  { path: "fallback_delay", label: "fallbackDelay", section: "action" },
] as const satisfies readonly PolicyFieldSpec[]

const routeFields = [
  { path: "outbound", label: "outbound", kind: "ref", ref: "outbound", section: "action", required: true },
  ...routeOptionFields,
] as const satisfies readonly PolicyFieldSpec[]

export const routeActionFields: Record<string, readonly PolicyFieldSpec[]> = {
  route: routeFields,
  "route-options": routeOptionFields,
  direct: directFields,
  bypass: routeFields,
  reject: [
    { path: "method", label: "rejectMethod", kind: "select", options: ["default", "drop", "reply"], section: "action" },
    { path: "no_drop", label: "rejectNoDrop", kind: "boolean", section: "action" },
  ],
  "hijack-dns": [],
  sniff: [
    { path: "sniffer", label: "sniffer", kind: "list", section: "action" },
    { path: "timeout", label: "sniffTimeout", section: "action" },
  ],
  resolve: [
    { path: "server", label: "resolveServer", kind: "ref", ref: "dns-server", section: "action" },
    { path: "strategy", label: "resolveStrategy", kind: "select", options: domainStrategies, section: "action" },
    { path: "disable_cache", label: "resolveDisableCache", kind: "boolean", section: "action" },
    { path: "rewrite_ttl", label: "resolveRewriteTTL", kind: "number", section: "action" },
    { path: "client_subnet", label: "resolveClientSubnet", section: "action" },
  ],
}

export const ruleSetTypes = ["inline", "local", "remote"] as const

const logicalRuleFields: readonly PolicyFieldSpec[] = [
  { path: "mode", label: "logicalMode", kind: "select", options: ["and", "or"], section: "basic" },
  { path: "rules", label: "logicalRules", kind: "json-array", section: "basic" },
  { path: "invert", label: "invert", kind: "boolean", section: "basic" },
]
const defaultRuleFields = routeMatchFields.filter((field) => field.path !== "type")
const ruleTypeFields: Record<string, readonly PolicyFieldSpec[]> = { default: defaultRuleFields, logical: logicalRuleFields }
const ruleSetFields: Record<string, readonly PolicyFieldSpec[]> = {
  inline: [{ path: "rules", label: "rules", kind: "json-array" }],
  local: [{ path: "format", label: "format" }, { path: "path", label: "path" }],
  remote: [
    { path: "format", label: "format" }, { path: "url", label: "url" },
    { path: "download_detour", label: "downloadDetour", kind: "ref", ref: "outbound" }, { path: "update_interval", label: "updateInterval" },
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
  if (field.kind === "list" || field.kind === "network-multi" || field.kind === "ref-multi") {
    return typeof value === "string" || Array.isArray(value) && value.every((item) => typeof item === "string")
  }
  if (field.kind === "number-list") return typeof value === "number" && Number.isFinite(value)
    || Array.isArray(value) && value.every((item) => typeof item === "number" && Number.isFinite(item))
  if (field.kind === "json-object") return value !== null && typeof value === "object" && !Array.isArray(value)
  if (field.kind === "json-array") return Array.isArray(value)
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean"
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
const routeMatchPaths = new Set<string>(routeMatchFields.map((field) => field.path))
const knownActionFields = uniqueFields(Object.values(routeActionFields))
  .filter((field) => !routeMatchPaths.has(field.path))
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

export function managedRouteGlobalFields() {
  return routeGlobalFields.filter((field) => field.path !== "final")
}

export function applyRouteGlobalFieldChange(_object: JsonObject, next: JsonObject) {
  return pruneInvisiblePolicyFields(next, managedRouteGlobalFields())
}

export function managedRouteRuleFields(type: string, action: string) {
  const match = type === "logical" ? logicalRuleFields : defaultRuleFields
  return [...match, ...(routeActionFields[action] ?? [])]
}

export function applyRouteRuleFieldChange(object: JsonObject, next: JsonObject) {
  const type = String(next.type ?? object.type ?? "default")
  const action = String(next.action ?? object.action ?? "route")
  return pruneInvisiblePolicyFields(next, managedRouteRuleFields(type, action))
}
