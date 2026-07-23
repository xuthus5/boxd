import { describe, expect, it } from "vitest"

import {
  connectionFacetValue,
  connectionFiltersActive,
  filterConnectionsByFacets,
  listConnectionFacets,
  matchesConnectionFacet,
} from "@/features/observability/connection-facets"
import type { Connection } from "@/lib/api/types"

const sample: Connection[] = [
  { id: 1, target: "a.com:443", outbound: "proxy", network: "tcp", protocol: "tls", upload: 1, download: 2, start: "t" },
  { id: 2, target: "b.com:443", outbound: "direct", network: "udp", protocol: "quic", upload: 1, download: 2, start: "t" },
  { id: 3, target: "c.com:443", outbound: "proxy", network: "tcp", protocol: "", upload: 1, download: 2, start: "t" },
  { id: 4, target: "d.com:53", outbound: "dns", network: "", protocol: "dns", upload: 1, download: 2, start: "t" },
]

describe("connection-facets", () => {
  it("lists facet options with counts and unknown fallback", () => {
    expect(connectionFacetValue(sample[2], "protocol")).toBe("—")
    expect(listConnectionFacets(sample, "network")).toEqual([
      { value: "tcp", count: 2 },
      { value: "udp", count: 1 },
      { value: "—", count: 1 },
    ])
    expect(listConnectionFacets(sample, "protocol")).toEqual([
      { value: "dns", count: 1 },
      { value: "quic", count: 1 },
      { value: "tls", count: 1 },
      { value: "—", count: 1 },
    ])
  })

  it("filters by query plus network/protocol facets", () => {
    expect(matchesConnectionFacet(sample[0], "network", "tcp")).toBe(true)
    expect(matchesConnectionFacet(sample[0], "network", "udp")).toBe(false)
    expect(filterConnectionsByFacets(sample, { network: "tcp" })).toHaveLength(2)
    expect(filterConnectionsByFacets(sample, { protocol: "quic" })).toEqual([sample[1]])
    expect(filterConnectionsByFacets(sample, { query: "proxy", network: "tcp" }).map((item) => item.id)).toEqual([1, 3])
    expect(connectionFiltersActive({ query: "x" })).toBe(true)
    expect(connectionFiltersActive({ network: "tcp" })).toBe(true)
    expect(connectionFiltersActive({})).toBe(false)
  })
})
