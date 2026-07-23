import { describe, expect, it } from "vitest"

import {
  batchTestToastTone,
  formatBatchTestToastMessage,
  summarizeBatchTestResults,
} from "@/features/nodes/batch-test-summary"
import type { TestResult } from "@/lib/api/types"

const sample: TestResult[] = [
  { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 },
  { tag: "hk-01", test_type: "http", success: true, latency_ms: 22 },
  { tag: "us-01", test_type: "tcp", success: false, error: "timeout" },
  { tag: "jp-01", test_type: "tcp", success: true, latency_ms: 40 },
  { tag: "sg-01", test_type: "http", success: false, error: "connection refused" },
]

const t = (key: string, values?: Record<string, string | number>) => {
  if (key === "nodes.batchComplete") return "batch done"
  if (key === "nodes.batchSummary") {
    return `${values?.success}/${values?.total} ok, ${values?.failed} failed, avg ${values?.avg}`
  }
  if (key === "nodes.batchBest") return `best ${values?.tag} ${values?.latency}`
  if (key === "nodes.batchWorst") return `worst ${values?.tag} ${values?.latency}`
  if (key === "nodes.batchFailedSamples") return `failed ${values?.samples}`
  return key
}

describe("summarizeBatchTestResults", () => {
  it("counts success/failure and tracks best/worst latency", () => {
    expect(summarizeBatchTestResults(sample)).toEqual({
      total: 5,
      success: 3,
      failed: 2,
      avgLatencyMs: 26.7,
      bestTag: "hk-01",
      bestLatencyMs: 18,
      worstTag: "jp-01",
      worstLatencyMs: 40,
      failedSamples: [
        { tag: "us-01", testType: "TCP", error: "timeout" },
        { tag: "sg-01", testType: "HTTP", error: "network: connection refused" },
      ],
    })
  })

  it("handles empty or missing results", () => {
    const empty = {
      total: 0,
      success: 0,
      failed: 0,
      avgLatencyMs: undefined,
      bestTag: undefined,
      bestLatencyMs: undefined,
      worstTag: undefined,
      worstLatencyMs: undefined,
      failedSamples: [],
    }
    expect(summarizeBatchTestResults(undefined)).toEqual(empty)
    expect(summarizeBatchTestResults([])).toEqual(empty)
  })

  it("formats toast message with best/worst and failed samples", () => {
    const summary = summarizeBatchTestResults(sample)
    expect(formatBatchTestToastMessage(summary, t)).toContain("3/5 ok, 2 failed")
    expect(formatBatchTestToastMessage(summary, t)).toContain("best hk-01 18ms")
    expect(formatBatchTestToastMessage(summary, t)).toContain("worst jp-01 40ms")
    expect(formatBatchTestToastMessage(summary, t)).toContain("us-01/TCP: timeout")
    expect(batchTestToastTone(summary)).toBe("warning")
    expect(batchTestToastTone(summarizeBatchTestResults([
      { tag: "x", test_type: "tcp", success: false, error: "boom" },
    ]))).toBe("error")
    expect(batchTestToastTone(summarizeBatchTestResults([
      { tag: "x", test_type: "tcp", success: true, latency_ms: 10 },
    ]))).toBe("success")
    expect(formatBatchTestToastMessage(summarizeBatchTestResults([]), t)).toBe("batch done")
  })
})
