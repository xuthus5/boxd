import { describe, expect, it } from "vitest"

import { buildHealthSummary, countNetworks, healthTone } from "@/features/dashboard/health-summary"
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

  it("handles empty snapshots and offline kernels", () => {
    expect(buildHealthSummary(undefined, { running: false })).toMatchObject({
      tone: "offline",
      active: 0,
      topOutbound: "—",
      topRule: "—",
    })
    expect(buildHealthSummary({ active_connections: 0, list: [] }, { running: true }).tone).toBe("idle")
  })
})
