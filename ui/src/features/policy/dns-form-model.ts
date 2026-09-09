import { hasRuleMatchConditions } from "@/features/policy/policy-match-complete"
import { dns114GlobalFields, dns114MatchFields, dns114RouteOptions, dnsRaceField, dnsSpeculativeField } from "@/features/policy/dns114-fields"
import {
  getPolicyPath,
  isJsonObject,
  pruneInvisiblePolicyFields,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
  type PolicyFieldTransform,
} from "@/features/policy/policy-form-model"
import {
  createPolicyNumberTransform,
  type PolicyNumberConstraints,
} from "@/features/policy/policy-number-transform"
import {
  dnsActionCleanupFields,
  dnsRuleCleanupFields,
  dnsServerCleanupFields,
  transitionCleanupFields,
} from "@/features/policy/dns-form-cleanup"

import { dnsServerFields } from "@/features/policy/dns-form-server-fields"
export { dnsServerFields }

const domainStrategies = ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"] as const

export const dnsGlobalFields = [
  ...dns114GlobalFields,
  { path: "final", label: "final", kind: "ref", ref: "dns-server", section: "basic" },
  { path: "strategy", label: "strategy", kind: "select", options: domainStrategies, section: "basic" },
  { path: "client_subnet", label: "clientSubnet", section: "basic" },
  { path: "disable_cache", label: "disableCache", kind: "boolean", section: "cache" },
  { path: "disable_expire", label: "disableExpire", kind: "boolean", section: "cache" },
  { path: "cache_capacity", label: "cacheCapacity", kind: "number", section: "cache" },
  { path: "reverse_mapping", label: "reverseMapping", kind: "boolean", section: "cache" },
] as const satisfies readonly PolicyFieldSpec[]

export const dnsServerTypes = [
  "local", "hosts", "udp", "tcp", "tls", "quic", "https", "h3", "dhcp", "fakeip", "mdns", "tailscale", "resolved", "openvpn", "openconnect",
] as const

export const dnsRuleMatchFields = [
  ...dns114MatchFields,
  { path: "type", label: "type", kind: "select", options: ["default", "logical"], section: "basic" },
  { path: "inbound", label: "inbound", kind: "ref-multi", ref: "inbound", section: "basic" },
  { path: "ip_version", label: "ipVersion", kind: "select", options: ["4", "6"], section: "basic" },
  { path: "query_type", label: "queryType", kind: "list", section: "basic" },
  { path: "network", label: "network", kind: "network-multi", section: "basic" },
  { path: "auth_user", label: "authUser", kind: "list", section: "basic" },
  { path: "protocol", label: "protocol", kind: "list", section: "basic" },
  { path: "domain", label: "domain", kind: "list", section: "domain" },
  { path: "domain_suffix", label: "domainSuffix", kind: "list", section: "domain" },
  { path: "domain_keyword", label: "domainKeyword", kind: "list", section: "domain" },
  { path: "domain_regex", label: "domainRegex", kind: "list", section: "domain" },
  { path: "source_ip_cidr", label: "sourceIPCIDR", kind: "list", section: "domain" },
  { path: "source_ip_is_private", label: "sourceIPIsPrivate", kind: "boolean", section: "domain" },
  { path: "ip_cidr", label: "ipCIDR", kind: "list", section: "domain" },
  { path: "ip_is_private", label: "ipIsPrivate", kind: "boolean", section: "domain" },
  { path: "ip_accept_any", label: "ipAcceptAny", kind: "boolean", section: "domain" },
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
  { path: "clash_mode", label: "clashMode", section: "process" },
  { path: "rule_set", label: "ruleSet", kind: "ref-multi", ref: "rule-set", section: "process" },
  { path: "rule_set_ip_cidr_match_source", label: "ruleSetIPCIDRMatchSource", kind: "boolean", section: "process" },
  { path: "network_type", label: "networkType", kind: "list", section: "process" },
  { path: "network_is_expensive", label: "networkIsExpensive", kind: "boolean", section: "process" },
  { path: "network_is_constrained", label: "networkIsConstrained", kind: "boolean", section: "process" },
  { path: "interface_address", label: "interfaceAddress", kind: "json-object", section: "process" },
  { path: "network_interface_address", label: "networkInterfaceAddress", kind: "json-object", section: "process" },
  { path: "default_interface_address", label: "defaultInterfaceAddress", kind: "list", section: "process" },
  { path: "wifi_ssid", label: "wifiSSID", kind: "list", section: "process" },
  { path: "wifi_bssid", label: "wifiBSSID", kind: "list", section: "process" },
  { path: "invert", label: "invert", kind: "boolean", section: "basic" },
] as const satisfies readonly PolicyFieldSpec[]

export const dnsActions = ["route", "evaluate", "respond", "route-options", "reject", "predefined"] as const

const routeOptionFields = [
  ...dns114RouteOptions,
  dnsRaceField,
  { path: "disable_cache", label: "disableCache", kind: "boolean", section: "action" },
  { path: "rewrite_ttl", label: "rewriteTTL", kind: "number", section: "action" },
  { path: "client_subnet", label: "clientSubnet", section: "action" },
] as const satisfies readonly PolicyFieldSpec[]

export const dnsActionFields: Record<string, readonly PolicyFieldSpec[]> = {
  route: [{ path: "server", label: "server", kind: "ref", ref: "dns-server", section: "action" }, dnsSpeculativeField, ...routeOptionFields],
  evaluate: [{ path: "server", label: "server", kind: "ref", ref: "dns-server", section: "action", required: true },
    { path: "tag", label: "evaluateTag", section: "action" }, dnsSpeculativeField, ...routeOptionFields],
  respond: [dnsRaceField],
  "route-options": routeOptionFields,
  reject: [
    dnsRaceField,
    { path: "method", label: "rejectMethod", kind: "select", options: ["default", "drop", "reply"], section: "action" },
    { path: "no_drop", label: "rejectNoDrop", kind: "boolean", section: "action" },
  ],
  predefined: [
    dnsRaceField,
    { path: "rcode", label: "rcode", section: "action" },
    { path: "answer", label: "answer", kind: "list", section: "action" },
    { path: "ns", label: "nameServer", kind: "list", section: "action" },
    { path: "extra", label: "extra", kind: "list", section: "action" },
  ],
}

const queryTypeNames = new Map([
  "A AAAA AFSDB AMTRELAY ANY APL ATMA AVC AXFR CAA CDNSKEY CDS CERT CNAME CSYNC DHCID DLV DNAME DNSKEY DS",
  "EID EUI48 EUI64 GID GPOS HINFO HIP HTTPS IPSECKEY ISDN IXFR KEY KX L32 L64 LOC LP MAILA MAILB MB MD MF MG",
  "MINFO MR MX NAPTR NID NIMLOC NINFO NS NSEC NSEC3 NSEC3PARAM NULL NXNAME NXT None OPENPGPKEY OPT PTR PX",
  "RESINFO RKEY RP RRSIG RT Reserved SIG SMIMEA SOA SPF SRV SSHFP SVCB TA TALINK TKEY TLSA TSIG TXT UID UINFO",
  "UNSPEC URI X25 ZONEMD NSAP-PTR",
].join(" ").split(" ").map((name) => [name.toUpperCase(), name]))
const rcodeNames = new Set(
  "NOERROR FORMERR SERVFAIL NXDOMAIN NOTIMP REFUSED YXDOMAIN YXRRSET NXRRSET NOTAUTH NOTZONE DSOTYPENI BADSIG BADKEY BADTIME BADMODE BADNAME BADALG BADTRUNC BADCOOKIE".split(" "),
)

function decimalInteger(raw: string, maximum: number): number | null {
  if (!/^\d+$/.test(raw)) return null
  const value = Number(raw)
  return Number.isSafeInteger(value) && value <= maximum ? value : null
}

function queryTypeToken(raw: string): string | number | null {
  const numeric = decimalInteger(raw, 0xFFFF)
  if (numeric !== null) return numeric
  return queryTypeNames.get(raw.toUpperCase()) ?? null
}

function transformQueryType(object: JsonObject, raw: string): JsonObject | null {
  const tokens = raw.split(/[\n,]/).map((token) => token.trim()).filter(Boolean)
  if (tokens.length === 0) return setPolicyPath(object, "query_type", undefined)
  const values = tokens.map(queryTypeToken)
  if (values.some((value) => value === null)) return null
  const current = getPolicyPath(object, "query_type")
  const value = Array.isArray(current) || values.length > 1 || current === undefined ? values : values[0]
  return setPolicyPath(object, "query_type", value as string | number | (string | number)[])
}

function transformRCode(object: JsonObject, raw: string, path = "rcode"): JsonObject | null {
  const token = raw.trim()
  if (!token) return setPolicyPath(object, path, undefined)
  const numeric = decimalInteger(token, 0xFFF)
  if (numeric !== null) return setPolicyPath(object, path, numeric)
  const name = token.toUpperCase()
  return rcodeNames.has(name) ? setPolicyPath(object, path, name) : null
}

const dnsNumberConstraints: PolicyNumberConstraints = {
  ip_version: { kind: "integer", maximum: 6, allowed: [4, 6] },
  source_port: { kind: "integer-list", maximum: 0xFFFF },
  port: { kind: "integer-list", maximum: 0xFFFF },
  user_id: { kind: "integer-list", maximum: 0x7FFFFFFF },
  server_port: { kind: "integer", maximum: 0xFFFF },
  cache_capacity: { kind: "integer", maximum: 0xFFFFFFFF },
  rewrite_ttl: { kind: "integer", maximum: 0xFFFFFFFF },
  "domain_resolver.rewrite_ttl": { kind: "integer", maximum: 0xFFFFFFFF },
  routing_mark: { kind: "mark", maximum: 0xFFFFFFFF },
}
const transformDNSNumber = createPolicyNumberTransform(dnsNumberConstraints)

export const transformDNSField: PolicyFieldTransform = (object, field, raw) => {
  if (field.path === "query_type") return transformQueryType(object, raw)
  if (field.path === "rcode" || field.path === "response_rcode") return transformRCode(object, raw, field.path)
  return transformDNSNumber(object, field, raw)
}

export function dnsServers(object: JsonObject): JsonObject[] {
  return Array.isArray(object.servers) ? object.servers.filter(isJsonObject) : []
}

export function dnsRules(object: JsonObject): JsonObject[] {
  return Array.isArray(object.rules) ? object.rules.filter(isJsonObject) : []
}

export function setDNSServers(object: JsonObject, servers: readonly JsonObject[]): JsonObject {
  return { ...object, servers: [...servers] }
}

export function setDNSRules(object: JsonObject, rules: readonly JsonObject[]): JsonObject {
  return { ...object, rules: [...rules] }
}

function dnsActionComplete(rule: JsonObject): boolean {
  const action = String(rule.action ?? "route")
  return !["route", "evaluate"].includes(action) || typeof rule.server === "string" && rule.server.trim().length > 0
}

export function isDNSRuleComplete(rule: JsonObject, depth = 0): boolean {
  if (depth === 0 && !dnsActionComplete(rule)) return false
  const actionKeys = ["action", "strategy", ...Object.values(dnsActionFields).flat().map((field) => field.path)]
  if (depth > 0 && actionKeys.some((path) => Object.hasOwn(rule, path))) return false
  if (rule.type !== "logical") return depth === 0 || hasRuleMatchConditions(rule)
  if (depth >= 64 || typeof rule.mode !== "string" || rule.mode.length === 0 || !Array.isArray(rule.rules) || rule.rules.length === 0) return false
  return rule.rules.every((child) => isJsonObject(child) && isDNSRuleComplete(child, depth + 1))
}

export function inferDNSServerType(server: JsonObject): string {
  return typeof server.type === "string" ? server.type : ""
}

export function changeDNSServerType(server: JsonObject, type: string): JsonObject {
  const current = inferDNSServerType(server)
  if (current === type) return server
  const next = transitionCleanupFields(server, dnsServerCleanupFields, { current, target: type })
  return setPolicyPath(next, "type", type)
}

export function changeDNSRuleType(rule: JsonObject, type: string): JsonObject {
  const current = String(rule.type ?? "default")
  if (current === type) return rule
  const next = transitionCleanupFields(rule, dnsRuleCleanupFields, { current, target: type })
  return setPolicyPath(next, "type", type === "default" ? undefined : type)
}

export function changeDNSAction(rule: JsonObject, action: string): JsonObject {
  const explicit = String(rule.action ?? "")
  if (explicit === action && explicit) return rule
  const current = explicit || "route"
  const next = transitionCleanupFields(rule, dnsActionCleanupFields, { current, target: action })
  return setPolicyPath(next, "action", action || undefined)
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function stringListValue(value: unknown): string {
  if (typeof value === "string") return value
  if (!Array.isArray(value)) return ""
  return value.filter((item): item is string => typeof item === "string").join(", ")
}

export interface DNSServerSummaryLabels {
  path: (value: string) => string
  predefined: (count: number) => string
  ipv4: (value: string) => string
  ipv6: (value: string) => string
  tag: (value: string) => string
  detour: (value: string) => string
  strategy: (value: string) => string
}

export interface DNSRuleSummaryLabels { logicalMode: (value: string) => string; matchLabel: (path: string) => string }

const defaultServerSummaryLabels: DNSServerSummaryLabels = {
  path: (value) => `path ${value}`,
  predefined: (count) => `predefined ${count}`,
  ipv4: (value) => `inet4 ${value}`,
  ipv6: (value) => `inet6 ${value}`,
  tag: (value) => `tag ${value}`,
  detour: (value) => `detour ${value}`,
  strategy: (value) => `strategy ${value}`,
}

const defaultRuleSummaryLabels: DNSRuleSummaryLabels = {
  logicalMode: (value) => `mode:${value}`,
  matchLabel: (path) => path,
}

function serverTypeDetails(server: JsonObject, type: string, labels: DNSServerSummaryLabels): string[] {
  if (type === "hosts") {
    const path = stringListValue(server.path)
    const predefined = isJsonObject(server.predefined) ? Object.keys(server.predefined).length : 0
    return [path ? labels.path(path) : "", predefined ? labels.predefined(predefined) : ""]
  }
  if (type === "fakeip") {
    const inet4 = stringValue(server.inet4_range)
    const inet6 = stringValue(server.inet6_range)
    return [inet4 ? labels.ipv4(inet4) : "", inet6 ? labels.ipv6(inet6) : ""]
  }
  return []
}

export function summarizeDNSServer(server: JsonObject, labels = defaultServerSummaryLabels): { type: string; detail: string } {
  const type = inferDNSServerType(server)
  const host = stringValue(server.server)
  const port = typeof server.server_port === "number" && Number.isFinite(server.server_port) ? `:${server.server_port}` : ""
  const primary = host ? `${host}${port}` : stringValue(server.interface) || stringValue(server.endpoint) || stringValue(server.service)
  const details = [primary, ...serverTypeDetails(server, type, labels)]
  if (stringValue(server.tag)) details.push(labels.tag(String(server.tag)))
  if (stringValue(server.detour)) details.push(labels.detour(String(server.detour)))
  if (stringValue(server.strategy)) details.push(labels.strategy(String(server.strategy)))
  return { type, detail: details.filter(Boolean).join(" · ") }
}

const summaryPaths = [
  ...dns114MatchFields.map((field) => field.path),
  "domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr", "source_ip_is_private",
  "ip_cidr", "ip_is_private", "ip_accept_any", "source_port", "source_port_range", "port", "port_range", "process_name",
  "process_path", "process_path_regex", "package_name", "user", "user_id", "rule_set",
  "rule_set_ip_cidr_match_source", "rule_set_ip_cidr_accept_empty", "inbound", "ip_version", "query_type", "network", "auth_user", "protocol",
  "outbound", "clash_mode", "network_type", "network_is_expensive", "network_is_constrained", "wifi_ssid",
  "interface_address", "network_interface_address", "default_interface_address", "wifi_bssid", "invert",
]

function summarizeValue(path: string, value: JsonObject[string] | undefined, labels: DNSRuleSummaryLabels): string[] {
  if (Array.isArray(value)) return value.flatMap((item) => typeof item === "string" || typeof item === "number" ? [String(item)] : [])
  if (isJsonObject(value) && Object.keys(value).length > 0) return [labels.matchLabel(path)]
  if (typeof value === "string" || typeof value === "number") return [String(value)]
  return value === true ? [labels.matchLabel(path)] : []
}

export function summarizeDNSRule(rule: JsonObject, labels = defaultRuleSummaryLabels): { matches: string[]; action: string } {
  const logical = rule.type === "logical" && typeof rule.mode === "string" ? [labels.logicalMode(rule.mode)] : []
  const matches = [...logical, ...summaryPaths.flatMap((path) => summarizeValue(path, rule[path], labels))]
  const action = String(rule.action ?? "route")
  const target = action === "route" && typeof rule.server === "string" && rule.server ? ` · ${rule.server}` : ""
  return { matches, action: `${action}${target}` }
}

export function applyDNSGlobalFieldChange(_object: JsonObject, next: JsonObject) {
  return pruneInvisiblePolicyFields(next, dnsGlobalFields)
}

export function applyDNSRuleFieldChange(object: JsonObject, next: JsonObject) {
  const action = String(next.action ?? object.action ?? "route")
  const actionFields = dnsActionFields[action] ?? []
  return pruneInvisiblePolicyFields(next, [...dnsRuleMatchFields, ...actionFields])
}

export function applyDNSServerFieldChange(type: string, next: JsonObject) {
  return pruneInvisiblePolicyFields(next, dnsServerFields[type] ?? [])
}
