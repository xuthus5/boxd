import { describe, expect, it } from "vitest"

import {
  buildDNSHref,
  dnsRuleAction,
  dnsRuleFiltersActive,
  dnsServerFiltersActive,
  dnsServerType,
  filterDNSRules,
  filterDNSServers,
  matchesDNSRule,
  matchesDNSRuleAction,
  matchesDNSServer,
  matchesDNSServerType,
  parseDNSSearchParams,
  summarizeDNSRuleActions,
  summarizeDNSServerTypes,
  toDNSSearchParams,
} from "@/features/policy/dns-filter"

describe("dns-filter", () => {
  it("matches servers by tag type and address", () => {
    const server = { tag: "dns-remote", type: "https", server: "1.1.1.1", server_port: 443 }
    expect(matchesDNSServer(server, "remote")).toBe(true)
    expect(matchesDNSServer(server, "https")).toBe(true)
    expect(matchesDNSServer(server, "1.1.1.1")).toBe(true)
    expect(matchesDNSServer(server, "local")).toBe(false)
    expect(matchesDNSServer(server, "")).toBe(true)
  })

  it("matches rules by action server and domain", () => {
    const rule = { action: "route", server: "dns-remote", domain: ["google.com"], type: "default" }
    expect(matchesDNSRule(rule, "google")).toBe(true)
    expect(matchesDNSRule(rule, "dns-remote")).toBe(true)
    expect(matchesDNSRule(rule, "route")).toBe(true)
    expect(matchesDNSRule(rule, "reject")).toBe(false)
  })

  it("filters by type/action facets and summarizes buckets", () => {
    const servers = [
      { tag: "dns-remote", type: "https", server: "1.1.1.1" },
      { tag: "dns-local", type: "udp", server: "127.0.0.1" },
      { tag: "legacy", address: "8.8.8.8" },
    ]
    const rules = [
      { action: "route", server: "dns-remote", domain: ["google.com"] },
      { action: "reject", domain: ["ads.example"] },
      { server: "dns-local", domain: ["intranet"] },
    ]
    expect(dnsServerType(servers[2])).toBe("legacy")
    expect(dnsRuleAction(rules[2])).toBe("route")
    expect(matchesDNSServerType(servers[0], "https")).toBe(true)
    expect(matchesDNSRuleAction(rules[1], "reject")).toBe(true)
    expect(filterDNSServers(servers, { serverType: "https" }).map((item) => item.tag)).toEqual(["dns-remote"])
    expect(filterDNSRules(rules, { ruleAction: "reject" })).toHaveLength(1)
    expect(summarizeDNSServerTypes(servers)).toEqual({
      total: 3,
      buckets: [
        { type: "https", count: 1 },
        { type: "legacy", count: 1 },
        { type: "udp", count: 1 },
      ],
    })
    expect(summarizeDNSRuleActions(rules)).toEqual({
      total: 3,
      buckets: [
        { action: "route", count: 2 },
        { action: "reject", count: 1 },
      ],
    })
    expect(summarizeDNSServerTypes(servers, "remote").buckets.map((bucket) => bucket.type)).toEqual(["https"])
    expect(summarizeDNSRuleActions(rules, "ads").buckets.map((bucket) => bucket.action)).toEqual(["reject"])
    expect(dnsServerFiltersActive({ serverType: "https" })).toBe(true)
    expect(dnsRuleFiltersActive({ ruleAction: "reject" })).toBe(true)
    expect(dnsServerFiltersActive({})).toBe(false)
    expect(dnsRuleFiltersActive({})).toBe(false)
  })

  it("parses and builds DNS deep-link query strings", () => {
    expect(parseDNSSearchParams(new URLSearchParams("sq=remote&rq=ads&stype=https&raction=reject"))).toEqual({
      servers: "remote",
      rules: "ads",
      serverType: "https",
      ruleAction: "reject",
    })
    expect(buildDNSHref({ servers: "remote", serverType: "https" })).toBe("/policy/dns?sq=remote&stype=https")
    expect(buildDNSHref({ rules: "ads", ruleAction: "reject" })).toBe("/policy/dns?rq=ads&raction=reject")
    expect(buildDNSHref({})).toBe("/policy/dns")
    expect(toDNSSearchParams({ servers: "  " }).toString()).toBe("")
  })
})
