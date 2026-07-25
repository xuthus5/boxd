import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyCloseError,
  closeErrorMessage,
  closeErrorClipboardText,
  closeErrorHintKey,
  filterSuppressedConnections,
  formatCloseErrorToast,
  formatClosedScopeMessage,
  groupClosingKey,
  isBenignCloseMiss,
  isBulkClosing,
  isConnectionRowBusy,
  isGroupClosing,
  liveConnectionIdSet,
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
    expect(isBulkClosing(null)).toBe(false)
    expect(isBulkClosing("all")).toBe(true)
    expect(isBulkClosing("filtered")).toBe(true)
    expect(isBulkClosing("group:outbound:proxy")).toBe(true)
    expect(isBulkClosing("1")).toBe(false)
    expect(isConnectionRowBusy("1", "1")).toBe(true)
    expect(isConnectionRowBusy("1", "2")).toBe(false)
    expect(isConnectionRowBusy("all", "2")).toBe(true)
    expect(isConnectionRowBusy(null, "1")).toBe(false)
    expect(isGroupClosing(null, "rule", "x")).toBe(false)
    expect(isGroupClosing("all", "rule", "x")).toBe(true)
    expect(isGroupClosing("filtered", "rule", "x")).toBe(true)
    expect(isGroupClosing("group:rule:x", "rule", "x")).toBe(true)
    expect(isGroupClosing("group:rule:x", "rule", "y")).toBe(false)
    expect(groupClosingKey("process", "/usr/bin/curl")).toBe("group:process:/usr/bin/curl")
  })

  it("suppresses and prunes closed ids against live snapshots", () => {
    const suppressed = suppressConnectionIds(new Set(), [1, "3"])
    expect(filterSuppressedConnections(sample, suppressed).map((item) => item.id)).toEqual([2])
    const pruned = pruneSuppressedIds(suppressed, new Set(["1"]))
    expect([...pruned]).toEqual(["1"])
    expect(pruneSuppressedIds(new Set(), new Set(["1"])).size).toBe(0)
    expect([...pruneSuppressedIds(new Set(["1"]), new Set(["1"]))]).toEqual(["1"])
    expect(filterSuppressedConnections(sample, new Set()).map((item) => item.id)).toEqual([1, 2])
    expect([...liveConnectionIdSet(sample)]).toEqual(["1", "2"])
  })

  it("treats not-found close races as benign", () => {
    expect(isBenignCloseMiss(new ApiError("gone", 404, "not_found"))).toBe(true)
    expect(isBenignCloseMiss(new ApiError("gone", 500, "not_found"))).toBe(true)
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
    expect(formatClosedScopeMessage("all", t)).toBe("closed all 0")
    expect(formatClosedScopeMessage("filtered", t)).toBe("closed filtered 0")
    expect(formatClosedScopeMessage("group", t)).toBe("closed group — 0")
    expect(formatClosedScopeMessage("one", t, { target: "  " })).toBe("closed one")
  })

  it("classifies close failures and formats diagnostics", () => {
    expect(classifyCloseError(new ApiError("service not available", 503, "unavailable"))).toBe("unavailable")
    expect(classifyCloseError(new ApiError("connection not found", 404, "not_found"))).toBe("not_found")
    expect(classifyCloseError(new ApiError("invalid connection id", 400, "invalid_request"))).toBe("invalid_request")
    expect(classifyCloseError(new ApiError("temporary unavailable", 500, "other"))).toBe("unavailable")
    expect(classifyCloseError(new ApiError("connection not found", 500, "other"))).toBe("not_found")
    expect(classifyCloseError(new ApiError("too many connections", 400, "other"))).toBe("invalid_request")
    expect(classifyCloseError(new ApiError("specify only one filter", 400, "other"))).toBe("invalid_request")
    expect(classifyCloseError(new ApiError("invalid filter", 400, "other"))).toBe("invalid_request")
    expect(classifyCloseError(new ApiError("unknown failure", 500, "other"))).toBe("unknown")
    expect(classifyCloseError(new Error("service unavailable"))).toBe("unavailable")
    expect(classifyCloseError(new Error("connection not found"))).toBe("not_found")
    expect(classifyCloseError(new Error("invalid request"))).toBe("invalid_request")
    expect(classifyCloseError(new Error("too many requests"))).toBe("invalid_request")
    expect(classifyCloseError(new Error("boom"))).toBe("unknown")
    expect(classifyCloseError("boom")).toBe("unknown")
    expect(closeErrorHintKey("unavailable")).toBe("observability.errorHintCloseUnavailable")
    expect(closeErrorHintKey("not_found")).toBe("observability.errorHintCloseNotFound")
    expect(closeErrorHintKey("invalid_request")).toBe("observability.errorHintCloseInvalid")
    expect(closeErrorHintKey("other")).toBe("observability.errorHintCloseUnknown")

    expect(closeErrorMessage(new Error("  detail  "), "fallback")).toBe("detail")
    expect(closeErrorMessage(new Error("  "), "fallback")).toBe("fallback")
    expect(closeErrorMessage({ message: "detail" }, "fallback")).toBe("fallback")

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

    expect(formatCloseErrorToast(new Error("  "), t, { scope: "one", target: "  " }))
      .toBe("fail one/—: close failed")
    expect(formatCloseErrorToast(new Error("unknown"), t, { scope: "all" }))
      .toBe("fail all/—: unknown")
    expect(formatCloseErrorToast(new Error("not found"), t, { scope: "filtered" }))
      .toBe("fail filtered/not_found: not found")
    expect(formatCloseErrorToast(new Error("invalid"), t, { scope: "group", group: "  " }))
      .toBe("fail —/invalid_request: invalid")
    expect(formatCloseErrorToast(new Error("bad"), t, { scope: "group", group: "proxy" }))
      .toBe("fail proxy/—: bad")

    expect(closeErrorClipboardText(
      new ApiError("service not available", 503, "unavailable"),
      { scope: "all", count: 3 },
    )).toBe([
      "scope: all",
      "count: 3",
      "code: unavailable",
      "error: service not available",
    ].join("\n"))
    expect(closeErrorClipboardText(new Error("  "), { scope: "one", target: "  " })).toBe([
      "scope: one",
      "code: unknown",
      "error: close failed",
    ].join("\n"))
    expect(closeErrorClipboardText(new Error("bad"), {
      scope: "group",
      target: "target",
      group: " proxy ",
      count: 0,
    })).toBe([
      "scope: group",
      "target: target",
      "group: proxy",
      "count: 0",
      "code: unknown",
      "error: bad",
    ].join("\n"))
  })
})
