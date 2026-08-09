import { describe, expect, it, vi } from "vitest"

import { ApiError } from "@/lib/api/client"
import { api } from "@/lib/api/endpoints"
import {
  classifyDelayErrorMessage,
  delayBatchFailureClipboardText,
  delayBatchToastTone,
  delayErrorHintKey,
  delayFailureClipboardText,
  delayFailureFromError,
  delayRequestErrorClipboardText,
  formatDelayBatchMessage,
  formatDelayFailureSample,
  formatDelayRequestErrorToast,
  formatDelayValue,
  isDelayFailure,
  measureGroupDelays,
  pickPrimaryGroup,
  sortDelayEntries,
  summarizeDelays,
} from "@/features/dashboard/proxy-delay"

vi.mock("@/lib/api/endpoints", () => ({
  api: {
    nodes: {
      delay: vi.fn(),
      urlTest: vi.fn(),
    },
  },
}))

describe("proxy-delay helpers", () => {
  it("picks preferred selector groups", () => {
    expect(pickPrimaryGroup([])).toBeNull()
    expect(pickPrimaryGroup([
      { tag: "other", type: "selector", now: "a", all: ["a"] },
      { tag: "proxy", type: "selector", now: "b", all: ["b"] },
    ])?.tag).toBe("proxy")
  })

  it("picks urltest group when no selector exists", () => {
    expect(pickPrimaryGroup([
      { tag: "empty", type: "selector", now: "", all: [] },
      { tag: "auto", type: "urltest", now: "a", all: ["a"] },
    ])?.tag).toBe("auto")
    expect(pickPrimaryGroup([
      { tag: "auto", type: "urltest", now: "a", all: ["a"] },
      { tag: "proxy", type: "urltest", now: "b", all: ["b"] },
    ])?.tag).toBe("proxy")
    expect(pickPrimaryGroup([
      { tag: "auto", type: "urltest", now: "a", all: [] },
    ])).toBeNull()
  })

  it("handles selector fallbacks and delay sort ties", () => {
    expect(pickPrimaryGroup([
      { tag: "empty", type: "selector", now: "", all: [] },
      { tag: "urltest", type: "urltest", now: "a", all: ["a"] },
    ])?.tag).toBe("urltest")
    expect(pickPrimaryGroup([
      { tag: "fallback", type: "selector", now: "a", all: ["a"] },
      { tag: "select", type: "selector", now: "b", all: ["b"] },
    ])?.tag).toBe("select")
    expect(pickPrimaryGroup([
      { tag: "fallback", type: "selector", now: "a", all: ["a"] },
      { tag: "other", type: "selector", now: "b", all: ["b"] },
    ])?.tag).toBe("fallback")
    expect(sortDelayEntries({
      z: 20,
      a: 20,
      y: { failed: true, error: "failed" },
      b: { failed: true, error: "failed" },
    }).map(([tag]) => tag)).toEqual(["a", "z", "b", "y"])
  })

  it("formats and sorts delays with structured failures last", () => {
    expect(formatDelayValue(12, "timeout")).toBe("12 ms")
    expect(formatDelayValue({ failed: true, error: "timeout", code: "timeout" }, "failed")).toBe("timeout")
    expect(formatDelayValue({ failed: true, error: "boom", code: "network" }, "failed")).toBe("network: boom")
    expect(formatDelayValue(undefined, "timeout")).toBe("—")
    expect(sortDelayEntries({
      b: 40,
      a: 12,
      c: { failed: true, error: "timeout", code: "timeout" },
    }).map(([tag]) => tag)).toEqual(["a", "b", "c"])
  })

  it("uses failure fallbacks when details are missing", () => {
    expect(formatDelayValue({ failed: true, error: "", code: "unknown" }, "failed")).toBe("failed")
    expect(formatDelayValue({ failed: true, error: "", code: "network" }, "failed")).toBe("network: failed")
    expect(formatDelayFailureSample({ tag: "x", error: "timeout", code: "timeout" })).toBe("x: timeout")
    expect(formatDelayFailureSample({ tag: "x", error: "boom", code: "network" })).toBe("x: network: boom")
    expect(formatDelayFailureSample({ tag: "x", error: "boom" })).toBe("x: boom")
    expect(delayFailureClipboardText("  ", { failed: true, error: "  ", code: "timeout" })).toBe("code: timeout")
    expect(delayFailureClipboardText(" proxy ", { failed: true, error: "failed", code: "" })).toBe(
      "tag: proxy\nerror: failed",
    )
  })

  it("classifies delay errors and summarizes batch diagnostics", () => {
    expect(classifyDelayErrorMessage("delay test failed: no response")).toBe("no_response")
    expect(classifyDelayErrorMessage("kernel not running", "unavailable")).toBe("unavailable")
    expect(delayFailureFromError(new ApiError("timeout", 502, "runtime_delay_failed")).code).toBe("timeout")
    expect(isDelayFailure({ failed: true, error: "x" })).toBe(true)

    const summary = summarizeDelays({
      a: 12,
      b: 34,
      c: { failed: true, error: "timeout", code: "timeout" },
      d: { failed: true, error: "connection refused", code: "network" },
    })
    expect(summary).toMatchObject({
      total: 4,
      ok: 2,
      failed: 2,
      avgLatencyMs: 23,
      bestTag: "a",
      bestLatencyMs: 12,
      worstTag: "b",
      worstLatencyMs: 34,
    })
    expect(summary.failedSamples).toEqual([
      { tag: "c", error: "timeout", code: "timeout" },
      { tag: "d", error: "connection refused", code: "network" },
    ])
    const t = (key: string, values?: Record<string, string | number>) => {
      if (key === "dashboard.proxyDelayDone") return `${values?.ok}/${values?.total} ok, ${values?.failed} failed`
      if (key === "dashboard.proxyDelayAvg") return `avg ${values?.avg}`
      if (key === "dashboard.proxyDelayBest") return `best ${values?.tag} ${values?.latency}`
      if (key === "dashboard.proxyDelayWorst") return `worst ${values?.tag} ${values?.latency}`
      if (key === "dashboard.proxyDelayFailedSamples") return `failed ${values?.samples}`
      return key
    }
    const message = formatDelayBatchMessage(summary, t)
    expect(message).toContain("2/4 ok, 2 failed")
    expect(message).toContain("best a 12ms")
    expect(message).toContain("c: timeout")
    expect(delayBatchToastTone(summary)).toBe("warning")
    expect(delayBatchToastTone({ total: 1, ok: 0, failed: 1, failedSamples: [] })).toBe("error")
  })

  it("covers explicit error codes and message classifications", () => {
    const cases: Array<[string | undefined, string | undefined, string]> = [
      [undefined, undefined, "unknown"],
      ["", "", "unknown"],
      ["ignored", "not_found", "not_found"],
      ["ignored", "runtime_group_not_found", "not_found"],
      ["ignored", "runtime_not_selectable", "unsupported"],
      ["kernel not running", undefined, "unavailable"],
      ["service not available", undefined, "unavailable"],
      ["resource not found", undefined, "not_found"],
      ["probe has no response", undefined, "no_response"],
      ["request timeout", undefined, "timeout"],
      ["deadline exceeded", undefined, "timeout"],
      ["i/o timeout", undefined, "timeout"],
      ["connection reset by peer", undefined, "network"],
      ["no such host", undefined, "network"],
      ["network unreachable", undefined, "network"],
      ["tls handshake failed", undefined, "network"],
      ["x509 certificate error", undefined, "network"],
      ["delay failed", "runtime_delay_failed", "timeout"],
      ["unclassified", undefined, "unknown"],
    ]
    for (const [message, code, expected] of cases) {
      expect(classifyDelayErrorMessage(message, code)).toBe(expected)
    }
    expect(delayErrorHintKey("unavailable")).toBe("dashboard.errorHintDelayUnavailable")
    expect(delayErrorHintKey("not_found")).toBe("dashboard.errorHintDelayNotFound")
    expect(delayErrorHintKey("no_response")).toBe("dashboard.errorHintDelayNoResponse")
    expect(delayErrorHintKey("timeout")).toBe("dashboard.errorHintDelayTimeout")
    expect(delayErrorHintKey("network")).toBe("dashboard.errorHintDelayNetwork")
    expect(delayErrorHintKey("unsupported")).toBe("dashboard.errorHintDelayUnsupported")
    expect(delayErrorHintKey("other")).toBe("dashboard.errorHintDelayUnknown")
  })

  it("normalizes API, native, and unknown failures", () => {
    expect(delayFailureFromError(new ApiError("  ", 500, "unavailable"))).toEqual({
      failed: true,
      error: "delay test failed",
      code: "unavailable",
    })
    expect(delayFailureFromError(new Error("  "))).toEqual({
      failed: true,
      error: "delay test failed",
      code: "unknown",
    })
    expect(delayFailureFromError("bad")).toEqual({
      failed: true,
      error: "delay test failed",
      code: "unknown",
    })
  })

  it("summarizes empty, non-finite, and unstructured failures", () => {
    expect(summarizeDelays({})).toEqual({
      total: 0,
      ok: 0,
      failed: 0,
      avgLatencyMs: undefined,
      bestTag: undefined,
      bestLatencyMs: undefined,
      worstTag: undefined,
      worstLatencyMs: undefined,
      failedSamples: [],
    })
    const summary = summarizeDelays({
      " ": { failed: true, error: "", code: "unknown" },
      nan: Number.NaN,
      nullish: null as never,
      fourth: { failed: true, error: "fourth", code: "network" },
      fifth: { failed: true, error: "fifth", code: "timeout" },
      sixth: { failed: true, error: "sixth", code: "unsupported" },
    })
    expect(summary).toMatchObject({ total: 6, ok: 0, failed: 6 })
    expect(summary.failedSamples).toEqual([
      { tag: "—", error: "failed", code: undefined },
      { tag: "nan", error: "failed", code: undefined },
      { tag: "nullish", error: "failed", code: undefined },
    ])
    expect(delayBatchToastTone({ total: 0, ok: 0, failed: 0, failedSamples: [] })).toBe("success")
    expect(formatDelayBatchMessage({ total: 0, ok: 0, failed: 0, failedSamples: [] }, (key) => key))
      .toBe("dashboard.proxyDelayDone")
  })

  it("formats request-level delay diagnostics", () => {
    const err = new Error("service not available")
    expect(formatDelayRequestErrorToast(err)).toContain("service not available")
    expect(delayRequestErrorClipboardText(err, "proxy-delay")).toContain("scope: proxy-delay")
    const summary = summarizeDelays({
      a: { failed: true, error: "timeout", code: "timeout" },
      b: 12,
    })
    expect(delayBatchFailureClipboardText(summary)).toContain("tag: a")
    expect(delayBatchFailureClipboardText({ total: 0, ok: 0, failed: 0, failedSamples: [] })).toBe("")
    expect(delayRequestErrorClipboardText(new Error("timeout"))).toContain("scope: delay")
    expect(formatDelayRequestErrorToast(new ApiError("offline", 503, "unavailable"))).toBe("unavailable: offline")
    expect(formatDelayRequestErrorToast({}, "fallback")).toBe("delay test failed")
  })

  it("uses urltest results and probes members after a request failure", async () => {
    vi.mocked(api.nodes.urlTest).mockResolvedValue({ a: 10, b: Number.NaN } as Record<string, number>)
    await expect(measureGroupDelays("proxy", ["a", "b", "c"])).resolves.toEqual({
      a: 10,
      b: { failed: true, error: "delay test failed: no response", code: "no_response" },
      c: { failed: true, error: "delay test failed: no response", code: "no_response" },
    })

    vi.mocked(api.nodes.urlTest).mockRejectedValue(new Error("urltest unavailable"))
    vi.mocked(api.nodes.delay).mockImplementation(async (tag) => {
      if (tag === "a") return { delay: 12 }
      if (tag === "b") return { delay: 0 }
      if (tag === "c") return {}
      throw new ApiError("node unavailable", 503, "unavailable")
    })
    await expect(measureGroupDelays("proxy", ["a", "b", "c", "d"])).resolves.toEqual({
      a: 12,
      b: { failed: true, error: "delay test failed: no response", code: "no_response" },
      c: { failed: true, error: "delay test failed: no response", code: "no_response" },
      d: { failed: true, error: "node unavailable", code: "unavailable" },
    })
  })

})
