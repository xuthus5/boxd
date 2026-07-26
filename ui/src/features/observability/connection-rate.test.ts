import { describe, expect, it } from "vitest"

import {
  calculateConnectionRates,
  connectionRateTotal,
  formatConnectionRate,
  formatConnectionRatePair,
} from "@/features/observability/connection-rate"
import type { Connection } from "@/lib/api/types"

const previous: Connection[] = [
  { id: 1, target: "a.example:443", outbound: "proxy", upload: 100, download: 200, start: "start-1" },
  { id: 2, target: "b.example:443", outbound: "direct", upload: 50, download: 75, start: "start-2" },
]

describe("connection-rate", () => {
  it("calculates rates for matching stable connections", () => {
    const current = calculateConnectionRates([
      { ...previous[0], upload: 300, download: 600 },
      { ...previous[1], upload: 50, download: 75 },
    ], previous, 2000)

    expect(current[0]).toMatchObject({ uploadRate: 100, downloadRate: 200 })
    expect(connectionRateTotal(current[0])).toBe(300)
    expect(current[1]).toMatchObject({ uploadRate: 0, downloadRate: 0 })
  })

  it("does not estimate new, restarted, or reset connections", () => {
    const current = calculateConnectionRates([
      { id: 3, target: "new.example:443", outbound: "proxy", upload: 10, download: 20, start: "new" },
      { ...previous[0], start: "restarted", upload: 300, download: 600 },
      { ...previous[1], upload: 10, download: 20 },
    ], previous, 1000)

    expect(current[0]).not.toHaveProperty("uploadRate")
    expect(current[1]).not.toHaveProperty("downloadRate")
    expect(current[2]).not.toHaveProperty("uploadRate")
    expect(connectionRateTotal(current[0])).toBeUndefined()
  })

  it("rejects invalid elapsed intervals and invalid counters", () => {
    const current = { ...previous[0], upload: Number.NaN, download: 300 }
    expect(calculateConnectionRates([current], previous, 0)[0]).not.toHaveProperty("uploadRate")
    expect(calculateConnectionRates([current], previous, -1)[0]).not.toHaveProperty("downloadRate")
    expect(calculateConnectionRates([current], previous, 1000)[0]).not.toHaveProperty("uploadRate")
  })

  it("formats byte rates and handles missing values", () => {
    expect(formatConnectionRate(undefined)).toBe("—")
    expect(formatConnectionRate(0)).toBe("0 B/s")
    expect(formatConnectionRate(99.9)).toBe("100 B/s")
    expect(formatConnectionRate(1024)).toBe("1.00 KB/s")
    expect(formatConnectionRatePair(1024, 2048)).toBe("↑ 1.00 KB/s · ↓ 2.00 KB/s")
    expect(formatConnectionRatePair(undefined, 1)).toBe("—")
    expect(connectionRateTotal({ ...previous[0], uploadRate: undefined, downloadRate: 1 })).toBeUndefined()
  })
})
