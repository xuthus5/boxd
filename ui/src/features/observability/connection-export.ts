import type { Connection } from "@/lib/api/types"

export type ConnectionSortKey = "traffic" | "download" | "upload" | "duration" | "target" | "outbound"

function trafficTotal(item: Connection): number {
  return (item.upload || 0) + (item.download || 0)
}

function startTime(item: Connection): number {
  const value = new Date(item.start).getTime()
  return Number.isFinite(value) ? value : Number.POSITIVE_INFINITY
}

export function compareConnections(left: Connection, right: Connection, sort: ConnectionSortKey): number {
  switch (sort) {
    case "download":
      return (right.download || 0) - (left.download || 0) || left.target.localeCompare(right.target)
    case "upload":
      return (right.upload || 0) - (left.upload || 0) || left.target.localeCompare(right.target)
    case "duration":
      return startTime(left) - startTime(right) || left.target.localeCompare(right.target)
    case "target":
      return left.target.localeCompare(right.target)
    case "outbound":
      return left.outbound.localeCompare(right.outbound) || left.target.localeCompare(right.target)
    case "traffic":
    default:
      return trafficTotal(right) - trafficTotal(left) || left.target.localeCompare(right.target)
  }
}

export function sortConnections(
  connections: readonly Connection[],
  sort: ConnectionSortKey,
): Connection[] {
  return [...connections].sort((left, right) => compareConnections(left, right, sort))
}

export function formatConnectionLine(item: Connection): string {
  const rule = item.rule?.trim() || "-"
  return [
    String(item.id),
    item.target || "-",
    item.outbound || "-",
    rule,
    String(item.upload || 0),
    String(item.download || 0),
    item.start || "-",
  ].join("\t")
}

export function formatConnectionExport(items: readonly Connection[]): string {
  if (items.length === 0) return ""
  const header = "id\ttarget\toutbound\trule\tupload\tdownload\tstart"
  return [header, ...items.map(formatConnectionLine)].join("\n")
}

export function buildConnectionExportFilename(now = new Date()): string {
  const stamp = now.toISOString().replace(/[:.]/g, "-")
  return `boxd-connections-${stamp}.log`
}
