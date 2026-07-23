import { describe, expect, it } from "vitest"

import { isRuleInverted, toggleRuleInvert } from "@/features/policy/rule-invert"

describe("rule-invert", () => {
  it("toggles invert flag without dropping other fields", () => {
    const base = { action: "route", outbound: "proxy", domain: ["a.com"] }
    expect(isRuleInverted(base)).toBe(false)
    const inverted = toggleRuleInvert(base)
    expect(inverted).toEqual({ ...base, invert: true })
    expect(isRuleInverted(inverted)).toBe(true)
    expect(toggleRuleInvert(inverted)).toEqual(base)
  })
})
