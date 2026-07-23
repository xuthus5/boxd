import { describe, expect, it } from "vitest"

import { summarizeBatchTestResults } from "@/features/nodes/batch-test-summary"
import type { TestResult } from "@/lib/api/types"

const sample: TestResult[] = [
  { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 },
  { tag: "hk-01", test_type: "http", success: true, latency_ms: 22 },
  { tag: "us-01", test_type: "tcp", success: false, error: "timeout" },
  { tag: "jp-01", test_type: "tcp", success: true, latency_ms: 40 },
]

describe("summarizeBatchTestResults", () => {
  it("counts success/failure and tracks best/worst latency", () => {
    expect(summarizeBatchTestResults(sample)).toEqual({
      total: 4,
      success: 3,
      failed: 1,
      avgLatencyMs: 26.7,
      bestTag: "hk-01",
      bestLatencyMs: 18,
      worstTag: "jp-01",
      worstLatencyMs: 40,
    })
  })

  it("handles empty or missing results", () => {
    expect(summarizeBatchTestResults(undefined)).toEqual({
      total: 0,
      success: 0,
      failed: 0,
      avgLatencyMs: undefined,
      bestTag: undefined,
      bestLatencyMs: undefined,
      worstTag: undefined,
      worstLatencyMs: undefined,
    })
    expect(summarizeBatchTestResults([])).toEqual({
      total: 0,
      success: 0,
      failed: 0,
      avgLatencyMs: undefined,
      bestTag: undefined,
      bestLatencyMs: undefined,
      worstTag: undefined,
      worstLatencyMs: undefined,
    })
  })
})
