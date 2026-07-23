import { describe, expect, it } from "vitest"

import {
  buildSubscriptionsHref,
  failedSubscriptionIds,
  filterSubscriptions,
  matchesSubscription,
  parseSubscriptionSearchParams,
  sortSubscriptions,
  subscriptionFiltersActive,
  summarizeSubscriptionStatus,
  toSubscriptionSearchParams,
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

  it("parses and builds subscription deep-link query strings", () => {
    expect(parseSubscriptionSearchParams(new URLSearchParams("q=主&status=error"))).toEqual({
      query: "主",
      status: "error",
    })
    expect(parseSubscriptionSearchParams(new URLSearchParams("status=nope"))).toEqual({
      query: undefined,
      status: undefined,
    })
    expect(buildSubscriptionsHref({ status: "error" })).toBe("/subscriptions?status=error")
    expect(buildSubscriptionsHref({ status: "all", query: "  " })).toBe("/subscriptions")
    expect(toSubscriptionSearchParams({ query: "hk", status: "ok" }).toString()).toBe("q=hk&status=ok")
    expect(subscriptionFiltersActive({ status: "error" })).toBe(true)
    expect(subscriptionFiltersActive({})).toBe(false)
  })

  it("summarizes status buckets for the current search query", () => {
    expect(summarizeSubscriptionStatus(sample)).toEqual({ total: 3, ok: 1, error: 2 })
    expect(summarizeSubscriptionStatus(sample, "备用")).toEqual({ total: 1, ok: 1, error: 0 })
  })
})
