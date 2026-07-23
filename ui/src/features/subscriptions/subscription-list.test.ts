import { describe, expect, it } from "vitest"

import {
  failedSubscriptionIds,
  filterSubscriptions,
  matchesSubscription,
  sortSubscriptions,
} from "@/features/subscriptions/subscription-list"
import type { Subscription } from "@/lib/api/types"

const sample: Subscription[] = [
  { id: "a", name: "主订阅", url: "https://a.example/sub", interval_min: 60, last_updated: "2026-01-01T00:00:00Z", error: "timeout" },
  { id: "b", name: "备用", url: "https://b.example/sub", interval_min: 60, last_updated: "2026-06-01T00:00:00Z" },
  { id: "c", name: "测试", url: "https://c.example/sub", interval_min: 60, last_updated: "2026-03-01T00:00:00Z", error: "403" },
]

describe("subscription-list", () => {
  it("sorts errors first then by last_updated desc", () => {
    const sorted = sortSubscriptions(sample)
    expect(sorted.map((item) => item.id)).toEqual(["c", "a", "b"])
  })

  it("filters by status and query", () => {
    expect(filterSubscriptions(sample, { status: "error" })).toHaveLength(2)
    expect(filterSubscriptions(sample, { status: "ok" })).toHaveLength(1)
    expect(filterSubscriptions(sample, { query: "备用" })[0]?.id).toBe("b")
    expect(matchesSubscription(sample[0], "timeout")).toBe(true)
  })

  it("lists failed ids", () => {
    expect(failedSubscriptionIds(sample)).toEqual(["a", "c"])
  })
})
