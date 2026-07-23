import type { Subscription } from "@/lib/api/types"

export type SubscriptionFilter = "all" | "error" | "ok"

export function sortSubscriptions(items: Subscription[]) {
  return [...items].sort((left, right) => {
    const leftError = Boolean(left.error)
    const rightError = Boolean(right.error)
    if (leftError !== rightError) return leftError ? -1 : 1
    const delta = new Date(right.last_updated).getTime() - new Date(left.last_updated).getTime()
    if (delta !== 0) return delta
    return left.id.localeCompare(right.id)
  })
}

export function matchesSubscription(item: Subscription, query: string) {
  if (!query) return true
  const haystack = [
    item.name,
    item.url,
    item.error ?? "",
    item.error_code ?? "",
    item.id,
    ...(item.outbounds?.map((outbound) => outbound.tag) ?? []),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function filterSubscriptions(
  items: Subscription[],
  options: { query?: string; status?: SubscriptionFilter } = {},
) {
  const query = options.query?.trim().toLowerCase() ?? ""
  const status = options.status ?? "all"
  return sortSubscriptions(items).filter((item) => {
    if (status === "error" && !item.error) return false
    if (status === "ok" && item.error) return false
    return matchesSubscription(item, query)
  })
}

export function failedSubscriptionIds(items: Subscription[]) {
  return items.filter((item) => Boolean(item.error)).map((item) => item.id)
}

export type SubscriptionListFilters = {
  query?: string
  status?: SubscriptionFilter
}

const STATUS_VALUES = new Set<SubscriptionFilter>(["all", "error", "ok"])

function readParam(params: { get(name: string): string | null }, key: string): string | undefined {
  const value = params.get(key)?.trim()
  return value ? value : undefined
}

export function parseSubscriptionSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): SubscriptionListFilters {
  const query = readParam(params, "q")
  const statusRaw = readParam(params, "status") as SubscriptionFilter | undefined
  const status = statusRaw && STATUS_VALUES.has(statusRaw) ? statusRaw : undefined
  return { query, status }
}

export function toSubscriptionSearchParams(filters: SubscriptionListFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  if (filters.status && filters.status !== "all") params.set("status", filters.status)
  return params
}

export function buildSubscriptionsHref(filters: SubscriptionListFilters = {}): string {
  const qs = toSubscriptionSearchParams(filters).toString()
  return qs ? `/subscriptions?${qs}` : "/subscriptions"
}

export function subscriptionFiltersActive(filters: SubscriptionListFilters): boolean {
  return Boolean(filters.query?.trim() || (filters.status && filters.status !== "all"))
}


export type SubscriptionStatusSummary = {
  total: number
  ok: number
  error: number
}

export function summarizeSubscriptionStatus(
  items: readonly Subscription[],
  query = "",
): SubscriptionStatusSummary {
  const summary: SubscriptionStatusSummary = { total: 0, ok: 0, error: 0 }
  const normalized = query.trim().toLowerCase()
  for (const item of items) {
    if (!matchesSubscription(item, normalized)) continue
    summary.total += 1
    if (item.error) summary.error += 1
    else summary.ok += 1
  }
  return summary
}
