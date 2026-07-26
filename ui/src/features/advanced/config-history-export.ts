import type { ConfigHistoryFilter } from "@/features/advanced/config-history-filter"
import type { ConfigApplyEvent } from "@/lib/api/types"

export interface ConfigHistoryExportOptions {
  query: string
  filter: ConfigHistoryFilter
  exportedAt?: Date
}

export function formatConfigHistoryExport(
  events: readonly ConfigApplyEvent[],
  options: ConfigHistoryExportOptions,
): string {
  const exportedAt = options.exportedAt ?? new Date()
  return `${JSON.stringify({
    exported_at: exportedAt.toISOString(),
    query: options.query.trim(),
    filter: options.filter,
    count: events.length,
    records: events,
  }, null, 2)}\n`
}

export function buildConfigHistoryExportFilename(now = new Date()): string {
  const stamp = now.toISOString().replace(/[:.]/g, "-")
  return `boxd-config-history-${stamp}.json`
}
