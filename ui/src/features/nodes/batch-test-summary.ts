/** Summarize bulk node speed-test results for toast / UI feedback. */

import type { TestResult } from "@/lib/api/types"

export type BatchTestSummary = {
  total: number
  success: number
  failed: number
  avgLatencyMs?: number
  bestTag?: string
  bestLatencyMs?: number
  worstTag?: string
  worstLatencyMs?: number
}

export function summarizeBatchTestResults(results: readonly TestResult[] | undefined): BatchTestSummary {
  const list = results ?? []
  let success = 0
  let failed = 0
  let latencySum = 0
  let latencyCount = 0
  let bestTag: string | undefined
  let bestLatencyMs: number | undefined
  let worstTag: string | undefined
  let worstLatencyMs: number | undefined

  for (const item of list) {
    if (item.success) {
      success += 1
      const latency = item.latency_ms
      if (typeof latency === "number" && Number.isFinite(latency)) {
        latencySum += latency
        latencyCount += 1
        if (bestLatencyMs === undefined || latency < bestLatencyMs) {
          bestLatencyMs = latency
          bestTag = item.tag
        }
        if (worstLatencyMs === undefined || latency > worstLatencyMs) {
          worstLatencyMs = latency
          worstTag = item.tag
        }
      }
    } else {
      failed += 1
    }
  }

  return {
    total: list.length,
    success,
    failed,
    avgLatencyMs: latencyCount ? Math.round((latencySum / latencyCount) * 10) / 10 : undefined,
    bestTag,
    bestLatencyMs,
    worstTag,
    worstLatencyMs,
  }
}
