import { describe, expect, it } from "vitest"

import type { ConfigApplyEvent } from "@/lib/api/types"
import { summarizeConfigHistory } from "@/features/advanced/config-history-summary"

function event(overrides: Partial<ConfigApplyEvent> = {}): ConfigApplyEvent {
  return {
    id: "event-1",
    source: "update",
    status: "applied",
    hash: "hash",
    size: 1,
    applied_at: "2026-07-26T00:00:00Z",
    ...overrides,
  }
}

describe("config history summary", () => {
  it("counts statuses, current records, and usable restore candidates", () => {
    expect(summarizeConfigHistory([
      event({ id: "applied", restorable: true }),
      event({ id: "validated", status: "validated" }),
      event({ id: "rolled-back", status: "rolled_back", restorable: true }),
      event({ id: "validate-failed", status: "validate_failed", restorable: true }),
      event({ id: "current", current: true, restorable: true }),
      event({ id: "", restorable: true }),
    ])).toEqual({ total: 6, applied: 3, validated: 1, failed: 2, restorable: 1, current: 1 })
  })

  it("handles empty history", () => {
    expect(summarizeConfigHistory([])).toEqual({ total: 0, applied: 0, validated: 0, failed: 0, restorable: 0, current: 0 })
  })
})
