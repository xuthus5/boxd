import type { Connection } from "@/lib/api/types"
import { connectionRateTotal, type ConnectionWithRates } from "@/features/observability/connection-rate"

export interface ConnectionGroupStat {
  key: string
  count: number
  upload: number
  download: number
  uploadRate: number
  downloadRate: number
  rateSamples: number
}

export function summarizeConnections(connections: ConnectionWithRates[]) {
  const upload = connections.reduce((sum, item) => sum + (item.upload || 0), 0)
  const download = connections.reduce((sum, item) => sum + (item.download || 0), 0)
  const uploadRate = connections.reduce((sum, item) => sum + (item.uploadRate ?? 0), 0)
  const downloadRate = connections.reduce((sum, item) => sum + (item.downloadRate ?? 0), 0)
  const rateSamples = connections.filter((item) => connectionRateTotal(item) !== undefined).length
  const outbounds = new Set(connections.map((item) => item.outbound).filter(Boolean)).size
  return { upload, download, uploadRate, downloadRate, rateSamples, outbounds }
}

function groupFieldValue(connection: Connection, field: "outbound" | "rule" | "process"): string {
  const raw = field === "outbound"
    ? connection.outbound
    : field === "rule"
      ? connection.rule
      : connection.process
  return (raw && raw.trim()) || "—"
}

export function aggregateConnections(
  connections: ConnectionWithRates[],
  field: "outbound" | "rule" | "process",
  limit = 8,
): ConnectionGroupStat[] {
  const buckets = new Map<string, ConnectionGroupStat>()
  for (const connection of connections) {
    const key = groupFieldValue(connection, field)
    const current = buckets.get(key) ?? { key, count: 0, upload: 0, download: 0, uploadRate: 0, downloadRate: 0, rateSamples: 0 }
    current.count += 1
    current.upload += connection.upload || 0
    current.download += connection.download || 0
    current.uploadRate += connection.uploadRate ?? 0
    current.downloadRate += connection.downloadRate ?? 0
    if (connectionRateTotal(connection) !== undefined) current.rateSamples += 1
    buckets.set(key, current)
  }
  return [...buckets.values()]
    .sort((left, right) => {
      const traffic = (right.upload + right.download) - (left.upload + left.download)
      if (traffic !== 0) return traffic
      if (right.count !== left.count) return right.count - left.count
      return left.key.localeCompare(right.key)
    })
    .slice(0, limit)
}

export function matchesConnection(connection: Connection, query: string) {
  if (!query) return true
  const haystack = [
    connection.target,
    connection.outbound,
    connection.rule ?? "",
    connection.network ?? "",
    connection.source ?? "",
    connection.inbound ?? "",
    connection.protocol ?? "",
    connection.process ?? "",
    String(connection.id),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function connectionIds(connections: Connection[]) {
  return connections.map((connection) => String(connection.id))
}

export function filterConnectionsByGroup(
  connections: Connection[],
  field: "outbound" | "rule" | "process",
  key: string,
) {
  return connections.filter((connection) => groupFieldValue(connection, field) === key)
}
