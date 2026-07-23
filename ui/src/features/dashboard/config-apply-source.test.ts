import { describe, expect, it } from "vitest"

import {
  configApplyErrorClipboardText,
  configApplyErrorPath,
  configApplyErrorSectionHref,
  configApplyErrorSectionOnlyHref,
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

describe("configApplyErrorClipboardText", () => {
  it("formats diagnostic payload for failed applies", () => {
    expect(configApplyErrorClipboardText({
      source: "raw",
      status: "rolled_back",
      hash: "abcdef01",
      size: 2048,
      error: "restart failed",
      applied_at: "2026-07-23T12:00:00Z",
    })).toBe([
      "source: raw",
      "status: rolled_back",
      "hash: abcdef01",
      "size: 2048",
      "code: restart_failed",
      "error: restart failed",
      "at: 2026-07-23T12:00:00Z",
    ].join("\n"))
  })

  it("includes path when validation message is pathful", () => {
    const payload = configApplyErrorClipboardText({
      source: "raw",
      status: "rolled_back",
      hash: "abcdef",
      size: 10,
      error: "inbounds[0].listen_port: invalid",
      applied_at: "2026-07-23T12:00:00Z",
    })
    expect(payload).toContain("path: inbounds[0].listen_port")
    expect(payload).toContain("code: config_invalid")
  })
})

describe("config apply path linkage", () => {
  it("routes pathful errors to the matching section", () => {
    expect(configApplyErrorPath({ error: "dns.servers[0].address: missing" })).toBe("dns.servers[0].address")
    expect(configApplyErrorSectionHref({
      error: "dns.servers[0].address: missing",
      source: "raw",
    })).toBe("/advanced/raw?path=dns.servers%5B0%5D.address")
    expect(configApplyErrorSectionHref({
      error: "restart failed after config save",
      source: "raw",
    })).toBe("/advanced/raw")
    expect(configApplyErrorSectionHref({
      error: "inbounds[0].listen_port: invalid",
      source: "update",
    })).toBe("/advanced/raw?path=inbounds%5B0%5D.listen_port")
    expect(configApplyErrorSectionOnlyHref({
      error: "inbounds[0].listen_port: invalid",
      source: "raw",
    })).toBe("/proxy/inbounds")
  })
})
