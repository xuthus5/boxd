import { describe, expect, it } from "vitest"

import { preflightConfig } from "@/features/config/config-preflight"
import type { SingBoxConfig } from "@/lib/api/types"

function bootstrapIssue(config: SingBoxConfig) {
  return preflightConfig(config).find((item) => (
    item.code === "dns_dependency_cycle" && item.path === "dns.servers[0].detour"
  ))
}

function proxyBootstrapConfig(resolver?: unknown): SingBoxConfig {
  return {
    outbounds: [{
      type: "socks",
      tag: "proxy",
      server: "proxy.example.com",
      server_port: 1080,
      ...(resolver === undefined ? {} : { domain_resolver: resolver }),
    }],
    dns: { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1", detour: "proxy" }], final: "remote" },
    route: { final: "proxy" },
  }
}

function detouredProxyBootstrapConfig(): SingBoxConfig {
  return {
    outbounds: [
      { type: "socks", tag: "proxy", server: "proxy.example.com", server_port: 1080, detour: "underlay" },
      { type: "direct", tag: "underlay", bind_interface: "lo" },
    ],
    dns: { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1", detour: "proxy" }], final: "remote" },
    route: { final: "proxy", default_domain_resolver: "remote" },
  }
}

describe("DNS outbound bootstrap preflight", () => {
  it("detects explicit, default, and single-server resolver cycles", () => {
    const configs = [
      proxyBootstrapConfig("remote"),
      proxyBootstrapConfig({ server: "remote" }),
      { ...proxyBootstrapConfig(), route: { final: "proxy", default_domain_resolver: "remote" } },
      proxyBootstrapConfig(),
      detouredProxyBootstrapConfig(),
    ]

    for (const config of configs) {
      expect(bootstrapIssue(config)).toEqual(expect.objectContaining({
        severity: "error",
        reference: "proxy",
        relatedPath: "outbounds[0].tag",
      }))
    }
  })

  it("follows selector members and endpoint resolver-on-detour behavior", () => {
    const selector = bootstrapIssue({
      outbounds: [
        { type: "selector", tag: "group", outbounds: ["proxy"] },
        { type: "socks", tag: "proxy", server: "proxy.example.com", server_port: 1080, domain_resolver: "remote" },
      ],
      dns: { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1", detour: "group" }], final: "remote" },
      route: { final: "group" },
    })
    expect(selector).toEqual(expect.objectContaining({ reference: "group", relatedPath: "outbounds[0].tag" }))

    const wireguard = bootstrapIssue({
      endpoints: [{
        type: "wireguard",
        tag: "edge",
        peers: [{ address: "wg.example.com", port: 51820 }],
        domain_resolver: "remote",
      }],
      dns: { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1", detour: "edge" }], final: "remote" },
      route: { final: "edge" },
    })
    expect(wireguard).toEqual(expect.objectContaining({ reference: "edge", relatedPath: "endpoints[0].tag" }))

    const tailscale = bootstrapIssue({
      endpoints: [{ type: "tailscale", tag: "tail" }],
      dns: { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1", detour: "tail" }], final: "remote" },
      route: { final: "tail", default_domain_resolver: "remote" },
    })
    expect(tailscale).toEqual(expect.objectContaining({ reference: "tail", relatedPath: "endpoints[0].tag" }))

    const selectorDetour = detouredProxyBootstrapConfig()
    selectorDetour.outbounds = [
      { type: "socks", tag: "proxy", server: "proxy.example.com", server_port: 1080, detour: "group" },
      { type: "selector", tag: "group", outbounds: ["underlay"] },
      { type: "direct", tag: "underlay", bind_interface: "lo" },
    ]
    expect(bootstrapIssue(selectorDetour)).toEqual(expect.objectContaining({ reference: "proxy" }))
  })

  it("ignores IP servers, unrelated detours, and ambiguous implicit resolvers", () => {
    const ipServer = proxyBootstrapConfig("remote")
    ipServer.outbounds = [{ ...ipServer.outbounds?.[0], server: "192.0.2.1" }]
    expect(bootstrapIssue(ipServer)).toBeUndefined()

    const unrelated = proxyBootstrapConfig("remote")
    unrelated.outbounds = [...(unrelated.outbounds ?? []), { type: "direct", tag: "direct" }]
    unrelated.dns = { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1", detour: "direct" }], final: "remote" }
    expect(bootstrapIssue(unrelated)).toBeUndefined()

    const ambiguous = proxyBootstrapConfig()
    ambiguous.dns = {
      servers: [
        { type: "udp", tag: "remote", server: "1.1.1.1", detour: "proxy" },
        { type: "udp", tag: "backup", server: "1.0.0.1" },
      ],
      final: "remote",
    }
    expect(bootstrapIssue(ambiguous)).toBeUndefined()

    const directDNS = detouredProxyBootstrapConfig()
    directDNS.outbounds = [
      { type: "socks", tag: "proxy", server: "192.0.2.2", server_port: 1080 },
      { type: "direct", tag: "underlay", bind_interface: "lo" },
    ]
    directDNS.dns = { servers: [{ type: "udp", tag: "remote", server: "1.1.1.1", detour: "underlay" }], final: "remote" }
    expect(bootstrapIssue(directDNS)).toBeUndefined()
  })
})
