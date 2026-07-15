import { describe, expect, it } from "vitest"

import {
  changeRouteAction,
  changeRouteRuleType,
  changeRuleSetType,
  routeActionFields,
  routeActions,
  routeGlobalFields,
  routeMatchFields,
  routeRuleSets,
  routeRules,
  ruleSetTypes,
  setRouteRuleSets,
  setRouteRules,
  summarizeRouteRule,
  summarizeRuleSet,
} from "@/features/policy/route-form-model"

const paths = (fields: readonly { path: string }[]) => fields.map((field) => field.path)

describe("route form metadata", () => {
  it("models route globals from sing-box 1.13", () => {
    expect(paths(routeGlobalFields)).toEqual([
      "final", "find_process", "auto_detect_interface", "override_android_vpn",
      "default_interface", "default_mark", "default_domain_resolver.server",
      "default_domain_resolver.strategy", "default_domain_resolver.disable_cache",
      "default_domain_resolver.rewrite_ttl", "default_domain_resolver.client_subnet",
      "default_network_strategy", "default_network_type", "default_fallback_network_type",
      "default_fallback_delay",
    ])
    expect(routeGlobalFields).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "find_process", kind: "boolean" }),
      expect.objectContaining({ path: "default_domain_resolver.rewrite_ttl", kind: "number" }),
      expect.objectContaining({ path: "default_network_type", kind: "list" }),
    ]))
  })

  it("models every approved default route match field", () => {
    expect(paths(routeMatchFields)).toEqual([
      "type", "inbound", "ip_version", "network", "auth_user", "protocol", "client",
      "domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr",
      "source_ip_is_private", "ip_cidr", "ip_is_private", "source_port",
      "source_port_range", "port", "port_range", "process_name", "process_path",
      "process_path_regex", "package_name", "user", "user_id", "rule_set",
      "rule_set_ip_cidr_match_source", "clash_mode", "network_type",
      "network_is_expensive", "network_is_constrained", "wifi_ssid", "wifi_bssid", "invert",
    ])
    expect(routeMatchFields).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "source_port", kind: "number-list" }),
      expect.objectContaining({ path: "user_id", kind: "number-list" }),
      expect.objectContaining({ path: "source_ip_is_private", kind: "boolean" }),
    ]))
  })

  it("models all route actions and their action-specific JSON types", () => {
    expect(routeActions).toEqual([
      "route", "route-options", "direct", "bypass", "reject", "hijack-dns", "sniff", "resolve",
    ])
    expect(paths(routeActionFields.route)).toContain("outbound")
    expect(paths(routeActionFields.bypass)).toContain("outbound")
    expect(paths(routeActionFields["route-options"])).not.toContain("outbound")
    expect(routeActionFields.direct).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "bind_interface" }),
      expect.objectContaining({ path: "domain_resolver.server" }),
      expect.objectContaining({ path: "network_type", kind: "list" }),
    ]))
    expect(routeActionFields.reject).toEqual([
      expect.objectContaining({ path: "method", kind: "select" }),
      expect.objectContaining({ path: "no_drop", kind: "boolean" }),
    ])
    expect(routeActionFields["hijack-dns"]).toEqual([])
    expect(ruleSetTypes).toEqual(["inline", "local", "remote"])
  })
})

describe("route type transitions", () => {
  it("removes logical-only fields when changing to a default rule", () => {
    expect(changeRouteRuleType({ type: "logical", mode: "and", rules: [], custom: "keep" }, "default"))
      .toEqual({ custom: "keep" })
  })

  it("retains fields shared by logical and default rules and removes incompatible fields", () => {
    expect(changeRouteRuleType({ domain: ["example.com"], invert: true, outbound: "proxy", custom: "keep" }, "logical"))
      .toEqual({ type: "logical", invert: true, outbound: "proxy", custom: "keep" })
  })

  it("preserves same and unknown current rule types", () => {
    const logical = { type: "logical", mode: "or", rules: [] }
    const custom = { type: "custom", domain: ["example.com"], custom: "keep" }
    expect(changeRouteRuleType(logical, "logical")).toBe(logical)
    expect(changeRouteRuleType(custom, "custom")).toBe(custom)
  })

  it("cleans known subtype paths when entering or leaving an unknown type", () => {
    expect(changeRouteRuleType({
      type: "custom", domain: ["example.com"], mode: "and", custom: "keep",
    }, "default")).toEqual({ domain: ["example.com"], custom: "keep" })
    expect(changeRouteRuleType({ domain: ["example.com"], custom: "keep" }, "custom"))
      .toEqual({ type: "custom", custom: "keep" })
  })
})

describe("route action transitions", () => {
  it("removes stale known fields and preserves unknown fields", () => {
    expect(changeRouteAction({ action: "reject", method: "drop", outbound: "old", custom: "keep" }, "route"))
      .toEqual({ action: "route", custom: "keep" })
  })

  it("retains compatible shared action options", () => {
    expect(changeRouteAction({
      action: "route", outbound: "proxy", override_address: "example.com", udp_connect: true,
      custom: "keep",
    }, "bypass")).toEqual({
      action: "bypass", outbound: "proxy", override_address: "example.com", udp_connect: true, custom: "keep",
    })
  })

  it("normalizes legacy outbound routing only after explicit route selection", () => {
    expect(changeRouteAction({ outbound: "proxy", override_port: 443, custom: "keep" }, "route"))
      .toEqual({ action: "route", outbound: "proxy", override_port: 443, custom: "keep" })
  })

  it("preserves same and unknown current actions", () => {
    const route = { action: "route", outbound: "proxy" }
    const custom = { action: "custom", payload: { enabled: true } }
    expect(changeRouteAction(route, "route")).toBe(route)
    expect(changeRouteAction(custom, "custom")).toBe(custom)
  })

  it("removes fields with incompatible JSON kinds across actions", () => {
    expect(changeRouteAction({ action: "route-options", fallback_delay: 250, custom: "keep" }, "direct"))
      .toEqual({ action: "direct", custom: "keep" })
  })

  it("cleans known action paths when entering or leaving an unknown action", () => {
    expect(changeRouteAction({ action: "custom", method: "drop", outbound: "old", payload: "keep" }, "reject"))
      .toEqual({ action: "reject", method: "drop", payload: "keep" })
    expect(changeRouteAction({ action: "route", outbound: "proxy", custom: "keep" }, "custom"))
      .toEqual({ action: "custom", custom: "keep" })
  })

  it("removes known nested direct fields without deleting unknown siblings", () => {
    expect(changeRouteAction({
      action: "direct",
      domain_resolver: { server: "dns-local", strategy: "prefer_ipv4", custom: "keep" },
    }, "reject")).toEqual({ action: "reject", domain_resolver: { custom: "keep" } })
  })
})

describe("route rule-set transitions", () => {
  it("removes remote-only fields when changing to local", () => {
    expect(changeRuleSetType({
      type: "remote", tag: "geo", url: "https://example/r.srs", update_interval: "1d", custom: "keep",
    }, "local")).toEqual({ type: "local", tag: "geo", custom: "keep" })
  })

  it("preserves shared format and removes known fields incompatible with the target", () => {
    expect(changeRuleSetType({ type: "local", tag: "geo", format: "binary", path: "/geo.srs" }, "remote"))
      .toEqual({ type: "remote", tag: "geo", format: "binary" })
    expect(changeRuleSetType({ type: "remote", tag: "geo", format: "binary", url: "https://example/r.srs" }, "inline"))
      .toEqual({ type: "inline", tag: "geo" })
  })

  it("preserves same and unknown current rule-set types", () => {
    const remote = { type: "remote", tag: "geo", url: "https://example/r.srs" }
    const custom = { type: "custom", tag: "geo", payload: { enabled: true } }
    expect(changeRuleSetType(remote, "remote")).toBe(remote)
    expect(changeRuleSetType(custom, "custom")).toBe(custom)
  })

  it("cleans known rule-set paths when entering or leaving an unknown type", () => {
    expect(changeRuleSetType({ type: "custom", path: "/keep.srs", url: "old", payload: "keep" }, "local"))
      .toEqual({ type: "local", path: "/keep.srs", payload: "keep" })
    expect(changeRuleSetType({ type: "remote", tag: "geo", url: "https://example/r.srs" }, "custom"))
      .toEqual({ type: "custom", tag: "geo" })
  })
})

describe("route arrays and summaries", () => {
  it("reads only object entries from route arrays", () => {
    expect(routeRules({ rules: [{ outbound: "proxy" }, null, "invalid"] })).toEqual([{ outbound: "proxy" }])
    expect(routeRuleSets({ rule_set: [{ type: "local", tag: "geo" }, 1] })).toEqual([{ type: "local", tag: "geo" }])
    expect(routeRules({ rules: {} })).toEqual([])
    expect(routeRuleSets({})).toEqual([])
  })

  it("sets route arrays immutably and copies readonly inputs", () => {
    const object = { final: "proxy", custom: "keep" }
    const rules = [{ outbound: "proxy" }] as const
    const ruleSets = [{ type: "local", tag: "geo" }] as const
    const withRules = setRouteRules(object, rules)
    const withRuleSets = setRouteRuleSets(object, ruleSets)
    expect(withRules).toEqual({ final: "proxy", custom: "keep", rules })
    expect(withRuleSets).toEqual({ final: "proxy", custom: "keep", rule_set: ruleSets })
    expect(withRules).not.toBe(object)
    expect(withRules.rules).not.toBe(rules)
    expect(withRuleSets.rule_set).not.toBe(ruleSets)
  })

  it("summarizes match values and legacy or explicit actions", () => {
    expect(summarizeRouteRule({ domain_suffix: ["example.com"], network: "tcp", outbound: "proxy" }))
      .toEqual({ matches: ["example.com", "tcp"], action: "proxy" })
    expect(summarizeRouteRule({ source_ip_is_private: true, port: [80, 443], action: "reject" }))
      .toEqual({ matches: ["source_ip_is_private", "80", "443"], action: "reject" })
    expect(summarizeRouteRule({ port: [80, null, false], action: "reject" }).matches).toEqual(["80"])
    expect(summarizeRouteRule({ rule_set_ip_cidr_match_source: true, action: "reject" }).matches)
      .toEqual(["rule_set_ip_cidr_match_source"])
    expect(summarizeRouteRule({ action: "route", outbound: "proxy" }).action).toBe("proxy")
    expect(summarizeRouteRule({ action: "custom" }).action).toBe("custom")
  })

  it("summarizes rule-set format and preferred location", () => {
    expect(summarizeRuleSet({
      type: "remote", tag: "geoip-cn", format: "binary", url: "https://example/geoip-cn.srs",
    })).toEqual({ type: "remote · binary", detail: "https://example/geoip-cn.srs" })
    expect(summarizeRuleSet({ tag: "inline", rules: [] })).toEqual({ type: "inline", detail: "inline" })
    expect(summarizeRuleSet({ type: "local", tag: "geo", path: "/etc/geo.srs" }))
      .toEqual({ type: "local", detail: "/etc/geo.srs" })
    expect(summarizeRuleSet({})).toEqual({ type: "inline", detail: "" })
  })
})
