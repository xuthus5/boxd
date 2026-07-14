import { describe, expect, it } from "vitest"

import {
  moveConfigArrayItem,
  removeConfigArrayItem,
  replaceConfigArrayItem,
  replaceConfigSection,
} from "@/features/config/config-update"
import type { SingBoxConfig } from "@/lib/api/types"

const config: SingBoxConfig = {
  log: { level: "info" },
  inbounds: [{ type: "mixed", tag: "mixed-in", sniff: true }],
}

describe("config updates", () => {
  it("replaces a section without mutating unrelated data", () => {
    const next = replaceConfigSection(config, "dns", { servers: [] })
    expect(next).toEqual({ ...config, dns: { servers: [] } })
    expect(next).not.toBe(config)
    expect(next.log).toBe(config.log)
  })

  it("replaces an array item and preserves unknown fields", () => {
    const next = replaceConfigArrayItem(config, { key: "inbounds", index: 0, patch: { tag: "renamed" } })
    expect(next.inbounds).toEqual([{ type: "mixed", tag: "renamed", sniff: true }])
  })

  it("removes and moves array items", () => {
    const source: SingBoxConfig = { outbounds: [{ tag: "a" }, { tag: "b" }, { tag: "c" }] }
    expect(removeConfigArrayItem(source, "outbounds", 1).outbounds).toEqual([{ tag: "a" }, { tag: "c" }])
    expect(moveConfigArrayItem(source, { key: "outbounds", from: 2, to: 0 }).outbounds).toEqual([
      { tag: "c" }, { tag: "a" }, { tag: "b" },
    ])
  })

  it("handles absent arrays, scalar items, and invalid move indexes", () => {
    const empty: SingBoxConfig = { inbounds: "invalid" }
    expect(removeConfigArrayItem(empty, "inbounds", 0).inbounds).toEqual([])
    expect(replaceConfigArrayItem({ inbounds: ["invalid"] }, { key: "inbounds", index: 0, patch: { tag: "fixed" } }).inbounds)
      .toEqual([{ tag: "fixed" }])
    expect(moveConfigArrayItem({ inbounds: [] }, { key: "inbounds", from: 4, to: 0 }).inbounds).toEqual([])
  })
})
