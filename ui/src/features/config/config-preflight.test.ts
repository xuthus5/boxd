import { describe, expect, it } from "vitest"

import { preflightConfig, type ConfigPreflightIssue } from "@/features/config/config-preflight"
import type { SingBoxConfig } from "@/lib/api/types"

function issuesFor(config: SingBoxConfig) {
  return preflightConfig(config)
}

function hasIssue(issues: ConfigPreflightIssue[], code: ConfigPreflightIssue["code"], path: string) {
  return issues.some((item) => item.code === code && item.path === path)
}

describe("preflightConfig", () => {
  it("accepts valid outbound, DNS, route, and nested rule references", () => {
    const issues = issuesFor({
      outbounds: [
        { type: "direct", tag: "direct" },
        { type: "selector", tag: "proxy", outbounds: ["direct"], default: "direct", domain_resolver: "remote" },
      ],
      endpoints: [{ type: "wireguard", tag: "edge", detour: "direct" }],
      dns: {
        servers: [
          { type: "local", tag: "local" },
          { type: "https", tag: "remote", detour: "direct", domain_resolver: "local" },
        ],
        final: "remote",
        rules: [{ action: "route", server: "remote", rule_set: ["geo"], rules: [{ server: "local" }] }],
      },
      route: {
        final: "proxy",
        default_domain_resolver: "remote",
        rule_set: [{ tag: "geo", type: "inline" }],
        rules: [{ outbound: "proxy", rule_set: "geo", rules: [{ outbound: "direct" }] }],
        geoip: { download_detour: "direct" },
        geosite: { download_detour: "direct" },
      },
      experimental: { clash_api: { external_ui_download_detour: "direct" } },
    })
    expect(issues).toEqual([])
  })

  it("finds duplicate tags across sing-box namespaces", () => {
    const issues = issuesFor({
      inbounds: [{ tag: "in" }, { tag: "in" }],
      outbounds: [{ tag: "same" }, { tag: "same" }],
      endpoints: [{ tag: "same" }],
      dns: { servers: [{ tag: "dns" }, { tag: "dns" }] },
      route: { rule_set: [{ tag: "rules" }, { tag: "rules" }, {}] },
    })
    expect(hasIssue(issues, "duplicate_tag", "inbounds[1].tag")).toBe(true)
    expect(hasIssue(issues, "duplicate_tag", "outbounds[1].tag")).toBe(true)
    expect(hasIssue(issues, "duplicate_tag", "endpoints[0].tag")).toBe(true)
    expect(hasIssue(issues, "duplicate_tag", "dns.servers[1].tag")).toBe(true)
    expect(hasIssue(issues, "duplicate_tag", "route.rule_set[1].tag")).toBe(true)
    expect(hasIssue(issues, "missing_tag", "route.rule_set[2].tag")).toBe(true)
  })

  it("reports dangling outbound references with precise paths", () => {
    const issues = issuesFor({
      outbounds: [{ type: "selector", tag: "group", outbounds: ["missing-member"], default: "missing-default", detour: "missing-detour", domain_resolver: "missing-dns" }],
      endpoints: [{ type: "wireguard", tag: "edge", detour: "missing-endpoint-detour" }],
      route: {
        final: "missing-final",
        rule_set: [{ tag: "remote", type: "remote", download_detour: "missing-rule-set-detour" }],
        rules: [{ outbound: "missing-rule-outbound", rules: [{ outbound: "missing-nested-outbound" }] }],
        geoip: { download_detour: "missing-geoip-detour" },
        geosite: { download_detour: "missing-geosite-detour" },
      },
      experimental: { clash_api: { external_ui_download_detour: "missing-ui-detour" } },
    })
    expect(hasIssue(issues, "missing_outbound", "outbounds[0].outbounds[0]")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "outbounds[0].default")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "outbounds[0].detour")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "endpoints[0].detour")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "route.final")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "route.rule_set[0].download_detour")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "route.rules[0].outbound")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "route.rules[0].rules[0].outbound")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "route.geoip.download_detour")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "route.geosite.download_detour")).toBe(true)
    expect(hasIssue(issues, "missing_outbound", "experimental.clash_api.external_ui_download_detour")).toBe(true)
    expect(issues.some((item) => item.code === "missing_outbound" && item.reference === "missing-dns")).toBe(false)
  })

  it("reports dangling DNS and rule-set references", () => {
    const issues = issuesFor({
      dns: {
        servers: [{
          tag: "dns1",
          detour: "missing-outbound",
          domain_resolver: { server: "missing-resolver" },
          address_resolver: "missing-address-resolver",
        }],
        final: "missing-final",
        rules: [{
          server: "missing-rule-server",
          rule_set: "missing-rule-set",
          rules: [{ server: "missing-nested-server", rule_set: ["missing-nested-set"] }],
        }],
      },
      route: {
        rules: [{ action: "resolve", server: "missing-route-server", rule_set: ["missing-route-set"] }],
      },
    })
    expect(hasIssue(issues, "missing_outbound", "dns.servers[0].detour")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "dns.servers[0].domain_resolver.server")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "dns.servers[0].address_resolver")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "dns.final")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "dns.rules[0].server")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "dns.rules[0].rules[0].server")).toBe(true)
    expect(hasIssue(issues, "missing_rule_set", "dns.rules[0].rule_set")).toBe(true)
    expect(hasIssue(issues, "missing_rule_set", "dns.rules[0].rules[0].rule_set[0]")).toBe(true)
    expect(hasIssue(issues, "missing_rule_set", "route.rules[0].rule_set[0]")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "route.rules[0].server")).toBe(true)
  })

  it("warns about empty selector and urltest groups", () => {
    const issues = issuesFor({
      outbounds: [
        { type: "selector", tag: "selector", outbounds: [] },
        { type: "urltest", tag: "urltest", outbounds: "invalid" },
        { type: "direct", tag: "direct" },
      ],
    })
    expect(issues).toEqual([
      expect.objectContaining({ code: "empty_group", path: "outbounds[0].outbounds", severity: "warning", reference: "selector" }),
      expect.objectContaining({ code: "empty_group", path: "outbounds[1].outbounds", severity: "warning", reference: "urltest" }),
    ])
  })

  it("supports implicit tags and ignores incomplete non-object sections", () => {
    const issues = issuesFor({
      inbounds: [null, { tag: "in" }],
      outbounds: [{ type: "direct" }],
      dns: { servers: [{ type: "local", tag: "local" }, null], final: "local", rules: [null] },
      route: { final: "0", rules: [null], rule_set: [null] },
      experimental: "invalid",
    })
    expect(issues).toEqual([])
  })

  it("resolves DNS references declared later and rejects an absent local tag", () => {
    const forwardIssues = issuesFor({
      dns: {
        servers: [
          { tag: "first", domain_resolver: "later" },
          { tag: "later" },
        ],
        final: "later",
      },
      route: { rules: [{ action: "resolve", server: "later" }] },
    })
    expect(forwardIssues).toEqual([])

    const missingIssues = issuesFor({ dns: { final: "local" } })
    expect(hasIssue(missingIssues, "missing_dns_server", "dns.final")).toBe(true)
  })

  it("handles string and object resolver forms", () => {
    const issues = issuesFor({
      outbounds: [
        { type: "direct", tag: "direct", domain_resolver: { server: "missing-object" } },
        { type: "direct", tag: "second", domain_resolver: 42 },
      ],
      route: { default_domain_resolver: "missing-string" },
      dns: { servers: [{ tag: "one", domain_resolver: "missing-server" }] },
    })
    expect(hasIssue(issues, "missing_dns_server", "outbounds[0].domain_resolver.server")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "route.default_domain_resolver")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "dns.servers[0].domain_resolver")).toBe(true)
  })
})
