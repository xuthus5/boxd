import { describe, expect, it } from "vitest"

import {
  buildRouteHref,
  matchesRouteRule,
  parseRouteSearchParams,
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

  it("parses and builds route deep-link query strings", () => {
    expect(parseRouteSearchParams(new URLSearchParams("q=google"))).toEqual({ query: "google" })
    expect(buildRouteHref({ query: "google" })).toBe("/policy/route?q=google")
    expect(buildRouteHref({})).toBe("/policy/route")
    expect(toRouteSearchParams({ query: "  " }).toString()).toBe("")
  })
})
