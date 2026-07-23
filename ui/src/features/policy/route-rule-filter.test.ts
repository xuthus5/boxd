import { describe, expect, it } from "vitest"

import { matchesRouteRule } from "@/features/policy/route-rule-filter"

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
})
