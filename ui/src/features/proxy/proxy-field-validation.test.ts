import { describe, expect, it } from "vitest"

import {
  isNumericFieldKind,
  isNumericFieldRawValid,
  parseFiniteNumber,
  parseFiniteNumberList,
} from "@/features/proxy/proxy-field-validation"

describe("proxy-field-validation", () => {
  it("parses finite numbers", () => {
    expect(parseFiniteNumber("")).toBeUndefined()
    expect(parseFiniteNumber("  ")).toBeUndefined()
    expect(parseFiniteNumber("12")).toBe(12)
    expect(parseFiniteNumber("12.5")).toBe(12.5)
    expect(parseFiniteNumber("abc")).toBeNull()
    expect(parseFiniteNumber("Infinity")).toBeNull()
  })

  it("parses number lists", () => {
    expect(parseFiniteNumberList("")).toBeUndefined()
    expect(parseFiniteNumberList("1, 2\n3")).toEqual([1, 2, 3])
    expect(parseFiniteNumberList("1,x")).toBeNull()
  })

  it("checks kind validity", () => {
    expect(isNumericFieldKind("number")).toBe(true)
    expect(isNumericFieldKind("number-list")).toBe(true)
    expect(isNumericFieldKind("text")).toBe(false)
    expect(isNumericFieldRawValid("number", "bad")).toBe(false)
    expect(isNumericFieldRawValid("number", "1")).toBe(true)
    expect(isNumericFieldRawValid("number-list", "1,2")).toBe(true)
    expect(isNumericFieldRawValid("text", "anything")).toBe(true)
  })
})
