import { describe, expect, it } from "vitest"

import { preflightConfig, type ConfigPreflightIssue } from "@/features/config/config-preflight"
import type { SingBoxConfig } from "@/lib/api/types"

function issuesFor(config: SingBoxConfig) {
  return preflightConfig(config)
}

function hasIssue(issues: ConfigPreflightIssue[], code: ConfigPreflightIssue["code"], path: string) {
  return issues.some((item) => item.code === code && item.path === path)
}

function issueByCode(issues: ConfigPreflightIssue[], code: string, path: string) {
  return issues.find((item) => item.code === code && item.path === path)
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
      ntp: { enabled: true, detour: "direct", domain_resolver: { server: "remote" } },
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

  it("rejects empty selector and urltest groups", () => {
    const issues = issuesFor({
      outbounds: [
        { type: "selector", tag: "selector", outbounds: [] },
        { type: "urltest", tag: "urltest", outbounds: "invalid" },
        { type: "direct", tag: "direct" },
      ],
    })
    expect(issues).toEqual([
      expect.objectContaining({ code: "empty_group", path: "outbounds[0].outbounds", severity: "error", reference: "selector" }),
      expect.objectContaining({ code: "empty_group", path: "outbounds[1].outbounds", severity: "error", reference: "urltest" }),
    ])
  })

  it("rejects selector defaults outside the member list", () => {
    const issues = issuesFor({
      outbounds: [
        { type: "direct", tag: "direct" },
        { type: "direct", tag: "other" },
        { type: "selector", tag: "group", outbounds: ["direct"], default: "other" },
      ],
    })
    expect(issueByCode(issues, "invalid_group_default", "outbounds[2].default")).toEqual(expect.objectContaining({
      severity: "error",
      reference: "other",
    }))
  })

  it("detects group and detour dependency cycles", () => {
    const issues = issuesFor({
      outbounds: [
        { type: "selector", tag: "group-a", outbounds: ["group-b"] },
        { type: "urltest", tag: "group-b", outbounds: ["group-a"] },
        { type: "direct", tag: "detour-a", detour: "detour-b" },
        { type: "direct", tag: "detour-b", detour: "detour-a" },
      ],
    })
    expect(issueByCode(issues, "outbound_dependency_cycle", "outbounds[1].outbounds[0]")).toEqual(
      expect.objectContaining({ severity: "error", reference: "group-a", relatedPath: "outbounds[0].tag" }),
    )
    expect(issueByCode(issues, "outbound_dependency_cycle", "outbounds[3].detour")).toEqual(
      expect.objectContaining({ severity: "error", reference: "detour-a", relatedPath: "outbounds[2].tag" }),
    )
  })

  it("detects modern and legacy DNS resolver cycles", () => {
    const issues = issuesFor({
      dns: {
        servers: [
          { type: "https", tag: "modern-a", server: "1.1.1.1", domain_resolver: "modern-b" },
          { type: "https", tag: "modern-b", server: "1.0.0.1", domain_resolver: { server: "modern-a" } },
          { tag: "legacy-a", address: "tls://8.8.8.8", address_resolver: "legacy-b" },
          { tag: "legacy-b", address: "tls://8.8.4.4", address_resolver: "legacy-a" },
        ],
        final: "modern-a",
      },
    })
    expect(issueByCode(issues, "dns_dependency_cycle", "dns.servers[1].domain_resolver.server")).toEqual(
      expect.objectContaining({ severity: "error", reference: "modern-a", relatedPath: "dns.servers[0].tag" }),
    )
    expect(issueByCode(issues, "dns_dependency_cycle", "dns.servers[3].address_resolver")).toEqual(
      expect.objectContaining({ severity: "error", reference: "legacy-a", relatedPath: "dns.servers[2].tag" }),
    )
  })

  it("rejects a modern DNS domain server with no resolution path", () => {
    const missing = issuesFor({
      dns: { servers: [{ type: "udp", tag: "remote", server: "dns.example.com" }] },
    })
    expect(issueByCode(missing, "missing_domain_resolver", "dns.servers[0].server")).toEqual(
      expect.objectContaining({ severity: "error", reference: "remote" }),
    )

    const validConfigs: SingBoxConfig[] = [
      { dns: { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1" }] } },
      { dns: { servers: [{ tag: "legacy", address: "udp://dns.example.com" }] } },
      { dns: { servers: [
        { type: "local", tag: "local" },
        { type: "udp", tag: "remote", server: "dns.example.com", domain_resolver: "local" },
      ] } },
      { dns: { servers: [
        { type: "local", tag: "local" },
        { type: "udp", tag: "remote", server: "dns.example.com", domain_resolver: { server: "local" } },
      ] } },
      {
        outbounds: [{ type: "socks", tag: "proxy", server: "192.0.2.2", server_port: 1080 }],
        dns: { servers: [{ type: "udp", tag: "remote", server: "dns.example.com", detour: "proxy" }] },
      },
    ]
    for (const config of validConfigs) {
      expect(issuesFor(config).some((item) => item.code === "missing_domain_resolver")).toBe(false)
    }
  })

  it("rejects FakeIP defaults and extra FakeIP servers", () => {
    const explicit = issuesFor({ dns: {
      servers: [
        { type: "local", tag: "local" },
        { type: "fakeip", tag: "fake", inet4_range: "198.18.0.0/15" },
      ],
      final: "fake",
    } })
    expect(issueByCode(explicit, "invalid_dns_default", "dns.final")).toEqual(
      expect.objectContaining({ severity: "error", reference: "fake", relatedPath: "dns.servers[1].tag" }),
    )

    const implicit = issuesFor({ dns: {
      servers: [{ tag: "legacy-fake", address: "fakeip" }, { type: "local", tag: "local" }],
      fakeip: { enabled: true, inet4_range: "198.18.0.0/15" },
    } })
    expect(issueByCode(implicit, "invalid_dns_default", "dns.servers[0].address")).toEqual(
      expect.objectContaining({ severity: "error", reference: "legacy-fake" }),
    )

    const multiple = issuesFor({ dns: {
      servers: [
        { type: "local", tag: "local" },
        { type: "fakeip", tag: "first-fake", inet4_range: "198.18.0.0/15" },
        { tag: "extra-fake", address: "fakeip" },
      ],
      fakeip: { enabled: true, inet4_range: "198.18.0.0/15" },
      final: "local",
    } })
    expect(issueByCode(multiple, "multiple_fakeip_dns_servers", "dns.servers[2].address")).toEqual(
      expect.objectContaining({ severity: "error", reference: "extra-fake", relatedPath: "dns.servers[1].tag" }),
    )
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

  it("checks NTP outbound and DNS resolver references", () => {
    const issues = issuesFor({
      outbounds: [{ type: "direct", tag: "direct" }],
      dns: { servers: [{ type: "local", tag: "dns-local" }] },
      ntp: {
        enabled: true,
        detour: "missing-ntp-outbound",
        domain_resolver: { server: "missing-ntp-resolver" },
      },
    })
    expect(hasIssue(issues, "missing_outbound", "ntp.detour")).toBe(true)
    expect(hasIssue(issues, "missing_dns_server", "ntp.domain_resolver.server")).toBe(true)
    expect(issuesFor({
      outbounds: [{ type: "direct", tag: "direct" }],
      dns: { servers: [{ type: "local", tag: "dns-local" }] },
      ntp: { enabled: true, detour: "direct", domain_resolver: "dns-local" },
    })).toEqual([])
    expect(issuesFor({
      ntp: { enabled: false, detour: "missing-disabled-outbound", domain_resolver: "missing-disabled-dns" },
    })).toEqual([])
  })
})
