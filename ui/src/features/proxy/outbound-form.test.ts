import { describe, expect, it } from "vitest"

import {
  changeOutboundTransportType, changeOutboundType, dialerFields, groupFields, outboundTypes,
  protocolFields, serverTypes, transportTypeFields,
} from "@/features/proxy/outbound-form-model"

describe("outbound form metadata", () => {
  it("lists supported outbound types and excludes removed types", () => {
    for (const type of ["direct", "block", "selector", "urltest", "socks", "http", "shadowsocks", "vmess", "vless", "trojan", "naive", "hysteria", "hysteria2", "tuic", "ssh", "tor", "shadowtls", "anytls"]) {
      expect(outboundTypes).toContain(type)
    }
    for (const type of ["dns", "wireguard", "shadowsocksr", "mixed"]) expect(outboundTypes).not.toContain(type)
    expect(serverTypes.has("vless")).toBe(true)
    expect(serverTypes.has("tor")).toBe(false)
  })

  it("models dialer, protocol, and group JSON value types", () => {
    expect(dialerFields).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "domain_resolver.server" }),
      expect.objectContaining({ path: "network_strategy", kind: "select" }),
      expect.objectContaining({ path: "udp_fragment", kind: "select" }),
    ]))
    expect(protocolFields("http")).toEqual(expect.arrayContaining([expect.objectContaining({ path: "headers", kind: "json-object" })]))
    expect(protocolFields("tor")).toEqual(expect.arrayContaining([expect.objectContaining({ path: "torrc", kind: "json-object" })]))
    expect(groupFields("selector")).toEqual(expect.arrayContaining([expect.objectContaining({ path: "outbounds", kind: "list" })]))
    expect(groupFields("urltest")).toEqual(expect.arrayContaining([expect.objectContaining({ path: "tolerance", kind: "number" })]))
  })
})

describe("outbound type transitions", () => {
  it("preserves shared server, dialer, TLS, transport, multiplex, and unknown fields", () => {
    const object = {
      type: "vless", tag: "proxy", server: "old.example.com", server_port: 443, uuid: "secret",
      detour: "direct", tls: { enabled: true, custom: "keep" }, transport: { type: "ws", path: "/ws", custom: "keep" },
      multiplex: { enabled: true, custom: "keep" }, custom: "keep",
    }
    expect(changeOutboundType(object, "trojan")).toEqual({
      type: "trojan", tag: "proxy", server: "old.example.com", server_port: 443, detour: "direct",
      tls: { enabled: true, custom: "keep" }, transport: { type: "ws", path: "/ws", custom: "keep" },
      multiplex: { enabled: true, custom: "keep" }, custom: "keep",
    })
  })

  it("clears TLS client credentials across protocols", () => {
    expect(changeOutboundType({
      type: "vless", server: "host", tls: {
        enabled: true, client_certificate: "certificate", client_certificate_path: "/certificate",
        client_key: "key", client_key_path: "/key", custom: "keep",
      },
    }, "trojan")).toEqual({
      type: "trojan", server: "host", tls: { enabled: true, custom: "keep" },
    })
  })

  it("clears incompatible server, dialer, protocol, and group fields", () => {
    expect(changeOutboundType({ type: "vless", server: "host", server_port: 443, detour: "direct", tls: {}, transport: {}, multiplex: {}, uuid: "secret" }, "block")).toEqual({ type: "block" })
    expect(changeOutboundType({ type: "selector", outbounds: ["a"], default: "a", interrupt_exist_connections: true, custom: "keep" }, "urltest")).toEqual({ type: "urltest", outbounds: ["a"], interrupt_exist_connections: true, custom: "keep" })
    expect(changeOutboundType({ type: "urltest", outbounds: ["a"], url: "https://example.com", interval: "3m", tolerance: 50 }, "selector")).toEqual({ type: "selector", outbounds: ["a"] })
  })

  it("preserves unknown types and same-type objects", () => {
    const object = { type: "custom", server: "host", custom: { enabled: true } }
    expect(changeOutboundType(object, "custom")).toBe(object)
    expect(changeOutboundType(object, "vless")).toEqual({ type: "vless", server: "host", custom: { enabled: true } })
  })

  it("cleans known incompatible and sensitive fields from unknown types", () => {
    expect(changeOutboundType({
      type: "custom", server: "host", server_port: 443, detour: "direct", password: "secret",
      tls: { enabled: true, client_key: "secret", custom: "keep" }, transport: { type: "ws" },
      multiplex: { enabled: true }, custom: "keep",
    }, "block")).toEqual({ type: "block", custom: "keep" })
    expect(changeOutboundType({
      type: "custom", server: "host", password: "secret",
      tls: { enabled: true, client_key: "secret", custom: "keep" }, custom: "keep",
    }, "vless")).toEqual({
      type: "vless", server: "host", tls: { enabled: true, custom: "keep" }, custom: "keep",
    })
  })
})

describe("outbound transport transitions", () => {
  it("distinguishes HTTP list host from HTTPUpgrade string host", () => {
    expect(transportTypeFields("http")).toEqual(expect.arrayContaining([expect.objectContaining({ path: "transport.host", kind: "list" })]))
    expect(transportTypeFields("httpupgrade").find((field) => field.path === "transport.host")?.kind).toBeUndefined()
  })

  it("removes incompatible subtype fields and preserves shared and unknown fields", () => {
    const object = { transport: { type: "http", host: ["a"], path: "/path", method: "GET", headers: { X: "1" }, idle_timeout: "1m", custom: "keep" } }
    expect(changeOutboundTransportType(object, "httpupgrade")).toEqual({ transport: { type: "httpupgrade", path: "/path", headers: { X: "1" }, custom: "keep" } })
    expect(changeOutboundTransportType({ transport: { type: "custom", host: ["a"], path: "/path", method: "GET", headers: { X: "1" }, custom: "keep" } }, "httpupgrade")).toEqual({ transport: { type: "httpupgrade", path: "/path", headers: { X: "1" }, custom: "keep" } })
    expect(changeOutboundTransportType({ transport: { type: "custom", host: [1], path: "/path", custom: "keep" } }, "http")).toEqual({ transport: { type: "http", path: "/path", custom: "keep" } })
  })
})

describe("outbound field compatibility branches", () => {
  it("matchesField type branches preserve compatible values", () => {
    // socks keeps username/password; http headers/path removed by incompatibility
    expect(changeOutboundType({
      type: "http", headers: { X: "1" }, path: "/x", username: "u", password: "p",
    }, "socks")).toEqual({ type: "socks" })

    expect(changeOutboundType({
      type: "selector", outbounds: ["a", "b"], default: "a", interrupt_exist_connections: true,
    }, "urltest")).toEqual({ type: "urltest", outbounds: ["a", "b"], interrupt_exist_connections: true })

    // boolean mismatch should drop interrupt flag
    expect(changeOutboundType({
      type: "selector", outbounds: ["a"], interrupt_exist_connections: "yes" as never,
    }, "urltest")).toEqual({ type: "urltest", outbounds: ["a"] })

    // transport shared fields kept when compatible types
    expect(changeOutboundTransportType({
      transport: { type: "ws", path: "/ws", headers: { A: "1" }, max_early_data: 1 },
    }, "http")).toEqual({
      transport: { type: "http", path: "/ws", headers: { A: "1" } },
    })

    // invalid list/json shaped values dropped when switching protocols
    expect(changeOutboundType({
      type: "shadowsocks", method: "aes-128-gcm", password: "secret", plugin_opts: { foo: 1 } as never,
    }, "http")).toEqual({ type: "http" })

    // number array is incompatible with list(string) and is dropped; credentials also cleared
    expect(changeOutboundType({
      type: "hysteria", server_ports: [443, 8443], hop_interval: "30s", up_mbps: 100, down_mbps: 100, auth_str: "s",
    }, "hysteria2")).toEqual({ type: "hysteria2", hop_interval: "30s", up_mbps: 100, down_mbps: 100 })

    // invalid string-field value is dropped on protocol switch
    expect(changeOutboundType({
      type: "socks", username: [{ name: "x" }] as never, password: "p",
    }, "http")).toEqual({ type: "http" })

    // list accepts string form and string arrays
    expect(changeOutboundType({
      type: "hysteria", server_ports: "443,8443", network: "tcp",
    }, "hysteria2")).toEqual({ type: "hysteria2", server_ports: "443,8443", network: "tcp" })
    expect(changeOutboundType({
      type: "hysteria", server_ports: ["443", "8443"],
    }, "hysteria2")).toEqual({ type: "hysteria2", server_ports: ["443", "8443"] })

    // json-object invalid array is dropped when switching into a protocol that expects object headers
    expect(changeOutboundType({
      type: "socks", headers: ["bad"] as never, username: "u", password: "p",
    }, "http")).toEqual({ type: "http" })
  })
})
