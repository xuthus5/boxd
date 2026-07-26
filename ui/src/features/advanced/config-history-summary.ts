import { configApplyEventFailed } from "@/features/dashboard/config-apply-source"
import type { ConfigApplyEvent } from "@/lib/api/types"

export interface ConfigHistorySummary {
  total: number
  applied: number
  validated: number
  failed: number
  restorable: number
  current: number
}

export function summarizeConfigHistory(events: readonly ConfigApplyEvent[]): ConfigHistorySummary {
  const summary: ConfigHistorySummary = {
    total: events.length,
    applied: 0,
    validated: 0,
    failed: 0,
    restorable: 0,
    current: 0,
  }
  for (const event of events) {
    const status = event.status.trim()
    if (status === "validated") summary.validated += 1
    else if (configApplyEventFailed(status)) summary.failed += 1
    else if (status === "applied") summary.applied += 1
    if (event.current) summary.current += 1
    if (event.id.trim() && event.restorable && !event.current && !configApplyEventFailed(status)) {
      summary.restorable += 1
    }
  }
  return summary
}
