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
