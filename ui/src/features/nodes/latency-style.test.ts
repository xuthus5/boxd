import { describe, expect, it } from "vitest"

import { latencyBadgeVariant, latencyTone, latencyToneClass } from "@/features/nodes/latency-style"

describe("latency-style", () => {
  it("maps latency ranges to tones", () => {
    expect(latencyTone(40)).toBe("excellent")
    expect(latencyTone(120)).toBe("good")
    expect(latencyTone(220)).toBe("fair")
    expect(latencyTone(400)).toBe("poor")
    expect(latencyTone(10, false)).toBe("failed")
    expect(latencyTone(undefined)).toBe("unknown")
  })

  it("picks badge variants and classes", () => {
    expect(latencyBadgeVariant("excellent")).toBe("default")
    expect(latencyBadgeVariant("failed")).toBe("destructive")
    expect(latencyToneClass("excellent")).toContain("emerald")
    expect(latencyToneClass("failed")).toBe("")
  })
})
