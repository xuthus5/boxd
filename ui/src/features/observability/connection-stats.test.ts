import { describe, expect, it } from "vitest"

import { aggregateConnections, filterConnectionsByGroup, matchesConnection, summarizeConnections } from "@/features/observability/connection-stats"
import type { Connection } from "@/lib/api/types"

const sample: Connection[] = [
  { id: 1, target: "a.com:443", outbound: "proxy", rule: "geosite-google", process: "/usr/bin/curl", upload: 10, download: 20, start: new Date().toISOString() },
  { id: 2, target: "b.com:443", outbound: "proxy", rule: "geoip-cn", process: "/usr/bin/curl", upload: 5, download: 5, start: new Date().toISOString() },
  { id: 3, target: "c.com:443", outbound: "direct", rule: "geoip-cn", process: "/Applications/Chrome.app", upload: 1, download: 2, start: new Date().toISOString() },
  { id: 4, target: "d.com:443", outbound: "proxy", rule: "", upload: 100, download: 200, start: new Date().toISOString() },
]

describe("connection-stats", () => {
  it("summarizes totals", () => {
    expect(summarizeConnections(sample)).toEqual({ upload: 116, download: 227, outbounds: 2 })
  })

  it("aggregates by outbound traffic", () => {
    const groups = aggregateConnections(sample, "outbound")
    expect(groups[0]).toMatchObject({ key: "proxy", count: 3, upload: 115, download: 225 })
    expect(groups[1]).toMatchObject({ key: "direct", count: 1 })
  })

  it("aggregates by rule and falls back for empty", () => {
    const groups = aggregateConnections(sample, "rule")
    expect(groups.find((item) => item.key === "geoip-cn")?.count).toBe(2)
    expect(groups.find((item) => item.key === "—")?.count).toBe(1)
  })

  it("aggregates by process", () => {
    const groups = aggregateConnections(sample, "process")
    expect(groups.find((item) => item.key === "/usr/bin/curl")?.count).toBe(2)
    expect(groups.find((item) => item.key === "/Applications/Chrome.app")?.count).toBe(1)
    expect(groups.find((item) => item.key === "—")?.count).toBe(1)
  })

  it("matches search haystack", () => {
    expect(matchesConnection(sample[0], "google")).toBe(true)
    expect(matchesConnection(sample[0], "direct")).toBe(false)
  })

  it("filters connections by outbound or rule group", () => {
    expect(filterConnectionsByGroup(sample, "outbound", "proxy")).toHaveLength(3)
    expect(filterConnectionsByGroup(sample, "rule", "—")).toHaveLength(1)
    expect(filterConnectionsByGroup(sample, "process", "/usr/bin/curl")).toHaveLength(2)
  })

  it("matches source and process fields", () => {
    const item = {
      ...sample[0],
      source: "10.0.0.8:1",
      inbound: "mixed-in",
      network: "tcp",
      protocol: "http",
      process: "/usr/bin/curl",
    }
    expect(matchesConnection(item, "10.0.0.8")).toBe(true)
    expect(matchesConnection(item, "mixed-in")).toBe(true)
    expect(matchesConnection(item, "curl")).toBe(true)
    expect(matchesConnection(item, "udp")).toBe(false)
  })
})
