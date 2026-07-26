import { describe, expect, it } from "vitest"

import {
  changeHeadlessRuleType,
  headlessRuleMatchFields,
  headlessRules,
  isHeadlessRuleComplete,
  setHeadlessRules,
  summarizeHeadlessRule,
} from "@/features/policy/route-headless-rule-model"

const paths = (fields: readonly { path: string }[]) => fields.map((field) => field.path)

describe("route headless rule metadata", () => {
  it("models every sing-box 1.13 headless match field", () => {
    expect(paths(headlessRuleMatchFields)).toEqual([
      "type", "query_type", "network", "domain", "domain_suffix", "domain_keyword", "domain_regex",
      "source_ip_cidr", "ip_cidr", "source_port", "source_port_range", "port", "port_range",
      "process_name", "process_path", "process_path_regex", "package_name", "network_type",
      "network_is_expensive", "network_is_constrained", "network_interface_address",
      "default_interface_address", "wifi_ssid", "wifi_bssid", "invert",
    ])
    expect(headlessRuleMatchFields).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "query_type", kind: "list" }),
      expect.objectContaining({ path: "source_port", kind: "number-list" }),
      expect.objectContaining({ path: "network", kind: "network-multi" }),
      expect.objectContaining({ path: "network_interface_address", kind: "json-object" }),
      expect.objectContaining({ path: "default_interface_address", kind: "list" }),
    ]))
  })

  it("cleans incompatible fields when changing rule type", () => {
    expect(changeHeadlessRuleType({
      domain_suffix: ["example.com"], network_interface_address: { wifi: ["192.0.2.0/24"] },
      invert: true, custom: "keep",
    }, "logical")).toEqual({ type: "logical", invert: true, custom: "keep" })
    expect(changeHeadlessRuleType({
      type: "logical", mode: "or", rules: [{ domain: ["example.com"] }], invert: true, custom: "keep",
    }, "default")).toEqual({ invert: true, custom: "keep" })
    expect(changeHeadlessRuleType({ type: "custom", query_type: [1, "AAAA"], payload: "keep" }, "default"))
      .toEqual({ query_type: [1, "AAAA"], payload: "keep" })
  })

  it("preserves same and unknown rule types", () => {
    const logical = { type: "logical", mode: "or", rules: [{ domain: ["example.com"] }] }
    const custom = { type: "custom", payload: { enabled: true } }
    expect(changeHeadlessRuleType(logical, "logical")).toBe(logical)
    expect(changeHeadlessRuleType(custom, "custom")).toBe(custom)
  })

  it("preserves compatible scalar, list, numeric, and object values", () => {
    expect(changeHeadlessRuleType({
      type: "custom",
      query_type: [1, "AAAA"], network: ["tcp", "udp"], domain: "example.com",
      source_port: [53, 443], port: 8443, network_is_expensive: false,
      network_interface_address: { wifi: ["192.0.2.0/24"] },
      default_interface_address: ["192.0.2.1"], invert: false, payload: "keep",
    }, "default")).toEqual({
      query_type: [1, "AAAA"], network: ["tcp", "udp"], domain: "example.com",
      source_port: [53, 443], port: 8443, network_is_expensive: false,
      network_interface_address: { wifi: ["192.0.2.0/24"] },
      default_interface_address: ["192.0.2.1"], invert: false, payload: "keep",
    })
    expect(changeHeadlessRuleType({
      type: "custom", mode: "and", rules: [{ domain: ["example.com"] }], invert: true,
      payload: "keep",
    }, "logical")).toEqual({
      type: "logical", mode: "and", rules: [{ domain: ["example.com"] }], invert: true,
      payload: "keep",
    })
  })

  it("drops incompatible values and handles unknown targets", () => {
    expect(changeHeadlessRuleType({
      type: "custom", query_type: { invalid: true }, network: ["tcp", 1], domain: [1],
      source_port: [53, "bad"], port: Number.NaN, network_is_expensive: "yes",
      network_interface_address: ["bad"], default_interface_address: [1],
      invert: "yes", payload: "keep",
    }, "default")).toEqual({ payload: "keep" })
    expect(changeHeadlessRuleType({
      type: "custom", mode: {}, rules: { invalid: true }, invert: "yes", payload: "keep",
    }, "logical")).toEqual({ type: "logical", payload: "keep" })
    expect(changeHeadlessRuleType({ domain: ["example.com"], payload: "keep" }, "future"))
      .toEqual({ type: "future", payload: "keep" })
  })
})

describe("route headless rule state", () => {
  it("reads and sets inline rules immutably", () => {
    expect(headlessRules({ rules: [{ domain: ["example.com"] }, null, "bad"] })).toEqual([
      { domain: ["example.com"] },
    ])
    expect(headlessRules({ rules: {} })).toEqual([])
    const ruleSet = { type: "inline", tag: "inline", custom: "keep" }
    const rules = [{ port: [443] }] as const
    const next = setHeadlessRules(ruleSet, rules)
    expect(next).toEqual({ ...ruleSet, rules })
    expect(next).not.toBe(ruleSet)
    expect(next.rules).not.toBe(rules)
  })

  it("validates default, logical, and unknown rule shapes", () => {
    expect(isHeadlessRuleComplete({})).toBe(false)
    expect(isHeadlessRuleComplete({ invert: true })).toBe(false)
    expect(isHeadlessRuleComplete({ domain_suffix: ["example.com"] })).toBe(true)
    expect(isHeadlessRuleComplete({ type: "logical", mode: "or", rules: [] })).toBe(false)
    expect(isHeadlessRuleComplete({ type: "logical", mode: "or", rules: [{ domain: ["example.com"] }] })).toBe(true)
    expect(isHeadlessRuleComplete({ type: "custom", payload: true })).toBe(true)
  })

  it("rejects incomplete logical rules and empty default values", () => {
    expect(isHeadlessRuleComplete({ type: "logical", mode: 1, rules: [{}] })).toBe(false)
    expect(isHeadlessRuleComplete({ type: "logical", mode: "", rules: [{}] })).toBe(false)
    expect(isHeadlessRuleComplete({ type: "logical", mode: "and", rules: {} })).toBe(false)
    expect(isHeadlessRuleComplete({ type: "logical", mode: "and", rules: ["bad"] })).toBe(false)
    expect(isHeadlessRuleComplete({ domain: [] })).toBe(false)
    expect(isHeadlessRuleComplete({ network_interface_address: {} })).toBe(false)
    expect(isHeadlessRuleComplete({ domain: "" })).toBe(false)
    expect(isHeadlessRuleComplete({ port: Number.NaN })).toBe(false)
    expect(isHeadlessRuleComplete({ network_is_expensive: false })).toBe(false)
    expect(isHeadlessRuleComplete({ port: 0 })).toBe(true)
    expect(isHeadlessRuleComplete({ network_is_constrained: true })).toBe(true)
  })

  it("summarizes default and logical inline rules", () => {
    expect(summarizeHeadlessRule({
      domain_suffix: ["example.com"], network_is_expensive: true,
      network_interface_address: { wifi: ["192.0.2.0/24"] },
    }, { matchLabel: (path) => `label:${path}` })).toEqual({
      type: "default",
      matches: ["example.com", "label:network_is_expensive", "label:network_interface_address"],
      childRules: 0,
    })
    expect(summarizeHeadlessRule({
      type: "logical", mode: "and", rules: [{ domain: ["example.com"] }, null],
    })).toEqual({ type: "logical", matches: ["and"], childRules: 1 })
  })

  it("summarizes primitive, empty, and invalid match values", () => {
    expect(summarizeHeadlessRule({
      domain: ["a.example", 443, true, {}], domain_suffix: "example.com", port: 8443,
      network_interface_address: {}, network_is_expensive: false, network_is_constrained: true,
    }, { matchLabel: (path) => `label:${path}` })).toEqual({
      type: "default",
      matches: ["a.example", "443", "example.com", "8443", "label:network_is_constrained"],
      childRules: 0,
    })
    expect(summarizeHeadlessRule({ type: "logical", mode: "", rules: {} }))
      .toEqual({ type: "logical", matches: [], childRules: 0 })
    expect(summarizeHeadlessRule({ type: "custom", payload: true }))
      .toEqual({ type: "custom", matches: [], childRules: 0 })
  })
})
