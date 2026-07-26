import { describe, expect, it } from "vitest"

import { buildHealthSummary, countNetworks, healthTone } from "@/features/dashboard/health-summary"
import type { ConnectionWithRates } from "@/features/observability/connection-rate"
import type { Connection } from "@/lib/api/types"

const sample: Connection[] = [
  { id: 1, target: "a.com:443", outbound: "proxy", rule: "geosite-google", network: "tcp", upload: 10, download: 20, start: "2026-07-23T00:00:00Z" },
  { id: 2, target: "b.com:443", outbound: "proxy", rule: "geoip-cn", network: "udp", upload: 5, download: 5, start: "2026-07-23T00:00:01Z" },
  { id: 3, target: "c.com:443", outbound: "direct", rule: "geoip-cn", network: "", upload: 1, download: 2, start: "2026-07-23T00:00:02Z" },
]

describe("health-summary", () => {
  it("counts networks and maps health tones", () => {
    expect(countNetworks(sample)).toEqual({ tcp: 1, udp: 1, otherNetwork: 1 })
    expect(healthTone(false, 3)).toBe("offline")
    expect(healthTone(true, 0)).toBe("idle")
    expect(healthTone(true, 12)).toBe("ok")
    expect(healthTone(true, 50)).toBe("warn")
  })

  it("builds a summary from the live connection snapshot", () => {
    const summary = buildHealthSummary({ active_connections: 3, list: sample }, { running: true })
    expect(summary.active).toBe(3)
    expect(summary.outbounds).toBe(2)
    expect(summary.topOutbound).toBe("proxy")
    expect(summary.topRule).toBe("geosite-google")
    expect(summary.upload).toBe(16)
    expect(summary.download).toBe(27)
    expect(summary.tone).toBe("ok")
  })

  it("summarizes live rates and selects the busiest outbound", () => {
    const rated: ConnectionWithRates[] = [
      { ...sample[0], uploadRate: 30, downloadRate: 70 },
      { ...sample[1], uploadRate: 10, downloadRate: 20 },
      { ...sample[2], uploadRate: 5, downloadRate: 10 },
    ]

    expect(buildHealthSummary({ active_connections: 3, list: rated }, { running: true })).toMatchObject({
      uploadRate: 45,
      downloadRate: 100,
      rateReady: true,
      topRateOutbound: "proxy",
    })
    expect(buildHealthSummary({ active_connections: 3, list: sample }, { running: true })).toMatchObject({
      rateReady: false,
      topRateOutbound: "—",
    })
    expect(buildHealthSummary({
      active_connections: 3,
      list: [rated[0], sample[1], rated[2]],
    }, { running: true })).toMatchObject({
      rateReady: false,
      topRateOutbound: "direct",
    })
  })

  it("handles empty snapshots and offline kernels", () => {
    expect(buildHealthSummary(undefined, { running: false })).toMatchObject({
      tone: "offline",
      active: 0,
      rateReady: false,
      topOutbound: "—",
      topRateOutbound: "—",
      topRule: "—",
    })
    expect(buildHealthSummary({ active_connections: 0, list: [] }, { running: true })).toMatchObject({
      tone: "idle",
      uploadRate: 0,
      downloadRate: 0,
      rateReady: true,
    })
  })
})
