import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  filterSuppressedConnections,
  isBenignCloseMiss,
  isBulkClosing,
  isConnectionRowBusy,
  isGroupClosing,
  pruneSuppressedIds,
  suppressConnectionIds,
} from "@/features/observability/connection-close"
import type { Connection } from "@/lib/api/types"

const sample: Connection[] = [
  { id: 1, target: "a", outbound: "proxy", upload: 0, download: 0, start: "2026-01-01T00:00:00Z" },
  { id: 2, target: "b", outbound: "direct", upload: 0, download: 0, start: "2026-01-01T00:00:00Z" },
]

describe("connection close helpers", () => {
  it("tracks bulk and per-row busy state", () => {
    expect(isBulkClosing("all")).toBe(true)
    expect(isBulkClosing("filtered")).toBe(true)
    expect(isBulkClosing("group:outbound:proxy")).toBe(true)
    expect(isBulkClosing("1")).toBe(false)
    expect(isConnectionRowBusy("1", "1")).toBe(true)
    expect(isConnectionRowBusy("1", "2")).toBe(false)
    expect(isConnectionRowBusy("all", "2")).toBe(true)
    expect(isGroupClosing("group:rule:x", "rule", "x")).toBe(true)
    expect(isGroupClosing("group:rule:x", "rule", "y")).toBe(false)
  })

  it("suppresses and prunes closed ids against live snapshots", () => {
    const suppressed = suppressConnectionIds(new Set(), [1, "3"])
    expect(filterSuppressedConnections(sample, suppressed).map((item) => item.id)).toEqual([2])
    const pruned = pruneSuppressedIds(suppressed, new Set(["1"]))
    expect([...pruned]).toEqual(["1"])
    expect(pruneSuppressedIds(new Set(), new Set(["1"])).size).toBe(0)
  })

  it("treats not-found close races as benign", () => {
    expect(isBenignCloseMiss(new ApiError("gone", 404, "not_found"))).toBe(true)
    expect(isBenignCloseMiss(new ApiError("boom", 500, "internal_error"))).toBe(false)
    expect(isBenignCloseMiss(new Error("x"))).toBe(false)
  })
})
