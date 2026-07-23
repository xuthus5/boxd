/** Connection list facet helpers (network/protocol/outbound/rule/process + query). */

import type { ConnectionSortKey } from "@/features/observability/connection-export"
import { matchesConnection } from "@/features/observability/connection-stats"
import type { Connection } from "@/lib/api/types"

export type ConnectionFacetField = "network" | "protocol" | "outbound" | "rule" | "process"

export type ConnectionView = "list" | "outbound" | "rule" | "process"

export type ConnectionFacetFilters = {
  query?: string
  network?: string
  protocol?: string
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
    && matchesConnectionFacet(connection, "process", filters.process)
  ))
}

export function connectionFiltersActive(filters: ConnectionFacetFilters): boolean {
  return Boolean(
    filters.query?.trim()
    || filters.network
    || filters.protocol
    || filters.outbound
    || filters.rule
    || filters.process,
  )
}

const CONNECTION_VIEWS = new Set<ConnectionView>(["list", "outbound", "rule", "process"])
const CONNECTION_SORTS = new Set<ConnectionSortKey>(["traffic", "download", "upload", "duration", "target", "outbound"])

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

const IPV4_HOST_PORT = /\b((?:\d{1,3}\.){3}\d{1,3})(?::(\d{1,5}))?\b/g
const BRACKETED_IPV6 = /\[([0-9a-fA-F:]+)\](?::(\d{1,5}))?/g
const DOMAIN_HOST_PORT = /\b((?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,})(?::(\d{1,5}))?\b/g

function isLikelyIPv4(value: string): boolean {
  const parts = value.split(".")
  if (parts.length !== 4) return false
  return parts.every((part) => {
    if (!/^\d{1,3}$/.test(part)) return false
    const n = Number(part)
    return n >= 0 && n <= 255
  })
}

function isLikelyPort(value: string | undefined): boolean {
  if (!value) return true
  if (!/^\d{1,5}$/.test(value)) return false
  const n = Number(value)
  return n >= 1 && n <= 65535
}

/** Extract a connection-search host from a sing-box style log message. */
export function logConnectionQuery(message: string | undefined): string {
  const raw = message?.trim() ?? ""
  if (!raw) return ""

  // Prefer destination hosts from "connection to ..."; fall back to "connection from ...".
  const connectionTo = raw.match(
    /\b(?:inbound|outbound)\s+connection\s+to\s+(\[?[\w.:-]+\]?)/i,
  )
  if (connectionTo?.[1]) return connectionTargetLogQuery(connectionTo[1])

  const connectionFrom = raw.match(
    /\b(?:inbound|outbound)\s+connection\s+from\s+(\[?[\w.:-]+\]?)/i,
  )
  if (connectionFrom?.[1]) {
    const host = connectionTargetLogQuery(connectionFrom[1])
    if (host && !host.startsWith("127.") && host !== "0.0.0.0" && host !== "::1") return host
  }

  // DNS-style: "lookup example.com" / "query example.com"
  const dnsMatch = raw.match(/\b(?:lookup|query|resolve)\s+([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})\b/i)
  if (dnsMatch?.[1]) return dnsMatch[1]

  // First bracketed IPv6 host
  BRACKETED_IPV6.lastIndex = 0
  const ipv6 = BRACKETED_IPV6.exec(raw)
  if (ipv6?.[1] && isLikelyPort(ipv6[2])) return ipv6[1]

  // First plausible domain
  DOMAIN_HOST_PORT.lastIndex = 0
  let domainHit: RegExpExecArray | null
  while ((domainHit = DOMAIN_HOST_PORT.exec(raw)) !== null) {
    const host = domainHit[1]
    if (!isLikelyPort(domainHit[2])) continue
    // Skip version-like tokens such as v1.2.3
    if (/^v?\d+(\.\d+)+$/.test(host)) continue
    return host
  }

  // First public-looking IPv4 (skip loopback/wildcard noise)
  IPV4_HOST_PORT.lastIndex = 0
  let ipHit: RegExpExecArray | null
  while ((ipHit = IPV4_HOST_PORT.exec(raw)) !== null) {
    const host = ipHit[1]
    if (!isLikelyIPv4(host) || !isLikelyPort(ipHit[2])) continue
    if (host.startsWith("127.") || host === "0.0.0.0") continue
    return host
  }

  return ""
}

/** Build a connections deep-link from a log message when a host can be extracted. */
export function logConnectionsHref(message: string | undefined): string {
  const query = logConnectionQuery(message)
  return query ? buildConnectionsHref({ query }) : ""
}

export type ConnectionFacetLinkField = "network" | "protocol" | "outbound" | "rule" | "process"

export function facetHref(field: ConnectionFacetLinkField, value?: string) {
  const trimmed = value?.trim()
  if (!trimmed || trimmed === "—") return ""
  return buildConnectionsHref({ [field]: trimmed })
}
