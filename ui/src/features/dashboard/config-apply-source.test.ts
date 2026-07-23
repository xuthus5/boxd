import { describe, expect, it } from "vitest"

import { configApplySourceKey, shortConfigHash } from "@/features/dashboard/config-apply-source"

describe("configApplySourceKey", () => {
  it("maps known sources", () => {
    expect(configApplySourceKey("update")).toBe("sourceUpdate")
    expect(configApplySourceKey("raw")).toBe("sourceRaw")
    expect(configApplySourceKey("dns_defaults")).toBe("sourceDNSDefaults")
  })

  it("falls back for unknown sources", () => {
    expect(configApplySourceKey("sync")).toBe("sourceUnknown")
    expect(configApplySourceKey("")).toBe("sourceUnknown")
  })
})

describe("shortConfigHash", () => {
  it("shortens hashes and handles empty", () => {
    expect(shortConfigHash("abcdef012345")).toBe("abcdef01")
    expect(shortConfigHash("ab", 8)).toBe("ab")
    expect(shortConfigHash("  ")).toBe("—")
  })
})
