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
