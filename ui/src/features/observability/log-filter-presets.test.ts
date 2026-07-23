import { describe, expect, it } from "vitest"

import {
  applyLogPreset,
  buildLogsHref,
  toLogSearchParams,
  logPresetById,
  LOG_FILTER_PRESETS,
  matchesLogFilter,
  parseLogSearchParams,
  resolveLogSeed,
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

  it("parses and builds log deep-link query strings", () => {
    expect(parseLogSearchParams(new URLSearchParams("tab=application&preset=errors&q=dns"))).toEqual({
      tab: "application",
      query: "dns",
      minimum: undefined,
      preset: "errors",
    })
    expect(buildLogsHref({ preset: "errors" })).toBe("/observability/logs?preset=errors")
    expect(buildLogsHref({ tab: "kernel", minimum: "all" })).toBe("/observability/logs?minimum=all")
    expect(buildLogsHref({ tab: "application", query: "dns", minimum: "warn" })).toBe(
      "/observability/logs?tab=application&q=dns&minimum=warn",
    )
    expect(resolveLogSeed({ preset: "errors" })).toEqual({
      filter: "error",
      minimum: "error",
      preset: "errors",
    })
    expect(toLogSearchParams({ preset: "dns" }).get("preset")).toBe("dns")
  })
})
