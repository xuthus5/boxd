import { describe, expect, it } from "vitest"

import {
  buildLatencyHealth,
  latencyHealthBarClass,
  latencyHealthTone,
  latencySuccessRate,
} from "@/features/nodes/latency-health"
import type { LatencyPoint } from "@/lib/api/types"

const sample: LatencyPoint[] = [
  { timestamp: "1", success: true, latency_ms: 20 },
  { timestamp: "2", success: false, error: "timeout" },
  { timestamp: "3", success: true, latency_ms: 40 },
  { timestamp: "4", success: true, latency_ms: 30 },
  { timestamp: "5", success: true, latency_ms: 25 },
]

describe("latency-health", () => {
  it("computes success rate and tone", () => {
    expect(latencySuccessRate([])).toBeUndefined()
    expect(latencySuccessRate(sample)).toBeCloseTo(0.8)
    expect(latencyHealthTone(undefined, 0)).toBe("unknown")
    expect(latencyHealthTone(1, 4)).toBe("excellent")
    expect(latencyHealthTone(0.9, 4)).toBe("good")
    expect(latencyHealthTone(0.7, 4)).toBe("fair")
    expect(latencyHealthTone(0.2, 4)).toBe("poor")
    expect(latencyHealthTone(0, 4)).toBe("failed")
  })

  it("builds health summary for node cards", () => {
    const health = buildLatencyHealth(sample)
    expect(health).toMatchObject({ count: 5, success: 4, failed: 1, percent: 80, tone: "good" })
    expect(health.latest).toBe(25)
    expect(health.avg).toBe(28.75)
    expect(latencyHealthBarClass("good")).toContain("sky")
    expect(buildLatencyHealth([]).tone).toBe("unknown")
  })
})
