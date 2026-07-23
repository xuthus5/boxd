import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyCloseError,
  closeErrorClipboardText,
  closeErrorHintKey,
  filterSuppressedConnections,
  formatCloseErrorToast,
  formatClosedScopeMessage,
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

  it("formats densified close messages", () => {
    const t = (key: string, values?: Record<string, string | number>) => {
      if (key === "observability.closedOneTarget") return `closed ${values?.target}`
      if (key === "observability.closedOne") return "closed one"
      if (key === "observability.closedAllDone") return `closed all ${values?.count}`
      if (key === "observability.closedFilteredDone") return `closed filtered ${values?.count}`
      if (key === "observability.closeGroupDoneCount") return `closed group ${values?.group} ${values?.count}`
      return key
    }
    expect(formatClosedScopeMessage("one", t, { target: "a.com:443" })).toBe("closed a.com:443")
    expect(formatClosedScopeMessage("one", t)).toBe("closed one")
    expect(formatClosedScopeMessage("all", t, { count: 3 })).toBe("closed all 3")
    expect(formatClosedScopeMessage("filtered", t, { count: 2 })).toBe("closed filtered 2")
    expect(formatClosedScopeMessage("group", t, { group: "proxy", count: 4 })).toBe("closed group proxy 4")
  })

  it("classifies close failures and formats diagnostics", () => {
    expect(classifyCloseError(new ApiError("service not available", 503, "unavailable"))).toBe("unavailable")
    expect(classifyCloseError(new ApiError("connection not found", 404, "not_found"))).toBe("not_found")
    expect(classifyCloseError(new ApiError("invalid connection id", 400, "invalid_request"))).toBe("invalid_request")
    expect(classifyCloseError(new Error("boom"))).toBe("unknown")
    expect(closeErrorHintKey("unavailable")).toBe("observability.errorHintCloseUnavailable")

    const t = (key: string, values?: Record<string, string | number>) => {
      if (key === "observability.closeFailed") return "close failed"
      if (key === "observability.closeFailedScope") return `fail ${values?.scope}/${values?.code}`
      if (key === "observability.closedOne") return "one"
      return key
    }
    expect(formatCloseErrorToast(
      new ApiError("service not available", 503, "unavailable"),
      t,
      { scope: "one", target: "a.com:443" },
    )).toBe("fail a.com:443/unavailable: service not available")

    expect(closeErrorClipboardText(
      new ApiError("service not available", 503, "unavailable"),
      { scope: "all", count: 3 },
    )).toBe([
      "scope: all",
      "count: 3",
      "code: unavailable",
      "error: service not available",
    ].join("\n"))
  })
})
