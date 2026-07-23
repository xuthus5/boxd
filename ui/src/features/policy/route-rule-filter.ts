import type { JsonObject } from "@/features/policy/policy-form-model"
import { summarizeRouteRule } from "@/features/policy/route-form-model"
import type { RouteRuleMetadata } from "@/lib/api/types"

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

export type RouteSearchFilters = {
  query?: string
}

export function parseRouteSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): RouteSearchFilters {
  const value = params.get("q")?.trim()
  return { query: value ? value : undefined }
}

export function toRouteSearchParams(filters: RouteSearchFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  return params
}

export function buildRouteHref(filters: RouteSearchFilters = {}): string {
  const qs = toRouteSearchParams(filters).toString()
  return qs ? `/policy/route?${qs}` : "/policy/route"
}

