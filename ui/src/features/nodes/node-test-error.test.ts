import { describe, expect, it } from "vitest"

import {
  classifyNodeTestErrorMessage,
  classifyNodeTestRequestError,
  formatNodeTestFailureSample,
  formatNodeTestRequestErrorToast,
  nodeTestErrorClipboardText,
  nodeTestErrorHintKey,
  nodeTestErrorLabel,
  nodeTestRequestErrorClipboardText,
  resolveNodeTestErrorCode,
} from "@/features/nodes/node-test-error"
import { ApiError } from "@/lib/api/client"

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
  it("classifies request-level test failures", () => {
    expect(classifyNodeTestRequestError(new ApiError("service not available", 503, "unavailable"))).toBe("unavailable")
    expect(classifyNodeTestRequestError(new Error("i/o timeout"))).toBe("timeout")
    expect(nodeTestRequestErrorClipboardText(new Error("network down"), "hk")).toContain("tag: hk")
    expect(formatNodeTestRequestErrorToast(new ApiError("kernel offline", 503, "unavailable"), "fallback"))
      .toBe("unavailable: kernel offline")
  })

  it.each([
    ["unsupported", "unsupported"],
    ["not probeable", "unsupported"],
    ["invalid parameter", "invalid_input"],
    ["required field missing", "invalid_input"],
    ["no response from peer", "no_response"],
    ["empty dns answer", "empty_response"],
    ["empty response body", "empty_response"],
    ["dns rcode SERVFAIL", "dns_rcode"],
    ["deadline exceeded", "timeout"],
    ["connection reset", "network"],
    ["no such host", "network"],
    ["ping failed", "network"],
    ["broken pipe", "network"],
    ["something else", "unknown"],
  ])("classifies message %s as %s", (message, expected) => {
    expect(classifyNodeTestErrorMessage(message)).toBe(expected)
  })

  it("covers hint, result, and clipboard fallbacks", () => {
    expect(nodeTestErrorHintKey()).toBe("nodes.errorHintUnknown")
    expect(nodeTestErrorHintKey("not-a-code")).toBe("nodes.errorHintUnknown")
    expect(nodeTestErrorHintKey("dns_rcode")).toBe("nodes.errorHintDNSRcode")
    expect(resolveNodeTestErrorCode({ success: false })).toBeUndefined()
    expect(resolveNodeTestErrorCode({ error_code: "custom" })).toBe("custom")
    expect(nodeTestErrorClipboardText({
      tag: "  ", test_type: "  ", success: false, error: "  ", error_code: "network", timestamp: "  ",
    })).toBe("code: network")
    expect(formatNodeTestFailureSample({ error: "boom", error_code: "unknown" })).toBe("boom")
    expect(formatNodeTestFailureSample({ error: "", error_code: "timeout" })).toBe("timeout: failed")
  })

  it("classifies every API request code and fallback shape", () => {
    expect(classifyNodeTestRequestError(new ApiError("bad", 400, "invalid_input"))).toBe("invalid_input")
    expect(classifyNodeTestRequestError(new ApiError("slow", 504, "timeout"))).toBe("timeout")
    expect(classifyNodeTestRequestError(new ApiError("no", 400, "unsupported"))).toBe("unsupported")
    expect(classifyNodeTestRequestError(new ApiError("bad", 500, "other"))).toBe("unknown")
    expect(classifyNodeTestRequestError(null)).toBe("unknown")
    expect(nodeTestRequestErrorClipboardText("connection refused", "  ")).toContain("code: network")
    expect(nodeTestRequestErrorClipboardText("", "hk")).toBe("")
    expect(formatNodeTestRequestErrorToast(new Error("  "), "fallback")).toBe("fallback")
    expect(formatNodeTestRequestErrorToast("", "fallback")).toBe("fallback")
    expect(formatNodeTestRequestErrorToast(new Error("ordinary"), "fallback")).toBe("ordinary")
  })

})
