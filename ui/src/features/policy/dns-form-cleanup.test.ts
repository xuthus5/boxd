import { describe, expect, it } from "vitest"

import {
  changeDNSRuleType,
  changeDNSServerType,
  dnsRuleMatchFields,
  dnsServerFields,
} from "@/features/policy/dns-form-model"

describe("DNS cleanup-only field transitions", () => {
  it("exposes active address match fields while keeping removed geo fields hidden", () => {
    const visible = dnsRuleMatchFields.map((field) => field.path)
    expect(visible).toContain("ip_accept_any")
    expect(visible).toContain("interface_address")
    expect(visible).toContain("network_interface_address")
    expect(visible).toContain("default_interface_address")
    expect(visible).toContain("rule_set_ip_cidr_accept_empty")
    expect(visible).not.toContain("geosite")
    expect(visible).not.toContain("source_geoip")
    expect(visible).not.toContain("geoip")
  })

  it("keeps cleanup-only TLS paths out of visible server metadata", () => {
    const visible = dnsServerFields.https.map((field) => field.path)
    expect(visible).toContain("tls.certificate_path")
    expect(visible).toContain("headers")
    expect(visible).not.toContain("tls.min_version")
    expect(visible).not.toContain("tls.ech.enabled")
    expect(visible).not.toContain("tls.reality.public_key")
  })

  it("removes every non-deprecated default-only rule path when changing to logical", () => {
    expect(changeDNSRuleType({
      geosite: ["cn"], source_geoip: ["private"], geoip: ["cn"], ip_accept_any: true,
      interface_address: { eth0: ["192.0.2.0/24"] },
      network_interface_address: { wifi: ["192.0.2.0/24"] },
      default_interface_address: ["192.0.2.0/24"], rule_set_ip_cidr_accept_empty: true,
      rule_set_ipcidr_match_source: true, invert: true, action: "reject", custom: "keep",
    }, "logical")).toEqual({ type: "logical", invert: true, action: "reject", custom: "keep" })
  })

  it("cleans complete TLS client paths while preserving unknown nested siblings", () => {
    const tls = {
      min_version: "1.2", max_version: "1.3", cipher_suites: ["TLS_AES_128_GCM_SHA256"],
      curve_preferences: ["X25519"], certificate: ["cert"], certificate_path: "/cert.pem",
      certificate_public_key_sha256: ["hash"], client_certificate: ["client-cert"],
      client_certificate_path: "/client.pem", client_key: ["key"], client_key_path: "/key.pem",
      fragment: true, fragment_fallback_delay: "500ms", record_fragment: true, kernel_tx: true, kernel_rx: true,
      ech: { enabled: true, config: ["config"], config_path: "/ech", query_server_name: "cloudflare-ech.com", custom: "keep" },
      utls: { enabled: true, fingerprint: "chrome", custom: "keep" },
      reality: { enabled: true, public_key: "key", short_id: "id", custom: "keep" },
      custom: "keep",
    }
    const unknownTLS = {
      ech: { custom: "keep" }, utls: { custom: "keep" }, reality: { custom: "keep" }, custom: "keep",
    }
    expect(changeDNSServerType({ type: "https", server: "dns.example", tls }, "udp"))
      .toEqual({ type: "udp", server: "dns.example", tls: unknownTLS })
    expect(changeDNSServerType({ type: "https", server: "dns.example", tls }, "local"))
      .toEqual({ type: "local", tls: unknownTLS })
  })

  it("retains valid cleanup-only target fields from unknown sources and drops invalid kinds", () => {
    expect(changeDNSServerType({
      type: "custom", server: "dns.example", inet4_range: "198.18.0.0/15",
      tls: {
        min_version: "1.2", cipher_suites: ["TLS_AES_128_GCM_SHA256"], client_key_path: 123,
        ech: { enabled: true, custom: "keep" }, custom: "keep",
      },
      payload: "keep",
    }, "tls")).toEqual({
      type: "tls", server: "dns.example",
      tls: {
        min_version: "1.2", cipher_suites: ["TLS_AES_128_GCM_SHA256"],
        ech: { enabled: true, custom: "keep" }, custom: "keep",
      },
      payload: "keep",
    })
  })

  it("removes scalar domain resolvers when a modern server changes to legacy", () => {
    expect(changeDNSServerType({
      type: "udp", server: "1.1.1.1", domain_resolver: "dns", custom: "keep",
    }, "legacy")).toEqual({ custom: "keep" })
  })

  it("removes scalar domain resolvers when a modern server changes to an unknown type", () => {
    expect(changeDNSServerType({
      type: "udp", server: "1.1.1.1", domain_resolver: "dns", custom: "keep",
    }, "custom")).toEqual({ type: "custom", custom: "keep" })
  })

  it("retains scalar domain resolvers when an unknown source changes to a modern server", () => {
    expect(changeDNSServerType({
      type: "custom", domain_resolver: "dns", payload: "keep",
    }, "udp")).toEqual({ type: "udp", domain_resolver: "dns", payload: "keep" })
  })

  it("applies leaf cleanup to object resolvers while preserving unknown siblings", () => {
    expect(changeDNSServerType({
      type: "udp",
      domain_resolver: { server: "dns", strategy: "prefer_ipv4", custom: "keep" },
    }, "hosts")).toEqual({ type: "hosts", domain_resolver: { custom: "keep" } })
    expect(changeDNSServerType({
      type: "custom",
      domain_resolver: { server: "dns", strategy: "prefer_ipv6", custom: "keep" },
    }, "tls")).toEqual({
      type: "tls",
      domain_resolver: { server: "dns", strategy: "prefer_ipv6", custom: "keep" },
    })
  })
})
