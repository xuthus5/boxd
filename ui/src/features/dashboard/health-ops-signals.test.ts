import { describe, expect, it } from "vitest"

import {
  buildHealthOpsSignals,
  countFailedSubscriptions,
  countProblemNodes,
  hasHealthOpsAlerts,
} from "@/features/dashboard/health-ops-signals"
import type { LatencyPoint, Outbound, Subscription } from "@/lib/api/types"

function node(tag: string): Outbound {
  return { tag, type: "vless", server: "example.com", port: 443, source: "import", raw: {} }
}

function history(
  tag: string,
  points: Array<{ success: boolean; latency_ms?: number }>,
): Record<string, Record<string, LatencyPoint[]>> {
  return {
    [tag]: {
      tcp: points.map((point, index) => ({
        timestamp: `2026-07-23T00:0${index}:00Z`,
        success: point.success,
        latency_ms: point.latency_ms ?? 20,
      })),
    },
  }
}

const subscriptions: Subscription[] = [
  {
    id: "ok",
    name: "ok",
    url: "https://example.com/ok",
    interval_min: 60,
    last_updated: "2026-01-01T00:00:00Z",
    outbounds: [],
  },
  {
    id: "bad",
    name: "bad",
    url: "https://example.com/bad",
    interval_min: 60,
    last_updated: "2026-01-01T00:00:00Z",
    outbounds: [],
    error: "timeout",
  },
]

describe("health-ops-signals", () => {
  it("counts failed subscriptions", () => {
    expect(countFailedSubscriptions(undefined)).toBe(0)
    expect(countFailedSubscriptions([])).toBe(0)
    expect(countFailedSubscriptions(subscriptions)).toBe(1)
  })

  it("counts unstable and failed nodes from latency history", () => {
    const nodes = [node("stable"), node("unstable"), node("failed"), node("unknown")]
    const map = {
      ...history("stable", [{ success: true }, { success: true }, { success: true }, { success: true }]),
      ...history("unstable", [{ success: true }, { success: false }, { success: false }, { success: false }]),
      ...history("failed", [{ success: false }, { success: false }]),
    }
    expect(countProblemNodes(nodes, map)).toEqual({
      unstableNodes: 1,
      failedNodes: 1,
      problemNodes: 2,
    })
  })

  it("builds combined ops signals and alert gate", () => {
    const empty = buildHealthOpsSignals({})
    expect(empty).toEqual({
      failedSubscriptions: 0,
      unstableNodes: 0,
      failedNodes: 0,
      problemNodes: 0,
    })
    expect(hasHealthOpsAlerts(empty)).toBe(false)

    const signals = buildHealthOpsSignals({
      subscriptions,
      nodes: [node("unstable"), node("failed")],
      history: {
        ...history("unstable", [{ success: true }, { success: false }, { success: false }]),
        ...history("failed", [{ success: false }]),
      },
    })
    expect(signals.failedSubscriptions).toBe(1)
    expect(signals.unstableNodes).toBe(1)
    expect(signals.failedNodes).toBe(1)
    expect(signals.problemNodes).toBe(2)
    expect(hasHealthOpsAlerts(signals)).toBe(true)
  })
})
