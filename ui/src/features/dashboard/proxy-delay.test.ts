import { describe, expect, it } from "vitest"

import {
  formatDelayValue,
  pickPrimaryGroup,
  sortDelayEntries,
  summarizeDelays,
} from "@/features/dashboard/proxy-delay"

describe("proxy-delay helpers", () => {
  it("picks preferred selector groups", () => {
    expect(pickPrimaryGroup([])).toBeNull()
    expect(pickPrimaryGroup([
      { tag: "other", type: "selector", now: "a", all: ["a"] },
      { tag: "proxy", type: "selector", now: "b", all: ["b"] },
    ])?.tag).toBe("proxy")
  })

  it("formats and sorts delays with failures last", () => {
    expect(formatDelayValue(12, "timeout")).toBe("12 ms")
    expect(formatDelayValue("error", "timeout")).toBe("timeout")
    expect(formatDelayValue(undefined, "timeout")).toBe("—")
    expect(sortDelayEntries({ b: 40, a: 12, c: "error" }).map(([tag]) => tag)).toEqual(["a", "b", "c"])
    expect(summarizeDelays({ a: 1, b: "error", c: 3 })).toEqual({ total: 3, ok: 2, failed: 1 })
  })
})
