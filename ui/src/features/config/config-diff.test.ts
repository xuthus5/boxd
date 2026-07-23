import { describe, expect, it } from "vitest"

import { diffConfig, formatConfigDiffSummary, summarizeConfigDiff } from "@/features/config/config-diff"

describe("config-diff", () => {
  it("diffs nested objects and arrays", () => {
    const before = { log: { level: "info" }, inbounds: [{ tag: "a" }], dns: { final: "local" } }
    const after = { log: { level: "debug" }, inbounds: [{ tag: "a" }, { tag: "b" }], experimental: { clash_api: {} } }
    const items = diffConfig(before, after)
    expect(items).toEqual(expect.arrayContaining([
      { path: "log.level", kind: "changed" },
      { path: "inbounds", kind: "changed" },
      { path: "dns", kind: "removed" },
      { path: "experimental", kind: "added" },
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
})
