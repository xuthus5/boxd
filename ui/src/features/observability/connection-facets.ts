/** Connection list facet helpers (network/protocol/inbound/outbound/rule/process + query). */

import type { ConnectionSortKey } from "@/features/observability/connection-export"
import { matchesConnection } from "@/features/observability/connection-stats"
import type { Connection } from "@/lib/api/types"

export type ConnectionFacetField = "network" | "protocol" | "inbound" | "outbound" | "rule" | "process"

export type ConnectionView = "list" | "outbound" | "rule" | "process"

export type ConnectionFacetFilters = {
  query?: string
  network?: string
  protocol?: string
  inbound?: string
  outbound?: string
  rule?: string
  process?: string
  view?: ConnectionView
  sort?: ConnectionSortKey
}

export type ConnectionFacetOption = {
  value: string
  count: number
}

const UNKNOWN = "—"

export function connectionFacetValue(connection: Connection, field: ConnectionFacetField): string {
  const raw = field === "network"
    ? connection.network
    : field === "protocol"
      ? connection.protocol
      : field === "inbound"
        ? connection.inbound
        : field === "outbound"
          ? connection.outbound
          : field === "rule"
            ? connection.rule
            : connection.process
  const value = raw?.trim()
  return value ? value : UNKNOWN
}

export function listConnectionFacets(
  connections: readonly Connection[],
  field: ConnectionFacetField,
): ConnectionFacetOption[] {
  const buckets = new Map<string, number>()
  for (const connection of connections) {
    const key = connectionFacetValue(connection, field)
    buckets.set(key, (buckets.get(key) ?? 0) + 1)
  }
  return [...buckets.entries()]
    .map(([value, count]) => ({ value, count }))
    .sort((left, right) => {
      if (right.count !== left.count) return right.count - left.count
      if (left.value === UNKNOWN) return 1
      if (right.value === UNKNOWN) return -1
      return left.value.localeCompare(right.value)
    })
}

/** Facet options for one field, counted after applying all other filters. */
export function listScopedConnectionFacets(
  connections: readonly Connection[],
  filters: ConnectionFacetFilters,
  field: ConnectionFacetField,
): ConnectionFacetOption[] {
  const scoped = filterConnectionsByFacets(connections, { ...filters, [field]: undefined })
  return listConnectionFacets(scoped, field)
}

export type ConnectionFacetSummarySection = {
  field: ConnectionFacetField
  options: ConnectionFacetOption[]
}

const SUMMARY_FIELDS: ConnectionFacetField[] = [
  "network",
  "protocol",
  "inbound",
  "outbound",
  "rule",
  "process",
]

export function summarizeConnectionFacets(
  connections: readonly Connection[],
  filters: ConnectionFacetFilters,
  limit = 4,
): ConnectionFacetSummarySection[] {
  if (connections.length === 0) return []
  return SUMMARY_FIELDS.flatMap((field) => {
    const options = listScopedConnectionFacets(connections, filters, field)
      .filter((option) => option.value !== UNKNOWN)
      .slice(0, limit)
    return options.length > 0 ? [{ field, options }] : []
  })
}

export function matchesConnectionFacet(
  connection: Connection,
  field: ConnectionFacetField,
  selected: string | undefined,
): boolean {
  if (!selected) return true
  return connectionFacetValue(connection, field) === selected
}

export function filterConnectionsByFacets<T extends Connection>(
  connections: readonly T[],
  filters: ConnectionFacetFilters,
): T[] {
  const query = filters.query?.trim().toLowerCase() ?? ""
  return connections.filter((connection) => (
    matchesConnection(connection, query)
    && matchesConnectionFacet(connection, "network", filters.network)
    && matchesConnectionFacet(connection, "protocol", filters.protocol)
    && matchesConnectionFacet(connection, "inbound", filters.inbound)
    && matchesConnectionFacet(connection, "outbound", filters.outbound)
    && matchesConnectionFacet(connection, "rule", filters.rule)
    && matchesConnectionFacet(connection, "process", filters.process)
  ))
}

export function connectionFiltersActive(filters: ConnectionFacetFilters): boolean {
  return Boolean(
    filters.query?.trim()
    || filters.network
    || filters.protocol
    || filters.inbound
    || filters.outbound
    || filters.rule
    || filters.process,
  )
}

const CONNECTION_VIEWS = new Set<ConnectionView>(["list", "outbound", "rule", "process"])
const CONNECTION_SORTS = new Set<ConnectionSortKey>(["traffic", "rate", "download", "upload", "duration", "target", "outbound"])

export function parseConnectionSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): ConnectionFacetFilters {
  const read = (key: string) => {
    const value = params.get(key)?.trim()
    return value ? value : undefined
  }
  const viewRaw = read("view")
  const view = viewRaw && CONNECTION_VIEWS.has(viewRaw as ConnectionView)
    ? (viewRaw as ConnectionView)
    : undefined
  const sortRaw = read("sort")
  const sort = sortRaw && CONNECTION_SORTS.has(sortRaw as ConnectionSortKey)
    ? (sortRaw as ConnectionSortKey)
    : undefined
  return {
    query: read("q"),
    network: read("network"),
    protocol: read("protocol"),
    inbound: read("inbound"),
    outbound: read("outbound"),
    rule: read("rule"),
    process: read("process"),
    view,
    sort,
  }
}

export function toConnectionSearchParams(filters: ConnectionFacetFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  if (filters.network) params.set("network", filters.network)
  if (filters.protocol) params.set("protocol", filters.protocol)
  if (filters.inbound) params.set("inbound", filters.inbound)
  if (filters.outbound) params.set("outbound", filters.outbound)
  if (filters.rule) params.set("rule", filters.rule)
  if (filters.process) params.set("process", filters.process)
  if (filters.view && filters.view !== "list") params.set("view", filters.view)
  if (filters.sort && filters.sort !== "traffic") params.set("sort", filters.sort)
  return params
}

export function buildConnectionsHref(filters: ConnectionFacetFilters = {}): string {
  const qs = toConnectionSearchParams(filters).toString()
  return qs ? `/observability/connections?${qs}` : "/observability/connections"
}


export type ConnectionFacetLinkField = "network" | "protocol" | "inbound" | "outbound" | "rule" | "process"

export function facetHref(field: ConnectionFacetLinkField, value?: string) {
  const trimmed = value?.trim()
  if (!trimmed || trimmed === "—") return ""
  return buildConnectionsHref({ [field]: trimmed })
}

export {
  connectionTargetLogQuery,
  logConnectionQuery,
  logConnectionsHref,
  logDNSHref,
  logDNSServerTag,
} from "@/features/observability/connection-log-links"
