import { describe, expect, it } from "vitest"

import {
  buildConnectionsHref,
  connectionTargetLogQuery,
  facetHref,
  connectionFacetValue,
  connectionFiltersActive,
  filterConnectionsByFacets,
  listConnectionFacets,
  listScopedConnectionFacets,
  logConnectionQuery,
  logConnectionsHref,
  logDNSHref,
  logDNSServerTag,
  matchesConnectionFacet,
  parseConnectionSearchParams,
  summarizeConnectionFacets,
  toConnectionSearchParams,
} from "@/features/observability/connection-facets"
import type { Connection } from "@/lib/api/types"

const sample: Connection[] = [
  { id: 1, target: "a.com:443", outbound: "proxy", rule: "geosite-google", network: "tcp", protocol: "tls", inbound: "mixed-in", process: "/usr/bin/curl", upload: 1, download: 2, start: "t" },
  { id: 2, target: "b.com:443", outbound: "direct", rule: "geoip-cn", network: "udp", protocol: "quic", inbound: "mixed-in", process: "/usr/bin/curl", upload: 1, download: 2, start: "t" },
  { id: 3, target: "c.com:443", outbound: "proxy", network: "tcp", protocol: "", inbound: "tun-in", process: "/Applications/Chrome.app", upload: 1, download: 2, start: "t" },
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

  it("scopes facet counts by other active filters and summarizes top values", () => {
    expect(listScopedConnectionFacets(sample, { network: "tcp" }, "outbound")).toEqual([
      { value: "proxy", count: 2 },
    ])
    expect(listScopedConnectionFacets(sample, { network: "tcp" }, "network")).toEqual([
      { value: "tcp", count: 2 },
      { value: "udp", count: 1 },
      { value: "—", count: 1 },
    ])
    const summary = summarizeConnectionFacets(sample, { outbound: "proxy" }, 2)
    expect(summary.find((section) => section.field === "network")?.options.map((item) => item.value)).toEqual(["tcp"])
    expect(summary.find((section) => section.field === "outbound")?.options.map((item) => item.value)).toEqual(["proxy", "direct"])
    expect(summary.every((section) => section.options.every((item) => item.value !== "—"))).toBe(true)
  })

  it("filters by query plus network/protocol/outbound/rule facets", () => {
    expect(matchesConnectionFacet(sample[0], "network", "tcp")).toBe(true)
    expect(matchesConnectionFacet(sample[0], "network", "udp")).toBe(false)
    expect(filterConnectionsByFacets(sample, { network: "tcp" })).toHaveLength(2)
    expect(filterConnectionsByFacets(sample, { protocol: "quic" })).toEqual([sample[1]])
    expect(filterConnectionsByFacets(sample, { outbound: "proxy" }).map((item) => item.id)).toEqual([1, 3])
    expect(filterConnectionsByFacets(sample, { rule: "geoip-cn" })).toEqual([sample[1]])
    expect(filterConnectionsByFacets(sample, { process: "/usr/bin/curl" }).map((item) => item.id)).toEqual([1, 2])
    expect(filterConnectionsByFacets(sample, { inbound: "mixed-in" }).map((item) => item.id)).toEqual([1, 2])
    expect(connectionFiltersActive({ inbound: "mixed-in" })).toBe(true)
    expect(filterConnectionsByFacets(sample, { query: "proxy", network: "tcp" }).map((item) => item.id)).toEqual([1, 3])
    expect(connectionFiltersActive({ query: "x" })).toBe(true)
    expect(connectionFiltersActive({ network: "tcp" })).toBe(true)
    expect(connectionFiltersActive({ outbound: "proxy" })).toBe(true)
    expect(connectionFiltersActive({ process: "/usr/bin/curl" })).toBe(true)
    expect(connectionFiltersActive({})).toBe(false)
  })

  it("parses and builds connections deep-link query strings", () => {
    expect(parseConnectionSearchParams(new URLSearchParams("network=tcp&inbound=mixed-in&outbound=proxy&process=/usr/bin/curl&q=api"))).toEqual({
      query: "api",
      network: "tcp",
      protocol: undefined,
      inbound: "mixed-in",
      outbound: "proxy",
      rule: undefined,
      process: "/usr/bin/curl",
      view: undefined,
      sort: undefined,
    })
    expect(buildConnectionsHref({ network: "udp", inbound: "mixed-in", rule: "geoip-cn", process: "/usr/bin/curl" })).toBe(
      "/observability/connections?network=udp&inbound=mixed-in&rule=geoip-cn&process=%2Fusr%2Fbin%2Fcurl",
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
  it("parses connection view tabs from the URL", () => {
    expect(parseConnectionSearchParams(new URLSearchParams("view=process&network=tcp"))).toEqual({
      query: undefined,
      network: "tcp",
      protocol: undefined,
      inbound: undefined,
      outbound: undefined,
      rule: undefined,
      process: undefined,
      view: "process",
      sort: undefined,
    })
    expect(parseConnectionSearchParams(new URLSearchParams("view=nope")).view).toBeUndefined()
    expect(toConnectionSearchParams({ view: "list", query: "api" }).toString()).toBe("q=api")
    expect(buildConnectionsHref({ view: "outbound", query: "api.telegram.org" })).toBe(
      "/observability/connections?q=api.telegram.org&view=outbound",
    )
  })

  it("extracts connection hosts from sing-box style log messages", () => {
    expect(logConnectionQuery(
      "[1690779573 0ms] inbound/mixed[mixed-in]: inbound connection to api.telegram.org:443",
    )).toBe("api.telegram.org")
    expect(logConnectionQuery(
      "outbound/vless[hk]: outbound connection to www.gstatic.com:443",
    )).toBe("www.gstatic.com")
    expect(logConnectionQuery(
      "inbound/mixed[mixed-in]: inbound connection from 127.0.0.1:45264",
    )).toBe("")
    expect(logConnectionQuery("dns query api.cloudflare.com")).toBe("api.cloudflare.com")
    expect(logConnectionQuery("route to [2001:db8::1]:443")).toBe("2001:db8::1")
    expect(logConnectionQuery("ready v1.2.3")).toBe("")
    expect(logConnectionsHref(
      "outbound connection to example.com:443",
    )).toBe("/observability/connections?q=example.com")
    expect(logConnectionsHref("kernel ready")).toBe("")
    expect(logDNSHref("dns query api.cloudflare.com")).toBe("/policy/dns?rq=api.cloudflare.com")
    expect(logDNSHref("dns reject rule matched")).toBe("/policy/dns?raction=reject")
    expect(logDNSHref("kernel ready")).toBe("")
  })

  it("parses connection sort from the URL", () => {
    expect(parseConnectionSearchParams(new URLSearchParams("sort=duration")).sort).toBe("duration")
    expect(parseConnectionSearchParams(new URLSearchParams("sort=nope")).sort).toBeUndefined()
    expect(toConnectionSearchParams({ sort: "traffic", query: "api" }).toString()).toBe("q=api")
    expect(buildConnectionsHref({ sort: "outbound", network: "tcp" })).toBe(
      "/observability/connections?network=tcp&sort=outbound",
    )
  })

})


  it("prefers DNS server tags when building DNS deep-links from logs", () => {
    expect(logDNSServerTag("dns/local[local]: exchange example.com")).toBe("local")
    expect(logDNSHref("dns/cf[cloudflare]: lookup api.telegram.org")).toBe("/policy/dns?sq=cloudflare")
    expect(logDNSHref("lookup example.com via nameserver")).toContain("/policy/dns")
  })
