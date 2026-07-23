import type { Connection } from "@/lib/api/types"

export interface ConnectionGroupStat {
  key: string
  count: number
  upload: number
  download: number
}

export function summarizeConnections(connections: Connection[]) {
  const upload = connections.reduce((sum, item) => sum + (item.upload || 0), 0)
  const download = connections.reduce((sum, item) => sum + (item.download || 0), 0)
  const outbounds = new Set(connections.map((item) => item.outbound).filter(Boolean)).size
  return { upload, download, outbounds }
}

export function aggregateConnections(
  connections: Connection[],
  field: "outbound" | "rule",
  limit = 8,
): ConnectionGroupStat[] {
  const buckets = new Map<string, ConnectionGroupStat>()
  for (const connection of connections) {
    const raw = field === "outbound" ? connection.outbound : connection.rule
    const key = (raw && raw.trim()) || "—"
    const current = buckets.get(key) ?? { key, count: 0, upload: 0, download: 0 }
    current.count += 1
    current.upload += connection.upload || 0
    current.download += connection.download || 0
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
    String(connection.id),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function connectionIds(connections: Connection[]) {
  return connections.map((connection) => String(connection.id))
}

export function filterConnectionsByGroup(
  connections: Connection[],
  field: "outbound" | "rule",
  key: string,
) {
  return connections.filter((connection) => {
    const raw = field === "outbound" ? connection.outbound : connection.rule
    const value = (raw && raw.trim()) || "—"
    return value === key
  })
}
