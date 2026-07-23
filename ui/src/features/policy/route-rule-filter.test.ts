import { describe, expect, it } from "vitest"

import {
  buildRouteHref,
  filterRouteRules,
  matchesRouteAction,
  matchesRouteRule,
  parseRouteSearchParams,
  routeFiltersActive,
  routeRuleAction,
  summarizeRouteActions,
  toRouteSearchParams,
} from "@/features/policy/route-rule-filter"

describe("matchesRouteRule", () => {
  const item = { action: "route", outbound: "proxy", domain: ["google.com"], type: "logical" }
  const metadata = { name: "Google", description: "search traffic" }

  it("matches metadata and summary fields", () => {
    expect(matchesRouteRule(item, metadata, "google")).toBe(true)
    expect(matchesRouteRule(item, metadata, "proxy")).toBe(true)
    expect(matchesRouteRule(item, metadata, "search")).toBe(true)
    expect(matchesRouteRule(item, metadata, "direct")).toBe(false)
  })

  it("returns true for empty query", () => {
    expect(matchesRouteRule(item, metadata, "")).toBe(true)
  })

  it("filters by action facet and summarizes buckets", () => {
    const rules = [
      { action: "route", outbound: "proxy", domain: ["google.com"] },
      { action: "reject", domain: ["ads.example"] },
      { outbound: "direct", domain: ["cn.example"] },
    ]
    const meta = [
      { name: "Google", description: "search" },
      { name: "Ads", description: "block" },
      { name: "CN", description: "local" },
    ]
    expect(routeRuleAction(rules[2])).toBe("route")
    expect(matchesRouteAction(rules[1], "reject")).toBe(true)
    expect(filterRouteRules(rules, { action: "reject" }, meta).map((rule) => rule.action)).toEqual(["reject"])
    expect(filterRouteRules(rules, { query: "google", action: "route" }, meta)).toHaveLength(1)
    expect(summarizeRouteActions(rules, "", meta)).toEqual({
      total: 3,
      buckets: [
        { action: "route", count: 2 },
        { action: "reject", count: 1 },
      ],
    })
    expect(summarizeRouteActions(rules, "ads", meta).buckets.map((bucket) => bucket.action)).toEqual(["reject"])
    expect(routeFiltersActive({ action: "reject" })).toBe(true)
    expect(routeFiltersActive({})).toBe(false)
  })

  it("parses and builds route deep-link query strings", () => {
    expect(parseRouteSearchParams(new URLSearchParams("q=google&action=reject"))).toEqual({
      query: "google",
      action: "reject",
    })
    expect(parseRouteSearchParams(new URLSearchParams(""))).toEqual({ query: undefined, action: undefined })
    expect(buildRouteHref({ query: "google", action: "route" })).toBe("/policy/route?q=google&action=route")
    expect(buildRouteHref({})).toBe("/policy/route")
    expect(toRouteSearchParams({ query: "  " }).toString()).toBe("")
  })
})
