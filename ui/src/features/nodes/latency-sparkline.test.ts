import { describe, expect, it } from "vitest"

import { buildSparklinePath } from "@/features/nodes/latency-sparkline-model"

describe("buildSparklinePath", () => {
  it("returns null for insufficient points", () => {
    expect(buildSparklinePath([])).toBeNull()
    expect(buildSparklinePath([{ timestamp: "t", success: true, latency_ms: 10 }])).toBeNull()
  })

  it("builds path for successful samples and skips failures", () => {
    const path = buildSparklinePath([
      { timestamp: "1", success: true, latency_ms: 10 },
      { timestamp: "2", success: false, error: "x" },
      { timestamp: "3", success: true, latency_ms: 30 },
      { timestamp: "4", success: true, latency_ms: 20 },
    ])
    expect(path).toMatch(/^M/)
    expect(path?.split(" ").length).toBeGreaterThan(2)
  })
})
