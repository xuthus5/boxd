import { describe, expect, it } from "vitest"

import {
  applyLogPreset,
  logPresetById,
  LOG_FILTER_PRESETS,
  matchesLogFilter,
} from "@/features/observability/log-filter-presets"

describe("log-filter-presets", () => {
  it("exposes named presets and applies them", () => {
    expect(LOG_FILTER_PRESETS.length).toBeGreaterThanOrEqual(4)
    expect(logPresetById("dns")?.query).toBe("dns")
    expect(applyLogPreset(logPresetById("errors"))).toEqual({ filter: "error", minimum: "error" })
    expect(applyLogPreset(undefined)).toEqual({ filter: "" })
  })

  it("matches single and multi-token queries with OR semantics", () => {
    expect(matchesLogFilter("info", "inbound connection from 1.1.1.1", "inbound")).toBe(true)
    expect(matchesLogFilter("info", "outbound/vless", "inbound outbound connection")).toBe(true)
    expect(matchesLogFilter("info", "dns query", "tls reality")).toBe(false)
    expect(matchesLogFilter("error", "failed", "")).toBe(true)
  })
})
