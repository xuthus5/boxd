import { describe, expect, it } from "vitest"

import {
  buildRuleSetHealth,
  parseAutoUpdateLog,
  parseRuleSetDuration,
  ruleSetHealthHref,
} from "@/features/dashboard/rule-set-health"
import type { LogEvent, RuleSetAutoUpdate, RuleSetStatusItem } from "@/lib/api/types"

const now = new Date("2026-07-26T12:00:00Z")
const autoUpdate: RuleSetAutoUpdate = { enabled: true, interval: "24h" }

function item(overrides: Partial<RuleSetStatusItem> = {}): RuleSetStatusItem {
  return {
    tag: "geo-cn",
    type: "remote",
    builtin: true,
    updatable: true,
    update_interval: "24h",
    file_size: 1024,
    last_updated: "2026-07-26T08:00:00Z",
    ...overrides,
  }
}

function log(message: string, timestamp = "2026-07-26T11:00:00Z"): LogEvent {
  return { level: "info", message, timestamp }
}

describe("rule-set health helpers", () => {
  it("parses supported and compound durations while rejecting invalid values", () => {
    expect(parseRuleSetDuration("24h")).toBe(24 * 60 * 60 * 1000)
    expect(parseRuleSetDuration("1h30m")).toBe(90 * 60 * 1000)
    expect(parseRuleSetDuration("500ms")).toBe(500)
    expect(parseRuleSetDuration("0s")).toBeUndefined()
    expect(parseRuleSetDuration("-1h")).toBeUndefined()
    expect(parseRuleSetDuration("forever")).toBeUndefined()
    expect(parseRuleSetDuration()).toBeUndefined()
  })

  it("parses the structured auto-update completion log", () => {
    expect(parseAutoUpdateLog(log("INFO ruleset auto update finished updated=2 failed=1 skipped=0"))).toEqual({
      updated: 2,
      failed: 1,
      skipped: 0,
      timestamp: "2026-07-26T11:00:00Z",
    })
    expect(parseAutoUpdateLog(log("ruleset auto update finished skipped=3 updated=0 failed=0"))).toEqual({
      updated: 0,
      failed: 0,
      skipped: 3,
      timestamp: "2026-07-26T11:00:00Z",
    })
    expect(parseAutoUpdateLog(log("WARN ruleset auto update failed err=read config: permission denied"))).toEqual({
      updated: 0,
      failed: 1,
      skipped: 0,
      error: "read config: permission denied",
      timestamp: "2026-07-26T11:00:00Z",
    })
    expect(parseAutoUpdateLog(log("ruleset auto update failed"))).toMatchObject({ error: "failed" })
    expect(parseAutoUpdateLog(log("ruleset auto update finished updated=1 failed=0"))).toBeUndefined()
    expect(parseAutoUpdateLog(log("unrelated message"))).toBeUndefined()
    expect(parseAutoUpdateLog({ level: "info", message: undefined } as never)).toBeUndefined()
  })

  it("classifies healthy, missing, stale, and unmanaged entries", () => {
    const health = buildRuleSetHealth([
      item(),
      item({ tag: "missing", file_size: 0, last_updated: undefined }),
      item({ tag: "stale", last_updated: "2026-07-24T00:00:00Z" }),
      item({ tag: "custom-local", type: "local", builtin: false, updatable: false, file_size: 42 }),
    ], autoUpdate, [], now)

    expect(health.tone).toBe("warning")
    expect(health.total).toBe(4)
    expect(health.updatable).toBe(3)
    expect(health.available).toBe(3)
    expect(health.missing).toBe(1)
    expect(health.stale).toBe(1)
    expect(health.unmanaged).toBe(1)
    expect(health.latestUpdatedAt).toBe("2026-07-26T08:00:00Z")
    expect(health.items.map((entry) => entry.state)).toEqual(["ready", "missing", "stale", "unmanaged"])
  })

  it("uses the auto-update interval for built-in local entries", () => {
    const health = buildRuleSetHealth([
      item({ type: "local", tag: "builtin-local", update_interval: undefined, last_updated: "2026-07-24T00:00:00Z" }),
    ], autoUpdate, [], now)
    expect(health.stale).toBe(1)
    expect(health.items[0]?.state).toBe("stale")
  })

  it("surfaces the latest automatic update failure as an error", () => {
    const health = buildRuleSetHealth(
      [item()],
      autoUpdate,
      [
        log("ruleset auto update finished updated=1 failed=0 skipped=0", "2026-07-26T10:00:00Z"),
        log("ruleset auto update finished updated=2 failed=1 skipped=0", "2026-07-26T11:00:00Z"),
      ],
      now,
    )
    expect(health.tone).toBe("error")
    expect(health.latestAutoUpdate?.failed).toBe(1)
  })

  it("returns empty and healthy states without inventing issues", () => {
    expect(buildRuleSetHealth([], autoUpdate, [], now)).toMatchObject({ tone: "empty", total: 0 })
    expect(buildRuleSetHealth([item()], autoUpdate, [log("ruleset auto update finished updated=1 failed=0 skipped=0")], now)).toMatchObject({
      tone: "healthy",
      missing: 0,
      stale: 0,
    })
    expect(buildRuleSetHealth([
      item({ tag: "", last_updated: "invalid-time", update_interval: "invalid" }),
    ], autoUpdate, [], now)).toMatchObject({
      tone: "healthy",
      latestUpdatedAt: undefined,
      items: [{ tag: "#1", state: "ready" }],
    })
  })

  it("creates route deep links for a rule-set index", () => {
    expect(ruleSetHealthHref()).toBe("/policy/route")
    expect(ruleSetHealthHref(2)).toBe("/policy/route?path=route.rule_set%5B2%5D")
  })
})
