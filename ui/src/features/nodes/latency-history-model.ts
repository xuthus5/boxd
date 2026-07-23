import { nodeTestTypes } from "@/features/nodes/node-test-inputs"
import type { LatencyPoint } from "@/lib/api/types"

export function latencyHistoryChartRows(points: readonly LatencyPoint[]) {
  return points.map((point, index) => ({
    index: index + 1,
    timestamp: point.timestamp,
    latency: point.success && typeof point.latency_ms === "number" ? point.latency_ms : null,
    success: point.success,
    error: point.error,
  }))
}

export function summarizeLatencyHistory(points: readonly LatencyPoint[]) {
  const ok = points.filter((point) => point.success && typeof point.latency_ms === "number")
  if (!ok.length) {
    return {
      count: points.length,
      success: 0,
      latest: undefined as number | undefined,
      avg: undefined as number | undefined,
      min: undefined as number | undefined,
      max: undefined as number | undefined,
    }
  }
  const values = ok.map((point) => point.latency_ms as number)
  const sum = values.reduce((acc, value) => acc + value, 0)
  return {
    count: points.length,
    success: values.length,
    latest: values.at(-1),
    avg: sum / values.length,
    min: Math.min(...values),
    max: Math.max(...values),
  }
}

export function latencyHistoryTypes(history?: Record<string, LatencyPoint[]>) {
  if (!history) return [] as string[]
  const preferred = nodeTestTypes.filter((type) => (history[type]?.length ?? 0) > 0)
  const extra = Object.keys(history).filter(
    (type) => !preferred.includes(type as (typeof nodeTestTypes)[number]) && (history[type]?.length ?? 0) > 0,
  )
  return [...preferred, ...extra]
}
