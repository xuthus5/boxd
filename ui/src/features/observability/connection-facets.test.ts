import { describe, expect, it } from "vitest"

import {
  buildConnectionsHref,
  connectionTargetLogQuery,
  facetHref,
  connectionFacetValue,
  connectionFiltersActive,
  filterConnectionsByFacets,
  listConnectionFacets,
  matchesConnectionFacet,
  parseConnectionSearchParams,
} from "@/features/observability/connection-facets"
import type { Connection } from "@/lib/api/types"

const sample: Connection[] = [
  { id: 1, target: "a.com:443", outbound: "proxy", rule: "geosite-google", network: "tcp", protocol: "tls", process: "/usr/bin/curl", upload: 1, download: 2, start: "t" },
  { id: 2, target: "b.com:443", outbound: "direct", rule: "geoip-cn", network: "udp", protocol: "quic", process: "/usr/bin/curl", upload: 1, download: 2, start: "t" },
  { id: 3, target: "c.com:443", outbound: "proxy", network: "tcp", protocol: "", process: "/Applications/Chrome.app", upload: 1, download: 2, start: "t" },
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
    expect(listConnectionFacets(sample, "outbound")[0]).toEqual({ value: "proxy", count: 2 })
  })

  it("filters by query plus network/protocol/outbound/rule facets", () => {
    expect(matchesConnectionFacet(sample[0], "network", "tcp")).toBe(true)
    expect(matchesConnectionFacet(sample[0], "network", "udp")).toBe(false)
    expect(filterConnectionsByFacets(sample, { network: "tcp" })).toHaveLength(2)
    expect(filterConnectionsByFacets(sample, { protocol: "quic" })).toEqual([sample[1]])
    expect(filterConnectionsByFacets(sample, { outbound: "proxy" }).map((item) => item.id)).toEqual([1, 3])
    expect(filterConnectionsByFacets(sample, { rule: "geoip-cn" })).toEqual([sample[1]])
    expect(filterConnectionsByFacets(sample, { process: "/usr/bin/curl" }).map((item) => item.id)).toEqual([1, 2])
    expect(filterConnectionsByFacets(sample, { query: "proxy", network: "tcp" }).map((item) => item.id)).toEqual([1, 3])
    expect(connectionFiltersActive({ query: "x" })).toBe(true)
    expect(connectionFiltersActive({ network: "tcp" })).toBe(true)
    expect(connectionFiltersActive({ outbound: "proxy" })).toBe(true)
    expect(connectionFiltersActive({ process: "/usr/bin/curl" })).toBe(true)
    expect(connectionFiltersActive({})).toBe(false)
  })

  it("parses and builds connections deep-link query strings", () => {
    expect(parseConnectionSearchParams(new URLSearchParams("network=tcp&outbound=proxy&process=/usr/bin/curl&q=api"))).toEqual({
      query: "api",
      network: "tcp",
      protocol: undefined,
      outbound: "proxy",
      rule: undefined,
      process: "/usr/bin/curl",
    })
    expect(buildConnectionsHref({ network: "udp", rule: "geoip-cn", process: "/usr/bin/curl" })).toBe(
      "/observability/connections?network=udp&rule=geoip-cn&process=%2Fusr%2Fbin%2Fcurl",
    )
    expect(buildConnectionsHref({})).toBe("/observability/connections")
  })

  it("derives log query hosts from connection targets", () => {
    expect(connectionTargetLogQuery("example.com:443")).toBe("example.com")
    expect(connectionTargetLogQuery("[2001:db8::1]:443")).toBe("2001:db8::1")
    expect(connectionTargetLogQuery("plain-host")).toBe("plain-host")
    expect(connectionTargetLogQuery("")).toBe("")
  })

  it("builds clickable facet deep-links", () => {
    expect(facetHref("process", "/usr/bin/curl")).toBe("/observability/connections?process=%2Fusr%2Fbin%2Fcurl")
    expect(facetHref("network", "—")).toBe("")
  })
})

