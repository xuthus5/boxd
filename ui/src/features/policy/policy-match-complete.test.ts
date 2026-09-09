import { describe, expect, it } from "vitest"

import { hasRuleMatchConditions } from "@/features/policy/policy-match-complete"

describe("nested rule match requirements", () => {
  it("does not count type, inversion, or empty values as a match", () => {
    for (const rule of [{}, { type: "default" }, { invert: true }, { domain: [] }, { domain: "" },
      { ip_is_private: false }, { interface_address: {} }, { source_hostname: null }, { port: Number.NaN }]) {
      expect(hasRuleMatchConditions(rule)).toBe(false)
    }
  })

  it("recognizes address maps, scalar ports, domains, and boolean conditions", () => {
    for (const rule of [{ interface_address: { eth0: ["192.0.2.0/24"] } }, { port: 443 },
      { domain: "example.org" }, { source_hostname: ["laptop"] }, { ip_is_private: true }]) {
      expect(hasRuleMatchConditions(rule)).toBe(true)
    }
  })
})
