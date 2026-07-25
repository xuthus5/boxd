import { describe, expect, it } from "vitest"

import { subscriptionRefreshSchedule } from "@/features/subscriptions/subscription-schedule"

describe("subscriptionRefreshSchedule", () => {
  const now = Date.parse("2026-07-25T08:00:00Z")

  it("uses the latest success or failure as the next refresh anchor", () => {
    expect(subscriptionRefreshSchedule({
      interval_min: 60,
      last_updated: "2026-07-25T07:30:00Z",
      error_at: "2026-07-25T07:45:00Z",
    }, now)).toEqual({ intervalMinutes: 60, nextAt: Date.parse("2026-07-25T08:45:00Z"), due: false })
  })

  it("marks never-attempted and overdue subscriptions as due", () => {
    expect(subscriptionRefreshSchedule({ interval_min: 60 }, now)).toEqual({
      intervalMinutes: 60,
      nextAt: null,
      due: true,
    })
    expect(subscriptionRefreshSchedule({
      interval_min: 30,
      last_updated: "2026-07-25T07:00:00Z",
    }, now)).toEqual({ intervalMinutes: 30, nextAt: Date.parse("2026-07-25T07:30:00Z"), due: true })
  })

  it("keeps legacy fallback intervals explicit", () => {
    expect(subscriptionRefreshSchedule({ interval_min: 0 }, now)).toEqual({
      intervalMinutes: null,
      nextAt: null,
      due: false,
    })
  })
})
