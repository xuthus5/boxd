import { describe, expect, it } from "vitest"

import { summarizeDNSRule, summarizeDNSServer } from "@/features/policy/dns-form-model"

describe("DNS server summary completeness", () => {
  it("summarizes legacy and modern endpoints with stable routing details", () => {
    expect(summarizeDNSServer({
      tag: "legacy", address: "https://dns.example/dns-query", detour: "direct", strategy: "prefer_ipv4",
    })).toEqual({
      type: "legacy",
      detail: "https://dns.example/dns-query · tag legacy · detour direct · strategy prefer_ipv4",
    })
    expect(summarizeDNSServer({
      type: "https", tag: "remote", server: "dns.example", server_port: 443,
      detour: "proxy", strategy: "prefer_ipv6",
    })).toEqual({
      type: "https", detail: "dns.example:443 · tag remote · detour proxy · strategy prefer_ipv6",
    })
  })

  it("uses interface or tag summaries for non-address server types", () => {
    expect(summarizeDNSServer({ type: "dhcp", tag: "lan", interface: "eth0" }))
      .toEqual({ type: "dhcp", detail: "eth0 · tag lan" })
    expect(summarizeDNSServer({ type: "hosts", tag: "hosts" }))
      .toEqual({ type: "hosts", detail: "tag hosts" })
    expect(summarizeDNSServer({ type: "fakeip", tag: "fake" }))
      .toEqual({ type: "fakeip", detail: "tag fake" })
  })

})

describe("DNS rule summary completeness", () => {
  it("summarizes logical mode without expanding child rules", () => {
    expect(summarizeDNSRule({ type: "logical", mode: "and", rules: [{ domain: "example.com" }], action: "reject" }))
      .toEqual({ matches: ["mode:and"], action: "reject" })
  })

  it("localizes logical and enabled boolean match labels", () => {
    const visited: string[] = []
    const summary = summarizeDNSRule({
      type: "logical", mode: "and", rules: [], source_ip_is_private: true,
      rule_set_ip_cidr_match_source: true, network_is_expensive: false, action: "reject",
    }, {
      logicalMode: (value) => `logical:${value}`,
      matchLabel: (path) => { visited.push(path); return `label:${path}` },
    })

    expect(summary.matches).toEqual([
      "logical:and", "label:source_ip_is_private", "label:rule_set_ip_cidr_match_source",
    ])
    expect(visited).toEqual(["source_ip_is_private", "rule_set_ip_cidr_match_source"])
  })

  it("keeps the real route action while appending its target", () => {
    expect(summarizeDNSRule({ server: "dns-local" }).action).toBe("route · dns-local")
    expect(summarizeDNSRule({ action: "route", server: "dns-remote" }).action).toBe("route · dns-remote")
    expect(summarizeDNSRule({ action: "route" }).action).toBe("route")
    expect(summarizeDNSRule({ action: "reject", server: "ignored" }).action).toBe("reject")
    expect(summarizeDNSRule({ action: "predefined" }).action).toBe("predefined")
    expect(summarizeDNSRule({ action: "custom" }).action).toBe("custom")
  })
})
