import { describe, expect, it } from "vitest"

import {
  formatTrafficTimestamp,
  getTrafficPointValue,
  getTrafficTimeDomain,
  getTrafficTimestampValue,
  getTrafficValueDomain,
} from "@/features/dashboard/traffic-chart-model"

describe("traffic chart model", () => {
  it("normalizes timestamps and the sliding time window", () => {
    const timestamp = "2026-01-01T00:00:30Z"
    const value = Date.parse(timestamp)

    expect(getTrafficTimestampValue({ timestamp })).toBe(value)
    expect(getTrafficTimestampValue({ timestamp: "invalid" })).toBe(0)
    expect(getTrafficTimestampValue({ timestamp: 1 as unknown as string })).toBe(0)
    expect(formatTrafficTimestamp(value)).not.toBe("")
    expect(formatTrafficTimestamp(timestamp)).not.toBe("")
    expect(formatTrafficTimestamp("invalid")).toBe("")
    expect(getTrafficTimeDomain([])).toEqual([0, 60_000])
    expect(getTrafficTimeDomain([{ timestamp: "invalid" }, { timestamp }])).toEqual([value - 60_000, value])
  })

  it("accepts only finite numeric values", () => {
    const point = { timestamp: "2026-01-01T00:00:00Z", rate: 42 }

    expect(getTrafficPointValue(point, "rate")).toBe(42)
    expect(getTrafficPointValue({ ...point, rate: "42" }, "rate")).toBeUndefined()
    expect(getTrafficPointValue({ ...point, rate: Number.POSITIVE_INFINITY }, "rate")).toBeUndefined()
    expect(getTrafficPointValue(point, "missing")).toBeUndefined()
  })

  it("builds a safe power-of-two value domain", () => {
    const timestamp = "2026-01-01T00:00:00Z"

    expect(getTrafficValueDomain([], ["rate"])).toEqual([0, 1])
    expect(getTrafficValueDomain([{ timestamp, rate: 10 }], ["rate"])).toEqual([0, 16])
    expect(getTrafficValueDomain([{ timestamp, rate: Number.MAX_VALUE }], ["rate"])).toEqual([0, Number.MAX_VALUE])
    expect(getTrafficValueDomain([{ timestamp, rate: "invalid" }], ["rate"])).toEqual([0, 1])
  })
})
