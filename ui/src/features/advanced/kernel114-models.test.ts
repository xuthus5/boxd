import { describe, expect, it } from "vitest"

import { createEndpointDraft, changeEndpointType, isEndpointReady, prepareEndpointObject, summarizeEndpoint } from "@/features/advanced/endpoints-form-model"
import { createServiceDraft, isServiceReady, prepareServiceObject } from "@/features/advanced/services-form-model"
import { createSharedResource, isSharedResourceReady, parseSharedResources, sharedResourceFields, updateSharedResource } from "@/features/advanced/shared-resource-model"
import { currentTypeFields } from "@/features/config/current-type-fields"
import { preflightConfig } from "@/features/config/config-preflight"
import { changeDNSAction, isDNSRuleComplete } from "@/features/policy/dns-form-model"
import { policyOutboundTags, policyRuleSetTags } from "@/features/policy/policy-form-model"
import { isRouteRuleComplete } from "@/features/policy/route-form-model"

describe("sing-box 1.14 configuration models", () => {
  it("keeps DNS response matches when changing evaluate into respond", () => {
    expect(changeDNSAction({ action: "evaluate", server: "remote", tag: "answer", timeout: "5s", race: true,
      match_response: "previous", response_rcode: "NOERROR", custom: 1 }, "respond"))
      .toEqual({ action: "respond", race: true, match_response: "previous", response_rcode: "NOERROR", custom: 1 })
    expect(isDNSRuleComplete({ action: "evaluate" })).toBe(false)
    expect(isDNSRuleComplete({ action: "evaluate", server: "remote" })).toBe(true)
    expect(isDNSRuleComplete({ action: "respond", match_response: "answer" })).toBe(true)
  })

  it("allows match-only nested rules and rejects all nested action options", () => {
    const children = [{ domain_suffix: ["example.org"] }, { type: "logical", mode: "or", rules: [{ port: [443] }] }]
    expect(isDNSRuleComplete({ type: "logical", mode: "and", rules: children, action: "reject" })).toBe(true)
    expect(isRouteRuleComplete({ type: "logical", mode: "and", rules: children, action: "reject" })).toBe(true)
    for (const child of [{ action: "reject" }, { server: "remote" }, { race: true }, { strategy: "ipv4_only" }]) {
      expect(isDNSRuleComplete({ type: "logical", mode: "and", rules: [child], action: "reject" })).toBe(false)
    }
    expect(isRouteRuleComplete({ type: "logical", mode: "and", rules: [{ outbound: "proxy" }], action: "reject" })).toBe(false)
    expect(isRouteRuleComplete({ action: "resolve" })).toBe(true)
  })

  it("supports VPN drafts, required fields, and opaque TLS options", () => {
    const client = { ...createEndpointDraft("openvpn-client"), tag: "vpn", server: "vpn.example", tls: { control_wrap: { type: "tls_crypt", key_path: "/vpn/key" } }, udp_timeout: "7m", udp_mapping: "address_dependent" }
    expect(isEndpointReady(client)).toBe(true)
    expect(prepareEndpointObject(client)).toEqual(client)
    expect(summarizeEndpoint(client)).toMatchObject({ type: "openvpn-client", detail: "vpn.example" })
    expect(isEndpointReady({ type: "openvpn-client", tag: "vpn", servers: [{ server: "one.example" }] })).toBe(true)
    expect(isEndpointReady({ type: "openvpn-client", tag: "vpn", servers: [null] })).toBe(false)
    expect(isEndpointReady({ type: "openvpn-client", tag: "vpn", servers: [] })).toBe(false)
    expect(isEndpointReady({ ...createEndpointDraft("openconnect"), tag: "vpn", server: "https://vpn.example" })).toBe(true)
    expect(isEndpointReady({ type: "openconnect", tag: "vpn" })).toBe(false)
    expect(isEndpointReady({ ...createEndpointDraft("openvpn-server"), tag: "vpn", address: ["10.8.0.1/24"] })).toBe(true)
    expect(isEndpointReady({ type: "openvpn-server", tag: "vpn", address: ["10.8.0.1/24"], listen_port: 0 })).toBe(false)
    expect(isEndpointReady({ type: "unknown", tag: "future" })).toBe(false)
  })

  it("changes VPN types without retaining credentials from another protocol", () => {
    const source = { type: "openconnect", tag: "vpn", cookie: "session-cookie", password: "secret", flavor: "gp" }
    expect(changeEndpointType(source, "openvpn-server")).toEqual({ type: "openvpn-server", tag: "vpn", listen: "127.0.0.1", listen_port: 1194, address: [], mode: "tls" })
    expect(changeEndpointType(source, "unknown")).toBe(source)
  })

  it("retains UDP settings shared by valid endpoint types", () => {
    const wireguard = { type: "wireguard", tag: "wg", address: ["10.0.0.1/24"], private_key: "private", udp_timeout: "10m", udp_filtering: "address_dependent" }
    expect(prepareEndpointObject(wireguard)).toEqual(wireguard)
    expect(prepareEndpointObject({ type: "tailscale", tag: "ts", udp_timeout: "10m" })).toEqual({ type: "tailscale", tag: "ts", udp_timeout: "10m" })
  })

  it("supports API, realm, and USB services with their own requirements", () => {
    const api = { ...createServiceDraft("api"), tag: "api", dashboard: { enabled: true, http_client: "downloads" } }
    expect(isServiceReady(api)).toBe(true)
    expect(prepareServiceObject(api)).toEqual(api)
    expect(isServiceReady(createServiceDraft("hysteria-realm"))).toBe(false)
    expect(isServiceReady({ ...createServiceDraft("hysteria-realm"), users: [{ name: "user", token: "token" }] })).toBe(true)
    expect(isServiceReady(createServiceDraft("usbip-server"))).toBe(true)
    expect(isServiceReady(createServiceDraft("usbip-client"))).toBe(false)
    expect(isServiceReady({ ...createServiceDraft("usbip-client"), server: "usb.example" })).toBe(true)
    expect(prepareServiceObject({ type: "hysteria-realm", listen: "127.0.0.1", listen_port: 8080, users: [{ name: "user", token: "token" }] }).users)
      .toEqual([{ name: "user", token: "token" }])
  })

  it("includes VPN destinations and multiple rule-set tags in references", () => {
    const config = { outbounds: [{ tag: "direct" }, { tag: "vpn" }], endpoints: [{ type: "openconnect", tag: "vpn" }] }
    expect(policyOutboundTags(config)).toEqual(["direct", "vpn"])
    expect(policyOutboundTags(config, "vpn")).toEqual(["direct"])
    expect(policyOutboundTags(undefined)).toEqual([])
    expect(policyRuleSetTags({ rule_set: [{ tag: ["cn", "private"] }, { tag: "cn" }] })).toEqual(["cn", "private"])
    expect(preflightConfig({ route: { rule_set: [{ type: "local", tag: ["cn", "private"], path: "/{tag}.srs" }], rules: [{ rule_set: "private", action: "reject" }] } })).toEqual([])
  })
})

describe("shared configuration resources", () => {
  it("parses only object arrays and preserves nested fields", () => {
    expect(parseSharedResources('[{"tag":"x","unknown":{"keep":true}}]')).toEqual([{ tag: "x", unknown: { keep: true } }])
    for (const raw of ["{", "null", "{}", "[null]", "[1]"]) expect(parseSharedResources(raw)).toBeNull()
    expect(parseSharedResources("[]")).toEqual([])
  })

  it("requires usable shared tags and namespace/provider targets", () => {
    expect(createSharedResource("http_clients")).toEqual({ tag: "", engine: "go", version: 2 })
    expect(createSharedResource("network_namespaces")).toEqual({ type: "default", tag: "" })
    expect(createSharedResource("certificate_providers")).toEqual({ type: "acme", tag: "" })
    expect(isSharedResourceReady("http_clients", { tag: "client" })).toBe(true)
    expect(isSharedResourceReady("http_clients", { tag: " " })).toBe(false)
    expect(isSharedResourceReady("network_namespaces", { type: "default", tag: "ns" })).toBe(false)
    expect(isSharedResourceReady("network_namespaces", { type: "default", tag: "ns", path: "/proc/1/ns/net" })).toBe(true)
    expect(isSharedResourceReady("network_namespaces", { type: "unshare", tag: "ns" })).toBe(true)
    expect(isSharedResourceReady("network_namespaces", { type: "unknown", tag: "ns" })).toBe(false)
    expect(isSharedResourceReady("certificate_providers", { type: "tailscale", tag: "tls" })).toBe(false)
    expect(isSharedResourceReady("certificate_providers", { type: "tailscale", tag: "tls", endpoint: "ts" })).toBe(true)
    expect(isSharedResourceReady("certificate_providers", { type: "acme", tag: "tls" })).toBe(true)
  })

  it("cleans version-specific HTTP options only after an explicit version change", () => {
    const current = { tag: "client", version: 3, initial_packet_size: 1300, idle_timeout: "1m", headers: { X: "1" } }
    expect(updateSharedResource("http_clients", current, { ...current, version: 1 })).toEqual({ tag: "client", version: 1, headers: { X: "1" } })
    expect(updateSharedResource("http_clients", current, { ...current, version: 2 })).toEqual({ tag: "client", version: 2, idle_timeout: "1m", headers: { X: "1" } })
    expect(updateSharedResource("http_clients", current, current)).toBe(current)
    expect(updateSharedResource("certificate_providers", { type: "acme", tag: "tls", account_key: "secret" }, { type: "tailscale", tag: "tls", account_key: "secret" })).toEqual({ type: "tailscale", tag: "tls" })
    expect(updateSharedResource("network_namespaces", { type: "default" }, { type: "unshare" })).toEqual({ type: "unshare", tag: "" })
  })

  it("selects protocol fields without deleting same-name fields belonging to the selected type", () => {
    const fields = [{ path: "users", label: "users", when: { path: "type", is: ["api", "hysteria-realm"] } },
      { path: "users", label: "users", when: [{ path: "type", is: "ccm" }, { path: "enabled", is: true }] },
      { path: "tag", label: "tag" }]
    expect(currentTypeFields(fields, "hysteria-realm")).toEqual([fields[0], fields[2]])
    expect(sharedResourceFields("network_namespaces", { type: "unshare" })).toContainEqual({ path: "pid_file", label: "pidFile" })
    expect(sharedResourceFields("certificate_providers", { type: "missing" })).toHaveLength(2)
  })
})
