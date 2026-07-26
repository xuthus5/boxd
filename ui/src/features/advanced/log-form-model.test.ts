import { describe, expect, it } from "vitest"

import {
  applyLogConfig,
  isLogStructureValid,
  logFields,
  logLevels,
  normalizeLogObject,
  prepareLogObject,
} from "@/features/advanced/log-form-model"

describe("log form model", () => {
  it("exposes every sing-box 1.13 log level and field", () => {
    expect(logLevels).toEqual(["trace", "debug", "info", "warn", "error", "fatal", "panic"])
    expect(logFields.map((field) => field.path)).toEqual(["disabled", "level", "output", "timestamp"])
  })

  it("accepts valid log objects and rejects malformed known fields", () => {
    expect(isLogStructureValid({ disabled: true, level: "warn", output: "box.log", timestamp: true })).toBe(true)
    expect(isLogStructureValid({ future_option: { enabled: true } })).toBe(true)
    expect(isLogStructureValid([])).toBe(false)
    expect(isLogStructureValid({ disabled: "yes" })).toBe(false)
    expect(isLogStructureValid({ level: "verbose" })).toBe(false)
    expect(isLogStructureValid({ output: 42 })).toBe(false)
    expect(isLogStructureValid({ timestamp: 1 })).toBe(false)
  })

  it("normalizes missing sections without inventing defaults", () => {
    expect(normalizeLogObject(undefined)).toEqual({})
    expect(normalizeLogObject(null)).toEqual({})
    expect(normalizeLogObject({ level: "info" })).toEqual({ level: "info" })
  })

  it("removes default and blank values while preserving unknown fields", () => {
    expect(prepareLogObject({
      disabled: false,
      level: " warn ",
      output: " /var/log/sing-box.log ",
      timestamp: false,
      future_option: { enabled: true },
    })).toEqual({
      level: "warn",
      output: "/var/log/sing-box.log",
      future_option: { enabled: true },
    })
    expect(prepareLogObject({ level: " ", output: " ", timestamp: true })).toEqual({ timestamp: true })
  })

  it("applies or removes the log section immutably", () => {
    const config = { log: { level: "info" }, outbounds: [] }
    expect(applyLogConfig(config, {})).toEqual({ outbounds: [] })
    expect(config).toHaveProperty("log.level", "info")
    expect(applyLogConfig(config, { disabled: true, level: "error" })).toEqual({
      log: { disabled: true, level: "error" },
      outbounds: [],
    })
  })
})
