import { connectionRateTotal, type ConnectionWithRates } from "@/features/observability/connection-rate"

export type ConnectionSortKey = "traffic" | "rate" | "download" | "upload" | "duration" | "target" | "outbound"

function trafficTotal(item: ConnectionWithRates): number {
  return (item.upload || 0) + (item.download || 0)
}

function startTime(item: ConnectionWithRates): number {
  const value = new Date(item.start).getTime()
  return Number.isFinite(value) ? value : Number.POSITIVE_INFINITY
}

export function compareConnections(left: ConnectionWithRates, right: ConnectionWithRates, sort: ConnectionSortKey): number {
  switch (sort) {
    case "rate": {
      const leftRate = connectionRateTotal(left)
      const rightRate = connectionRateTotal(right)
      if (leftRate === undefined || rightRate === undefined) {
        if (leftRate === undefined && rightRate !== undefined) return 1
        if (leftRate !== undefined && rightRate === undefined) return -1
        return trafficTotal(right) - trafficTotal(left) || left.target.localeCompare(right.target)
      }
      return rightRate - leftRate || left.target.localeCompare(right.target)
    }
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
  connections: readonly ConnectionWithRates[],
  sort: ConnectionSortKey,
): ConnectionWithRates[] {
  return [...connections].sort((left, right) => compareConnections(left, right, sort))
}

export function formatConnectionLine(item: ConnectionWithRates): string {
  const rule = item.rule?.trim() || "-"
  return [
    String(item.id),
    item.target || "-",
    item.outbound || "-",
    rule,
    item.network?.trim() || "-",
    item.source?.trim() || "-",
    item.inbound?.trim() || "-",
    item.protocol?.trim() || "-",
    item.process?.trim() || "-",
    String(item.upload || 0),
    String(item.download || 0),
    item.uploadRate === undefined ? "" : String(item.uploadRate),
    item.downloadRate === undefined ? "" : String(item.downloadRate),
    item.start || "-",
  ].join("\t")
}

export function formatConnectionExport(items: readonly ConnectionWithRates[]): string {
  if (items.length === 0) return ""
  const header = "id\ttarget\toutbound\trule\tnetwork\tsource\tinbound\tprotocol\tprocess\tupload\tdownload\tupload_rate\tdownload_rate\tstart"
  return [header, ...items.map(formatConnectionLine)].join("\n")
}

export function buildConnectionExportFilename(now = new Date()): string {
  const stamp = now.toISOString().replace(/[:.]/g, "-")
  return `boxd-connections-${stamp}.log`
}

/** 连接诊断剪贴板文本，便于排障粘贴。 */
export function formatConnectionClipboardText(item: ConnectionWithRates): string {
  const lines = [
    `id: ${item.id}`,
    item.target?.trim() ? `target: ${item.target.trim()}` : "",
    item.outbound?.trim() ? `outbound: ${item.outbound.trim()}` : "",
    item.rule?.trim() ? `rule: ${item.rule.trim()}` : "",
    item.network?.trim() ? `network: ${item.network.trim()}` : "",
    item.source?.trim() ? `source: ${item.source.trim()}` : "",
    item.inbound?.trim() ? `inbound: ${item.inbound.trim()}` : "",
    item.protocol?.trim() ? `protocol: ${item.protocol.trim()}` : "",
    item.process?.trim() ? `process: ${item.process.trim()}` : "",
    `upload: ${item.upload || 0}`,
    `download: ${item.download || 0}`,
    item.uploadRate === undefined ? "" : `upload_rate: ${item.uploadRate}`,
    item.downloadRate === undefined ? "" : `download_rate: ${item.downloadRate}`,
    item.start?.trim() ? `start: ${item.start.trim()}` : "",
  ].filter(Boolean)
  return lines.join("\n")
}
