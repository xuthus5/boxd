import { describe, expect, it } from "vitest"

import {
  buildProxyHref,
  filterProxyItems,
  matchesProxyItem,
  matchesProxyType,
  parseProxySearchParams,
  proxyFiltersActive,
  proxyItemType,
  summarizeProxyTypes,
  toProxySearchParams,
} from "@/features/proxy/proxy-filter"

const sample = [
  { tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 },
  { tag: "tun-in", type: "tun", interface_name: "tun0" },
  { tag: "hk", type: "vless", server: "a.example.com", server_port: 443, transport: { type: "ws" } },
  { tag: "direct", type: "direct" },
  { tag: "proxy", type: "selector", outbounds: ["hk", "direct"] },
]

describe("matchesProxyItem", () => {
  it("matches tag type listen server and transport", () => {
    expect(matchesProxyItem(sample[0], "mixed")).toBe(true)
    expect(matchesProxyItem(sample[0], "1080")).toBe(true)
    expect(matchesProxyItem(sample[2], "ws")).toBe(true)
    expect(matchesProxyItem(sample[2], "a.example")).toBe(true)
    expect(matchesProxyItem(sample[2], "trojan")).toBe(false)
  })

  it("returns true for empty query", () => {
    expect(matchesProxyItem({ tag: "x", type: "direct" }, "")).toBe(true)
  })

  it("filters by type facet and summarizes buckets", () => {
    expect(proxyItemType({ tag: "x" })).toBe("unknown")
    expect(matchesProxyType(sample[2], "vless")).toBe(true)
    expect(filterProxyItems(sample, { type: "mixed" }).map((item) => item.tag)).toEqual(["mixed-in"])
    expect(filterProxyItems(sample, { query: "example", type: "vless" }).map((item) => item.tag)).toEqual(["hk"])
    expect(summarizeProxyTypes(sample)).toEqual({
      total: 5,
      buckets: [
        { type: "direct", count: 1 },
        { type: "mixed", count: 1 },
        { type: "selector", count: 1 },
        { type: "tun", count: 1 },
        { type: "vless", count: 1 },
      ],
    })
    expect(summarizeProxyTypes(sample, "in").buckets.map((bucket) => bucket.type)).toEqual(["mixed", "tun"])
    expect(proxyFiltersActive({ type: "mixed" })).toBe(true)
    expect(proxyFiltersActive({})).toBe(false)
  })

  it("parses and builds proxy list deep-link query strings", () => {
    expect(parseProxySearchParams(new URLSearchParams("q=mixed&type=tun"))).toEqual({
      query: "mixed",
      type: "tun",
    })
    expect(parseProxySearchParams(new URLSearchParams(""))).toEqual({ query: undefined, type: undefined })
    expect(buildProxyHref("inbounds", { query: "tun", type: "tun" })).toBe("/proxy/inbounds?q=tun&type=tun")
    expect(buildProxyHref("outbounds", {})).toBe("/proxy/outbounds")
    expect(toProxySearchParams({ query: "  " }).toString()).toBe("")
  })
})
