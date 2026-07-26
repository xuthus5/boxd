import type { ConfigApplyEvent } from "@/lib/api/types"
import { configApplyEventFailed } from "@/features/dashboard/config-apply-source"

export const configHistoryFilters = [
  "all",
  "applied",
  "validated",
  "failed",
  "restorable",
] as const

export type ConfigHistoryFilter = typeof configHistoryFilters[number]

export function isConfigHistoryFilter(value: string): value is ConfigHistoryFilter {
  return configHistoryFilters.includes(value as ConfigHistoryFilter)
}

export function filterConfigHistory(
  events: readonly ConfigApplyEvent[],
  query: string,
  filter: ConfigHistoryFilter,
): ConfigApplyEvent[] {
  const normalizedQuery = query.trim().toLowerCase()
  return events.filter((event) => matchesFilter(event, filter) && matchesQuery(event, normalizedQuery))
}

function matchesFilter(event: ConfigApplyEvent, filter: ConfigHistoryFilter): boolean {
  switch (filter) {
    case "applied":
      return event.status.trim() === "applied"
    case "validated":
      return event.status.trim() === "validated"
    case "failed":
      return configApplyEventFailed(event.status)
    case "restorable":
      return Boolean(event.id.trim()) && event.restorable === true && !event.current && !configApplyEventFailed(event.status)
    default:
      return true
  }
}

function matchesQuery(event: ConfigApplyEvent, query: string): boolean {
  if (!query) return true
  const searchable = [event.hash, event.source, event.status, event.error, event.error_code]
    .filter(Boolean)
    .join(" ")
    .toLowerCase()
  return searchable.includes(query)
}
