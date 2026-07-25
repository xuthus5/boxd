import { describe, expect, it } from "vitest"

import { parsePolicyItemPath, policyDialogSelectionFromPath } from "@/features/policy/policy-path"

describe("parsePolicyItemPath", () => {
  it("parses route rules and rule sets", () => {
    expect(parsePolicyItemPath("route.rules[0].outbound", "route")).toEqual({
      section: "route",
      kind: "rules",
      index: 0,
      relativePath: "outbound",
    })
    expect(parsePolicyItemPath("rule_set[2].url", "route")).toEqual({
      section: "route",
      kind: "rule_set",
      index: 2,
      relativePath: "url",
    })
    expect(parsePolicyItemPath("route.rule_set[1]", "route")).toEqual({
      section: "route",
      kind: "rule_set",
      index: 1,
      relativePath: "",
    })
  })

  it("parses dns servers and rules", () => {
    expect(parsePolicyItemPath("dns.servers[1].tag", "dns")).toEqual({
      section: "dns",
      kind: "servers",
      index: 1,
      relativePath: "tag",
    })
    expect(parsePolicyItemPath("rules[0].server", "dns")).toEqual({
      section: "dns",
      kind: "rules",
      index: 0,
      relativePath: "server",
    })
  })

  it("rejects mismatched sections and non-item paths", () => {
    expect(parsePolicyItemPath("route.final", "route")).toBeNull()
    expect(parsePolicyItemPath("dns.rules[0].server", "route")).toBeNull()
    expect(parsePolicyItemPath("servers[0].tag", "route")).toBeNull()
    expect(parsePolicyItemPath("", "dns")).toBeNull()
  })
})

describe("policyDialogSelectionFromPath", () => {
  it("opens in-range route rule selection", () => {
    expect(policyDialogSelectionFromPath("route.rules[0].outbound", "route", {
      rules: [{ action: "route", outbound: "proxy" }],
      metadata: [{ name: "n", description: "" }],
    })).toEqual({
      kind: "rule",
      index: 0,
      item: { action: "route", outbound: "proxy" },
      metadata: { name: "n", description: "" },
      jumpPath: "outbound",
    })
  })

  it("returns null for out-of-range indexes", () => {
    expect(policyDialogSelectionFromPath("dns.servers[2].tag", "dns", {
      servers: [{ tag: "a" }],
    })).toBeNull()
  })

  it("opens rule-set and server selections without optional metadata", () => {
    expect(policyDialogSelectionFromPath("route.rule_set[0]", "route", {
      ruleSets: [{ tag: "geo" }],
    })).toEqual({ kind: "rule-set", index: 0, item: { tag: "geo" }, jumpPath: "" })
    expect(policyDialogSelectionFromPath("rules[0]", "dns", {
      rules: [{ action: "reject" }],
    })).toEqual({ kind: "rule", index: 0, item: { action: "reject" }, metadata: undefined, jumpPath: "" })
    expect(policyDialogSelectionFromPath("dns.servers[0]", "dns", {
      servers: [{ tag: "local" }],
    })).toEqual({ kind: "server", index: 0, item: { tag: "local" }, jumpPath: "" })
  })

  it("returns null when the selected list is unavailable", () => {
    expect(policyDialogSelectionFromPath("route.rule_set[0].url", "route", {})).toBeNull()
    expect(policyDialogSelectionFromPath("route.rules[0].action", "route", {})).toBeNull()
    expect(policyDialogSelectionFromPath("dns.servers[0].tag", "dns", {})).toBeNull()
  })
})
