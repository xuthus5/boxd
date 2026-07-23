import { describe, expect, it } from "vitest"

import { computeVirtualWindow } from "@/lib/virtual-window"

describe("computeVirtualWindow", () => {
  it("returns empty window for zero items", () => {
    expect(computeVirtualWindow({
      count: 0,
      scrollTop: 0,
      viewportHeight: 200,
      itemHeight: 40,
    })).toEqual({ startIndex: 0, endIndex: 0, offsetTop: 0, totalHeight: 0 })
  })

  it("windows the middle of a long list", () => {
    const window = computeVirtualWindow({
      count: 1000,
      scrollTop: 4000,
      viewportHeight: 320,
      itemHeight: 40,
      overscan: 2,
    })
    expect(window.totalHeight).toBe(40000)
    expect(window.startIndex).toBe(98)
    expect(window.endIndex).toBe(111)
    expect(window.offsetTop).toBe(98 * 40)
  })

  it("clamps to bounds at the end", () => {
    const window = computeVirtualWindow({
      count: 10,
      scrollTop: 1000,
      viewportHeight: 200,
      itemHeight: 40,
      overscan: 5,
    })
    expect(window.endIndex).toBe(10)
    expect(window.startIndex).toBeLessThanOrEqual(window.endIndex)
    expect(window.startIndex).toBeGreaterThanOrEqual(0)
    expect(window.totalHeight).toBe(400)
    expect(window.offsetTop).toBe(window.startIndex * 40)
  })

  it("guards invalid sizes", () => {
    const window = computeVirtualWindow({
      count: 5,
      scrollTop: -10,
      viewportHeight: -1,
      itemHeight: 0,
      overscan: -3,
    })
    expect(window.totalHeight).toBe(5)
    expect(window.startIndex).toBe(0)
    expect(window.endIndex).toBeGreaterThan(0)
  })
})
