/** Connection list facet helpers (network/protocol/outbound/rule + query). */

import { matchesConnection } from "@/features/observability/connection-stats"
import type { Connection } from "@/lib/api/types"

export type ConnectionFacetField = "network" | "protocol" | "outbound" | "rule"

export type ConnectionFacetFilters = {
  query?: string
  network?: string
  protocol?: string
  outbound?: string
  rule?: string
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
      : field === "outbound"
        ? connection.outbound
        : connection.rule
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

export function matchesConnectionFacet(
  connection: Connection,
  field: ConnectionFacetField,
  selected: string | undefined,
): boolean {
  if (!selected) return true
  return connectionFacetValue(connection, field) === selected
}

export function filterConnectionsByFacets(
  connections: readonly Connection[],
  filters: ConnectionFacetFilters,
): Connection[] {
  const query = filters.query?.trim().toLowerCase() ?? ""
  return connections.filter((connection) => (
    matchesConnection(connection, query)
    && matchesConnectionFacet(connection, "network", filters.network)
    && matchesConnectionFacet(connection, "protocol", filters.protocol)
    && matchesConnectionFacet(connection, "outbound", filters.outbound)
    && matchesConnectionFacet(connection, "rule", filters.rule)
  ))
}

export function connectionFiltersActive(filters: ConnectionFacetFilters): boolean {
  return Boolean(
    filters.query?.trim()
    || filters.network
    || filters.protocol
    || filters.outbound
    || filters.rule,
  )
}

export function parseConnectionSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): ConnectionFacetFilters {
  const read = (key: string) => {
    const value = params.get(key)?.trim()
    return value ? value : undefined
  }
  return {
    query: read("q"),
    network: read("network"),
    protocol: read("protocol"),
    outbound: read("outbound"),
    rule: read("rule"),
  }
}

export function buildConnectionsHref(filters: ConnectionFacetFilters = {}): string {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  if (filters.network) params.set("network", filters.network)
  if (filters.protocol) params.set("protocol", filters.protocol)
  if (filters.outbound) params.set("outbound", filters.outbound)
  if (filters.rule) params.set("rule", filters.rule)
  const qs = params.toString()
  return qs ? `/observability/connections?${qs}` : "/observability/connections"
}


/** Build a log search query from a connection target (host:port or host). */
export function connectionTargetLogQuery(target: string | undefined): string {
  const raw = target?.trim() ?? ""
  if (!raw) return ""
  if (raw.startsWith("[")) {
    const close = raw.indexOf("]")
    if (close > 0) return raw.slice(1, close)
  }
  // host:port -> host (only when a single trailing numeric port exists)
  const match = raw.match(/^(.*):(\d+)$/)
  if (match && !match[1].includes(":")) return match[1]
  return raw
}

