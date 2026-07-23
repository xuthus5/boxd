import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

function text(value: JsonValue | undefined) {
  return typeof value === "string" || typeof value === "number" ? String(value) : ""
}

export function proxyItemType(item: JsonObject) {
  const value = text(item.type).trim()
  return value || "unknown"
}

export function matchesProxyItem(item: JsonObject, query: string) {
  if (!query) return true
  const transport = typeof item.transport === "object" && item.transport && !Array.isArray(item.transport)
    ? text(item.transport.type)
    : ""
  const haystack = [
    text(item.tag),
    text(item.type),
    text(item.listen),
    text(item.interface_name),
    text(item.listen_port),
    text(item.server),
    text(item.server_port),
    transport,
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function matchesProxyType(item: JsonObject, type: string | undefined) {
  if (!type) return true
  return proxyItemType(item).toLowerCase() === type.toLowerCase()
}

export type ProxyListFilters = {
  query?: string
  type?: string
}

export type ProxyTypeBucket = {
  type: string
  count: number
}

export type ProxyTypeSummary = {
  total: number
  buckets: ProxyTypeBucket[]
}

export function summarizeProxyTypes(
  items: readonly JsonObject[],
  query = "",
): ProxyTypeSummary {
  const normalized = query.trim().toLowerCase()
  const counts = new Map<string, number>()
  let total = 0
  for (const item of items) {
    if (!matchesProxyItem(item, normalized)) continue
    total += 1
    const type = proxyItemType(item)
    counts.set(type, (counts.get(type) ?? 0) + 1)
  }
  const buckets = Array.from(counts.entries())
    .map(([type, count]) => ({ type, count }))
    .sort((left, right) => {
      if (right.count !== left.count) return right.count - left.count
      return left.type.localeCompare(right.type)
    })
  return { total, buckets }
}

export function filterProxyItems(
  items: readonly JsonObject[],
  filters: ProxyListFilters = {},
) {
  const query = filters.query?.trim().toLowerCase() ?? ""
  return items.filter((item) => matchesProxyItem(item, query) && matchesProxyType(item, filters.type))
}

export function proxyFiltersActive(filters: ProxyListFilters): boolean {
  return Boolean(filters.query?.trim() || filters.type)
}

function readParam(params: { get(name: string): string | null }, key: string): string | undefined {
  const value = params.get(key)?.trim()
  return value ? value : undefined
}

export function parseProxySearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): ProxyListFilters {
  return {
    query: readParam(params, "q"),
    type: readParam(params, "type"),
  }
}

export function toProxySearchParams(filters: ProxyListFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  const type = filters.type?.trim()
  if (query) params.set("q", query)
  if (type) params.set("type", type)
  return params
}

export function buildProxyHref(
  configKey: "inbounds" | "outbounds",
  filters: ProxyListFilters = {},
): string {
  const base = configKey === "inbounds" ? "/proxy/inbounds" : "/proxy/outbounds"
  const qs = toProxySearchParams(filters).toString()
  return qs ? `${base}?${qs}` : base
}
