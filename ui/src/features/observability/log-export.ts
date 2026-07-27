import type { LogEvent } from "@/lib/api/types"

export function formatLogLine(item: LogEvent): string {
  const time = item.timestamp?.trim() || "-"
  const level = item.level?.trim() || "-"
  const message = item.message ?? ""
  return `${time}\t${level}\t${message}`
}

export function formatLogMessage(item: LogEvent): string {
  return (item.message ?? "").trim()
}

export function formatLogTimestamp(timestamp?: string): string {
  if (!timestamp) return "—"
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString()
}

export function formatLogExport(items: readonly LogEvent[]): string {
  return items.map(formatLogLine).join("\n")
}

export function buildLogExportFilename(source: string, now = new Date()): string {
  const stamp = now.toISOString().replace(/[:.]/g, "-")
  const safe = source.trim().toLowerCase().replace(/[^a-z0-9_-]+/g, "-") || "logs"
  return `boxd-${safe}-${stamp}.log`
}

export function downloadTextFile(filename: string, text: string, doc: Document = document): void {
  const blob = new Blob([text], { type: "text/plain;charset=utf-8" })
  const url = URL.createObjectURL(blob)
  const anchor = doc.createElement("a")
  anchor.href = url
  anchor.download = filename
  anchor.rel = "noopener"
  doc.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
