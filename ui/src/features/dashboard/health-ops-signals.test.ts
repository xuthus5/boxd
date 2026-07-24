import { describe, expect, it } from "vitest"

import {
  buildHealthOpsSignals,
  countFailedSubscriptions,
  countProblemNodes,
  hasHealthOpsAlerts,
  isConfigApplyFailure,
  summarizeApplyFailures,
} from "@/features/dashboard/health-ops-signals"
import type { ConfigApplyEvent, LatencyPoint, Outbound, Subscription } from "@/lib/api/types"

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
      failedSubscriptionItems: [],
      unstableNodes: 0,
      failedNodes: 0,
      problemNodes: 0,
      problemNodeItems: [],
      applyFailures: 0,
      latestApplyFailure: undefined,
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
    expect(signals.failedSubscriptionItems.map((item) => item.id)).toEqual(["bad"])
    expect(signals.unstableNodes).toBe(1)
    expect(signals.failedNodes).toBe(1)
    expect(signals.problemNodes).toBe(2)
    expect(signals.problemNodeItems.map((item) => item.tag).sort()).toEqual(["failed", "unstable"])
    expect(hasHealthOpsAlerts(signals)).toBe(true)
  })
})

describe("apply failure signals", () => {
  it("detects rolled_back and validate_failed statuses", () => {
    expect(isConfigApplyFailure({ status: "rolled_back" } as ConfigApplyEvent)).toBe(true)
    expect(isConfigApplyFailure({ status: "validate_failed" } as ConfigApplyEvent)).toBe(true)
    expect(isConfigApplyFailure({ status: "applied" } as ConfigApplyEvent)).toBe(false)
    expect(isConfigApplyFailure({ status: "validated" } as ConfigApplyEvent)).toBe(false)
  })

  it("counts consecutive newest apply failures only", () => {
    const events: ConfigApplyEvent[] = [
      {
        id: "1", source: "validate_raw", status: "validate_failed", hash: "a", size: 1,
        error: "bad", applied_at: "2026-07-24T12:00:00Z",
      },
      {
        id: "2", source: "raw", status: "rolled_back", hash: "b", size: 2,
        error: "restart failed", applied_at: "2026-07-24T11:00:00Z",
      },
      {
        id: "3", source: "update", status: "applied", hash: "c", size: 3,
        applied_at: "2026-07-24T10:00:00Z",
      },
    ]
    const summary = summarizeApplyFailures(events)
    expect(summary.applyFailures).toBe(2)
    expect(summary.latestApplyFailure?.id).toBe("1")
    expect(summarizeApplyFailures([{ id: "x", source: "update", status: "applied", hash: "h", size: 1, applied_at: "t" }])).toEqual({
      applyFailures: 0,
    })
  })

  it("includes apply failures in combined ops signals", () => {
    const signals = buildHealthOpsSignals({
      applyEvents: [
        {
          id: "1", source: "validate_inbounds", status: "validate_failed", hash: "a", size: 1,
          error: "listen_port invalid", applied_at: "2026-07-24T12:00:00Z",
        },
      ],
    })
    expect(signals.applyFailures).toBe(1)
    expect(signals.latestApplyFailure?.source).toBe("validate_inbounds")
    expect(hasHealthOpsAlerts(signals)).toBe(true)
  })
})
