import { describe, expect, it } from "vitest"

import {
  applyLogPreset,
  buildLogsHref,
  toLogSearchParams,
  logPresetById,
  LOG_FILTER_PRESETS,
  matchesLogFilter,
  matchesLogLevel,
  parseLogSearchParams,
  resolveLogSeed,
  summarizeLogLevels,
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

  it("summarizes log levels and matches exact level filters", () => {
    const items = [
      { level: "error", message: "failed dns" },
      { level: "info", message: "dns query" },
      { level: "info", message: "connection opened" },
      { level: "WARN", message: "tls handshake" },
    ]
    expect(summarizeLogLevels(items)).toEqual({
      total: 4,
      buckets: [
        { level: "error", count: 1 },
        { level: "warn", count: 1 },
        { level: "info", count: 2 },
      ],
    })
    expect(summarizeLogLevels(items, "dns").buckets.map((bucket) => bucket.level)).toEqual(["error", "info"])
    expect(matchesLogLevel("ERROR", "error")).toBe(true)
    expect(matchesLogLevel("info", "error")).toBe(false)
  })

  it("parses and builds log deep-link query strings", () => {
    expect(parseLogSearchParams(new URLSearchParams("tab=application&preset=errors&q=dns&level=error"))).toEqual({
      tab: "application",
      query: "dns",
      minimum: undefined,
      level: "error",
      preset: "errors",
    })
    expect(buildLogsHref({ preset: "errors" })).toBe("/observability/logs?preset=errors")
    expect(buildLogsHref({ tab: "kernel", minimum: "all" })).toBe("/observability/logs?minimum=all")
    expect(buildLogsHref({ tab: "application", query: "dns", minimum: "warn", level: "error" })).toBe(
      "/observability/logs?tab=application&q=dns&minimum=warn&level=error",
    )
    expect(resolveLogSeed({ preset: "errors" })).toEqual({
      filter: "error",
      minimum: "error",
      level: undefined,
      preset: "errors",
    })
    expect(toLogSearchParams({ preset: "dns", level: "info" }).get("level")).toBe("info")
  })
})
