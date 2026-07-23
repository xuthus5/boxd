/** Latency history success-rate helpers for node cards. */

import { summarizeLatencyHistory } from "@/features/nodes/latency-history-model"
import type { LatencyPoint } from "@/lib/api/types"

export type LatencyHealthTone = "unknown" | "excellent" | "good" | "fair" | "poor" | "failed"

export type LatencyHealth = {
  count: number
  success: number
  failed: number
  rate: number | undefined
  percent: number
  tone: LatencyHealthTone
  latest?: number
  avg?: number
}

export function latencySuccessRate(points: readonly LatencyPoint[]): number | undefined {
  if (!points.length) return undefined
  const success = points.filter((point) => point.success).length
  return success / points.length
}

export function latencyHealthTone(rate: number | undefined, count: number): LatencyHealthTone {
  if (!count || rate === undefined) return "unknown"
  if (rate >= 0.95) return "excellent"
  if (rate >= 0.8) return "good"
  if (rate >= 0.6) return "fair"
  if (rate > 0) return "poor"
  return "failed"
}

export function latencyHealthBarClass(tone: LatencyHealthTone): string {
  switch (tone) {
    case "excellent":
      return "bg-emerald-600 dark:bg-emerald-500"
    case "good":
      return "bg-sky-600 dark:bg-sky-500"
    case "fair":
      return "bg-amber-500 dark:bg-amber-400"
    case "poor":
      return "bg-orange-600 dark:bg-orange-500"
    case "failed":
      return "bg-destructive"
    default:
      return "bg-muted-foreground/40"
  }
}

export function buildLatencyHealth(points: readonly LatencyPoint[]): LatencyHealth {
  const summary = summarizeLatencyHistory(points)
  const success = points.filter((point) => point.success).length
  const failed = Math.max(0, points.length - success)
  const rate = latencySuccessRate(points)
  const percent = rate === undefined ? 0 : Math.round(rate * 100)
  return {
    count: points.length,
    success,
    failed,
    rate,
    percent,
    tone: latencyHealthTone(rate, points.length),
    latest: summary.latest,
    avg: summary.avg,
  }
}
