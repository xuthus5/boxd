import { describe, expect, it } from "vitest"

import {
  diffConfig,
  formatConfigDiffSummary,
  previewJsonValue,
  summarizeConfigDiff,
} from "@/features/config/config-diff"

describe("config-diff", () => {
  it("diffs nested objects and arrays with values", () => {
    const before = { log: { level: "info" }, inbounds: [{ tag: "a" }], dns: { final: "local" } }
    const after = { log: { level: "debug" }, inbounds: [{ tag: "a" }, { tag: "b" }], experimental: { clash_api: {} } }
    const items = diffConfig(before, after)
    expect(items).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "log.level", kind: "changed", before: "info", after: "debug" }),
      expect.objectContaining({ path: "inbounds", kind: "changed" }),
      expect.objectContaining({ path: "dns", kind: "removed", before: { final: "local" } }),
      expect.objectContaining({ path: "experimental", kind: "added", after: { clash_api: {} } }),
    ]))
    expect(summarizeConfigDiff(items).total).toBe(items.length)
  })

  it("formats a compact summary", () => {
    const items = diffConfig({ a: 1 }, { a: 2, b: 3 })
    const text = formatConfigDiffSummary(items, {
      added: "added",
      removed: "removed",
      changed: "changed",
      none: "none",
      more: "+{{count}} more",
    })
    expect(text).toContain("changed")
    expect(text).toContain("~a")
    expect(formatConfigDiffSummary([], {
      added: "added", removed: "removed", changed: "changed", none: "none", more: "+{{count}} more",
    })).toBe("none")
  })

  it("previews json values compactly", () => {
    expect(previewJsonValue(undefined)).toBe("—")
    expect(previewJsonValue(null)).toBe("null")
    expect(previewJsonValue(true)).toBe("true")
    expect(previewJsonValue("hello")).toBe("\"hello\"")
    expect(previewJsonValue({ a: 1, b: 2, c: 3, d: 4, e: 5, f: 6 })).toMatch(/\{/)
    expect(previewJsonValue(Array.from({ length: 20 }, (_, i) => i))).toMatch(/\[/)
  })
})
