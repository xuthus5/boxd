import { describe, expect, it } from "vitest"

import { formatRelativeTime } from "@/features/subscriptions/relative-time"

describe("formatRelativeTime", () => {
  const now = Date.parse("2026-07-23T08:00:00Z")

  it("formats recent times", () => {
    expect(formatRelativeTime("2026-07-23T07:59:30Z", now, "en")).toMatch(/second|now|ago/i)
    expect(formatRelativeTime("2026-07-23T07:00:00Z", now, "en")).toMatch(/hour/i)
    expect(formatRelativeTime("2026-07-20T08:00:00Z", now, "en")).toMatch(/day/i)
  })

  it("returns empty for invalid input", () => {
    expect(formatRelativeTime("not-a-date", now)).toBe("")
    expect(formatRelativeTime("", now)).toBe("")
  })
})
