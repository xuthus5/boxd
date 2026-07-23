import { describe, expect, it } from "vitest"

import {
  classifyNodeTestErrorMessage,
  formatNodeTestFailureSample,
  nodeTestErrorClipboardText,
  nodeTestErrorHintKey,
  nodeTestErrorLabel,
  resolveNodeTestErrorCode,
} from "@/features/nodes/node-test-error"

describe("node test error helpers", () => {
  it("formats failed probe diagnostics with code", () => {
    expect(nodeTestErrorClipboardText({
      tag: "hk",
      test_type: "http",
      success: false,
      error: "timeout",
      error_code: "timeout",
      timestamp: "2026-07-24T00:00:00Z",
    })).toBe([
      "tag: hk",
      "test: http",
      "code: timeout",
      "error: timeout",
      "at: 2026-07-24T00:00:00Z",
    ].join("\n"))
  })

  it("returns empty payload for successful results", () => {
    expect(nodeTestErrorClipboardText({
      tag: "hk",
      test_type: "tcp",
      success: true,
    })).toBe("")
  })

  it("falls back when error text is empty", () => {
    expect(nodeTestErrorLabel({ error: "  " }, "failed")).toBe("failed")
    expect(nodeTestErrorLabel({ error: "boom" }, "failed")).toBe("boom")
    expect(nodeTestErrorLabel({ error_code: "network" }, "failed")).toBe("network")
  })

  it("classifies and resolves error codes", () => {
    expect(classifyNodeTestErrorMessage("i/o timeout")).toBe("timeout")
    expect(classifyNodeTestErrorMessage("connection refused")).toBe("network")
    expect(classifyNodeTestErrorMessage("test service not available")).toBe("unavailable")
    expect(resolveNodeTestErrorCode({ error: "x", error_code: "network" })).toBe("network")
    expect(resolveNodeTestErrorCode({ error: "delay test failed: no response" })).toBe("no_response")
    expect(resolveNodeTestErrorCode({ success: true, error: "x" })).toBeUndefined()
    expect(nodeTestErrorHintKey("timeout")).toBe("nodes.errorHintTimeout")
    expect(formatNodeTestFailureSample({ error: "boom", error_code: "network" })).toBe("network: boom")
    expect(formatNodeTestFailureSample({ error: "timeout", error_code: "timeout" })).toBe("timeout")
    expect(formatNodeTestFailureSample({ error: "weird" })).toBe("weird")
  })
})
