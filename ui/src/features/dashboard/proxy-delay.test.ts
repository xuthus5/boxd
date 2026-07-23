import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyDelayErrorMessage,
  delayBatchToastTone,
  delayFailureFromError,
  formatDelayBatchMessage,
  formatDelayValue,
  isDelayFailure,
  pickPrimaryGroup,
  sortDelayEntries,
  summarizeDelays,
} from "@/features/dashboard/proxy-delay"

describe("proxy-delay helpers", () => {
  it("picks preferred selector groups", () => {
    expect(pickPrimaryGroup([])).toBeNull()
    expect(pickPrimaryGroup([
      { tag: "other", type: "selector", now: "a", all: ["a"] },
      { tag: "proxy", type: "selector", now: "b", all: ["b"] },
    ])?.tag).toBe("proxy")
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
})
