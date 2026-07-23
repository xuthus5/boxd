import { describe, expect, it } from "vitest"

import { nodeTestErrorClipboardText, nodeTestErrorLabel } from "@/features/nodes/node-test-error"

describe("node test error helpers", () => {
  it("formats failed probe diagnostics", () => {
    expect(nodeTestErrorClipboardText({
      tag: "hk",
      test_type: "http",
      success: false,
      error: "timeout",
      timestamp: "2026-07-24T00:00:00Z",
    })).toBe([
      "tag: hk",
      "test: http",
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
  })
})
