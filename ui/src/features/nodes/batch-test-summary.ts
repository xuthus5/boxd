/** Summarize bulk node speed-test results for toast / UI feedback. */

import { formatNodeTestFailureSample } from "@/features/nodes/node-test-error"
import type { TestResult } from "@/lib/api/types"

export type BatchTestFailureSample = {
  tag: string
  testType: string
  error: string
}

export type BatchTestSummary = {
  total: number
  success: number
  failed: number
  avgLatencyMs?: number
  bestTag?: string
  bestLatencyMs?: number
  worstTag?: string
  worstLatencyMs?: number
  failedSamples: BatchTestFailureSample[]
}

const FAILED_SAMPLE_LIMIT = 3

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
  const failedSamples: BatchTestFailureSample[] = []

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
      continue
    }
    failed += 1
    if (failedSamples.length < FAILED_SAMPLE_LIMIT) {
      failedSamples.push({
        tag: item.tag?.trim() || "—",
        testType: (item.test_type ?? "").trim().toUpperCase() || "—",
        error: formatNodeTestFailureSample(item),
      })
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
    failedSamples,
  }
}

export function formatBatchTestToastMessage(
  summary: BatchTestSummary,
  t: (key: string, values?: Record<string, string | number>) => string,
): string {
  if (summary.total === 0) return t("nodes.batchComplete")
  const parts = [
    t("nodes.batchSummary", {
      success: summary.success,
      failed: summary.failed,
      total: summary.total,
      avg: summary.avgLatencyMs === undefined ? "—" : `${summary.avgLatencyMs}ms`,
    }),
  ]
  if (summary.bestTag && summary.bestLatencyMs !== undefined) {
    parts.push(t("nodes.batchBest", {
      tag: summary.bestTag,
      latency: `${summary.bestLatencyMs}ms`,
    }))
  }
  if (summary.worstTag && summary.worstLatencyMs !== undefined) {
    parts.push(t("nodes.batchWorst", {
      tag: summary.worstTag,
      latency: `${summary.worstLatencyMs}ms`,
    }))
  }
  if (summary.failedSamples.length) {
    const sample = summary.failedSamples
      .map((item) => `${item.tag}/${item.testType}: ${item.error}`)
      .join("; ")
    parts.push(t("nodes.batchFailedSamples", { samples: sample }))
  }
  return parts.join(" · ")
}

export function batchTestToastTone(summary: BatchTestSummary): "success" | "warning" | "error" {
  if (summary.failed > 0 && summary.success === 0) return "error"
  if (summary.failed > 0) return "warning"
  return "success"
}
