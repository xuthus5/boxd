import { describe, expect, it } from "vitest"

import type { ConfigApplyEvent } from "@/lib/api/types"
import {
  filterConfigHistory,
  isConfigHistoryFilter,
} from "@/features/advanced/config-history-filter"

function event(overrides: Partial<ConfigApplyEvent> = {}): ConfigApplyEvent {
  return {
    id: "event-1",
    source: "update",
    status: "applied",
    hash: "abcdef0123456789",
    size: 128,
    applied_at: "2026-07-26T00:00:00Z",
    ...overrides,
  }
}

function ids(events: ConfigApplyEvent[]) {
  return events.map((item) => item.id)
}

describe("config history filter", () => {
  it("filters status groups and only keeps usable restorable events", () => {
    const events = [
      event({ id: "applied", status: "applied", restorable: true }),
      event({ id: "validated", status: "validated" }),
      event({ id: "rolled-back", status: "rolled_back", restorable: true }),
      event({ id: "validate-failed", status: "validate_failed", restorable: true }),
      event({ id: "current", current: true, restorable: true }),
      event({ id: "missing-id", id: "", restorable: true }),
    ]

    expect(ids(filterConfigHistory(events, "", "all"))).toEqual([
      "applied", "validated", "rolled-back", "validate-failed", "current", "",
    ])
    expect(ids(filterConfigHistory(events, "", "applied"))).toEqual(["applied", "current", ""])
    expect(ids(filterConfigHistory(events, "", "validated"))).toEqual(["validated"])
    expect(ids(filterConfigHistory(events, "", "failed"))).toEqual(["rolled-back", "validate-failed"])
    expect(ids(filterConfigHistory(events, "", "restorable"))).toEqual(["applied"])
  })

  it("searches hash, source, status, error, and error code case-insensitively", () => {
    const target = event({
      id: "target",
      source: "validate_dns",
      status: "validate_failed",
      hash: "ABC123",
      error: "Port denied by kernel",
      error_code: "CONFIG_INVALID",
    })
    const other = event({ id: "other", hash: "fedcba" })
    const events = [target, other]

    for (const query of ["abc123", "VALIDATE_DNS", "validate_failed", "PORT DENIED", "config_invalid"]) {
      expect(ids(filterConfigHistory(events, query, "all"))).toEqual(["target"])
    }
    expect(filterConfigHistory(events, "no such record", "all")).toEqual([])
  })

  it("combines a trimmed search query with a status filter", () => {
    const events = [
      event({ id: "dns-failed", source: "validate_dns", status: "validate_failed" }),
      event({ id: "raw-failed", source: "validate_raw", status: "validate_failed" }),
      event({ id: "dns-ok", source: "validate_dns", status: "validated" }),
    ]

    expect(ids(filterConfigHistory(events, "  dns  ", "failed"))).toEqual(["dns-failed"])
    expect(isConfigHistoryFilter("restorable")).toBe(true)
    expect(isConfigHistoryFilter("invalid")).toBe(false)
  })
})
