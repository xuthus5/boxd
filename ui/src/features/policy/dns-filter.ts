import {
  inferDNSServerType,
  summarizeDNSRule,
  summarizeDNSServer,
} from "@/features/policy/dns-form-model"
import type { JsonObject } from "@/features/policy/policy-form-model"

function text(value: unknown) {
  return typeof value === "string" || typeof value === "number" ? String(value) : ""
}

export function dnsServerType(item: JsonObject) {
  const value = inferDNSServerType(item).trim()
  return value || "legacy"
}

export function dnsRuleAction(item: JsonObject) {
  const value = text(item.action).trim()
  return value || "route"
}

export function matchesDNSServer(item: JsonObject, query: string) {
  if (!query) return true
  const summary = summarizeDNSServer(item)
  const haystack = [
    text(item.tag),
    text(item.type),
    inferDNSServerType(item),
    summary.type,
    summary.detail,
    text(item.server),
    text(item.server_port),
    text(item.address),
    text(item.detour),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function matchesDNSServerType(item: JsonObject, type: string | undefined) {
  if (!type) return true
  return dnsServerType(item).toLowerCase() === type.toLowerCase()
}

export function matchesDNSRule(item: JsonObject, query: string) {
  if (!query) return true
  const summary = summarizeDNSRule(item)
  const haystack = [
    summary.action,
    ...summary.matches,
    text(item.type),
    text(item.action),
    text(item.server),
    text(item.mode),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function matchesDNSRuleAction(item: JsonObject, action: string | undefined) {
  if (!action) return true
  return dnsRuleAction(item).toLowerCase() === action.toLowerCase()
}

export type DNSSearchFilters = {
  servers?: string
  rules?: string
  serverType?: string
  ruleAction?: string
}

export type DNSTypeBucket = {
  type: string
  count: number
}

export type DNSTypeSummary = {
  total: number
  buckets: DNSTypeBucket[]
}

export type DNSActionBucket = {
  action: string
  count: number
}

export type DNSActionSummary = {
  total: number
  buckets: DNSActionBucket[]
}

function sortBuckets<T extends { count: number }>(
  buckets: T[],
  key: (bucket: T) => string,
) {
  return buckets.sort((left, right) => {
    if (right.count !== left.count) return right.count - left.count
    return key(left).localeCompare(key(right))
  })
}

export function summarizeDNSServerTypes(
  items: readonly JsonObject[],
  query = "",
): DNSTypeSummary {
  const normalized = query.trim().toLowerCase()
  const counts = new Map<string, number>()
  let total = 0
  for (const item of items) {
    if (!matchesDNSServer(item, normalized)) continue
    total += 1
    const type = dnsServerType(item)
    counts.set(type, (counts.get(type) ?? 0) + 1)
  }
  return {
    total,
    buckets: sortBuckets(
      Array.from(counts.entries()).map(([type, count]) => ({ type, count })),
      (bucket) => bucket.type,
    ),
  }
}

export function summarizeDNSRuleActions(
  items: readonly JsonObject[],
  query = "",
): DNSActionSummary {
  const normalized = query.trim().toLowerCase()
  const counts = new Map<string, number>()
  let total = 0
  for (const item of items) {
    if (!matchesDNSRule(item, normalized)) continue
    total += 1
    const action = dnsRuleAction(item)
    counts.set(action, (counts.get(action) ?? 0) + 1)
  }
  return {
    total,
    buckets: sortBuckets(
      Array.from(counts.entries()).map(([action, count]) => ({ action, count })),
      (bucket) => bucket.action,
    ),
  }
}

export function filterDNSServers(
  items: readonly JsonObject[],
  filters: Pick<DNSSearchFilters, "servers" | "serverType"> = {},
) {
  const query = filters.servers?.trim().toLowerCase() ?? ""
  return items.filter((item) => (
    matchesDNSServer(item, query) && matchesDNSServerType(item, filters.serverType)
  ))
}

export function filterDNSRules(
  items: readonly JsonObject[],
  filters: Pick<DNSSearchFilters, "rules" | "ruleAction"> = {},
) {
  const query = filters.rules?.trim().toLowerCase() ?? ""
  return items.filter((item) => (
    matchesDNSRule(item, query) && matchesDNSRuleAction(item, filters.ruleAction)
  ))
}

export function dnsServerFiltersActive(filters: Pick<DNSSearchFilters, "servers" | "serverType">): boolean {
  return Boolean(filters.servers?.trim() || filters.serverType)
}

export function dnsRuleFiltersActive(filters: Pick<DNSSearchFilters, "rules" | "ruleAction">): boolean {
  return Boolean(filters.rules?.trim() || filters.ruleAction)
}

function readParam(params: { get(name: string): string | null }, key: string): string | undefined {
  const value = params.get(key)?.trim()
  return value ? value : undefined
}

export function parseDNSSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): DNSSearchFilters {
  return {
    servers: readParam(params, "sq"),
    rules: readParam(params, "rq"),
    serverType: readParam(params, "stype"),
    ruleAction: readParam(params, "raction"),
  }
}

export function toDNSSearchParams(filters: DNSSearchFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const servers = filters.servers?.trim()
  const rules = filters.rules?.trim()
  const serverType = filters.serverType?.trim()
  const ruleAction = filters.ruleAction?.trim()
  if (servers) params.set("sq", servers)
  if (rules) params.set("rq", rules)
  if (serverType) params.set("stype", serverType)
  if (ruleAction) params.set("raction", ruleAction)
  return params
}

export function buildDNSHref(filters: DNSSearchFilters = {}): string {
  const qs = toDNSSearchParams(filters).toString()
  return qs ? `/policy/dns?${qs}` : "/policy/dns"
}
