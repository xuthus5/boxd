import { describe, expect, it } from "vitest"

import {
  configApplySourceHref,
  configApplySourceKey,
  shortConfigHash,
} from "@/features/dashboard/config-apply-source"

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

describe("configApplySourceHref", () => {
  it("maps sources to ops pages", () => {
    expect(configApplySourceHref("raw")).toBe("/advanced/raw")
    expect(configApplySourceHref("dns_defaults")).toBe("/policy/dns")
    expect(configApplySourceHref("outbounds_defaults")).toBe("/proxy/outbounds")
    expect(configApplySourceHref("inbounds_defaults")).toBe("/proxy/inbounds")
    expect(configApplySourceHref("route_defaults")).toBe("/policy/route")
    expect(configApplySourceHref("unknown")).toBe("/advanced/raw")
  })
})

describe("shortConfigHash", () => {
  it("shortens hashes and handles empty", () => {
    expect(shortConfigHash("abcdef012345")).toBe("abcdef01")
    expect(shortConfigHash("ab", 8)).toBe("ab")
    expect(shortConfigHash("  ")).toBe("—")
  })
})
