/** Network/protocol facet helpers for the live connections list. */

import { matchesConnection } from "@/features/observability/connection-stats"
import type { Connection } from "@/lib/api/types"

export type ConnectionFacetField = "network" | "protocol"

export type ConnectionFacetFilters = {
  query?: string
  network?: string
  protocol?: string
}

export type ConnectionFacetOption = {
  value: string
  count: number
}

const UNKNOWN = "—"

export function connectionFacetValue(connection: Connection, field: ConnectionFacetField): string {
  const raw = field === "network" ? connection.network : connection.protocol
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
  ))
}

export function connectionFiltersActive(filters: ConnectionFacetFilters): boolean {
  return Boolean(filters.query?.trim() || filters.network || filters.protocol)
}
