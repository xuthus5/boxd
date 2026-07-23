import type { JsonObject } from "@/features/policy/policy-form-model"
import { summarizeRouteRule } from "@/features/policy/route-form-model"
import type { RouteRuleMetadata } from "@/lib/api/types"

export function routeRuleAction(item: JsonObject) {
  const value = String(item.action ?? "").trim()
  return value || "route"
}

export function matchesRouteRule(
  item: JsonObject,
  metadata: RouteRuleMetadata | undefined,
  query: string,
  matchLabel?: (path: string) => string,
) {
  if (!query) return true
  const summary = summarizeRouteRule(item, { matchLabel: matchLabel ?? ((path) => path) })
  const haystack = [
    metadata?.name ?? "",
    metadata?.description ?? "",
    summary.action,
    ...summary.matches,
    String(item.type ?? ""),
    String(item.outbound ?? ""),
    String(item.action ?? ""),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function matchesRouteAction(item: JsonObject, action: string | undefined) {
  if (!action) return true
  return routeRuleAction(item).toLowerCase() === action.toLowerCase()
}

export type RouteSearchFilters = {
  query?: string
  action?: string
}

export type RouteActionBucket = {
  action: string
  count: number
}

export type RouteActionSummary = {
  total: number
  buckets: RouteActionBucket[]
}

export function summarizeRouteActions(
  items: readonly JsonObject[],
  query = "",
  metadata: readonly (RouteRuleMetadata | undefined)[] = [],
  matchLabel?: (path: string) => string,
): RouteActionSummary {
  const normalized = query.trim().toLowerCase()
  const counts = new Map<string, number>()
  let total = 0
  for (let index = 0; index < items.length; index += 1) {
    const item = items[index]
    if (!matchesRouteRule(item, metadata[index], normalized, matchLabel)) continue
    total += 1
    const action = routeRuleAction(item)
    counts.set(action, (counts.get(action) ?? 0) + 1)
  }
  const buckets = Array.from(counts.entries())
    .map(([action, count]) => ({ action, count }))
    .sort((left, right) => {
      if (right.count !== left.count) return right.count - left.count
      return left.action.localeCompare(right.action)
    })
  return { total, buckets }
}

export function filterRouteRules(
  items: readonly JsonObject[],
  filters: RouteSearchFilters = {},
  metadata: readonly (RouteRuleMetadata | undefined)[] = [],
  matchLabel?: (path: string) => string,
) {
  const query = filters.query?.trim().toLowerCase() ?? ""
  return items.filter((item, index) => (
    matchesRouteRule(item, metadata[index], query, matchLabel)
    && matchesRouteAction(item, filters.action)
  ))
}

export function routeFiltersActive(filters: RouteSearchFilters): boolean {
  return Boolean(filters.query?.trim() || filters.action)
}

function readParam(params: { get(name: string): string | null }, key: string): string | undefined {
  const value = params.get(key)?.trim()
  return value ? value : undefined
}

export function parseRouteSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): RouteSearchFilters {
  return {
    query: readParam(params, "q"),
    action: readParam(params, "action"),
  }
}

export function toRouteSearchParams(filters: RouteSearchFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  const action = filters.action?.trim()
  if (query) params.set("q", query)
  if (action) params.set("action", action)
  return params
}

export function buildRouteHref(filters: RouteSearchFilters = {}): string {
  const qs = toRouteSearchParams(filters).toString()
  return qs ? `/policy/route?${qs}` : "/policy/route"
}
