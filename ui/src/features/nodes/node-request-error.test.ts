import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyNodeRequestError,
  formatNodeRequestErrorToast,
  nodeRequestErrorClipboardText,
  nodeRequestErrorHintKey,
} from "@/features/nodes/node-request-error"

describe("node request error helpers", () => {
  it("classifies API and message failures", () => {
    expect(classifyNodeRequestError(new ApiError("bad", 400, "invalid_request"))).toBe("invalid_input")
    expect(classifyNodeRequestError(new ApiError("missing", 404, "node_not_found"))).toBe("not_found")
    expect(classifyNodeRequestError(new ApiError("duplicate", 409, "conflict"))).toBe("conflict")
    expect(classifyNodeRequestError(new ApiError("duplicate", 409, "node_tag_conflict"))).toBe("conflict")
    expect(classifyNodeRequestError(new ApiError("offline", 503, "unavailable"))).toBe("unavailable")
    expect(classifyNodeRequestError(new Error("Failed to fetch"))).toBe("network")
    expect(classifyNodeRequestError(new Error("timeout"))).toBe("timeout")
    expect(nodeRequestErrorHintKey("invalid_input")).toBe("nodes.errorHintRequestInvalid")
    expect(nodeRequestErrorHintKey("conflict")).toBe("nodes.errorHintRequestConflict")
  })

  it.each([
    ["not available", "unavailable"],
    ["not running", "unavailable"],
    ["not found", "not_found"],
    ["invalid payload", "invalid_input"],
    ["field required", "invalid_input"],
    ["parse failed", "invalid_input"],
    ["deadline exceeded", "timeout"],
    ["unsupported operation", "unsupported"],
    ["not selectable", "unsupported"],
    ["failed to fetch", "network"],
    ["connection refused", "network"],
    ["offline", "network"],
    ["failed to update", "update_failed"],
    ["ordinary error", "unknown"],
  ])("classifies message %s as %s", (message, expected) => {
    expect(classifyNodeRequestError(new Error(message))).toBe(expected)
  })

  it("covers API aliases, hints, and empty error shapes", () => {
    expect(classifyNodeRequestError(new ApiError("missing", 404, "not_found"))).toBe("not_found")
    expect(classifyNodeRequestError(new ApiError("failed", 500, "node_update_failed"))).toBe("update_failed")
    expect(classifyNodeRequestError(new ApiError("unsupported", 400, "runtime_not_selectable"))).toBe("unsupported")
    expect(classifyNodeRequestError(new ApiError("slow", 504, "timeout"))).toBe("timeout")
    expect(classifyNodeRequestError(new ApiError("gateway", 502, "bad_gateway"))).toBe("network")
    expect(classifyNodeRequestError(new ApiError("request", 500, "request_failed"))).toBe("network")
    expect(classifyNodeRequestError(new ApiError("other", 500, "other"))).toBe("unknown")
    expect(classifyNodeRequestError(undefined)).toBe("unknown")
    expect(nodeRequestErrorHintKey()).toBe("nodes.errorHintRequestUnknown")
    expect(nodeRequestErrorHintKey("other")).toBe("nodes.errorHintRequestUnknown")
    expect(nodeRequestErrorHintKey("network")).toBe("nodes.errorHintRequestNetwork")
  })

  it("formats toast and clipboard diagnostics", () => {
    expect(formatNodeRequestErrorToast(new ApiError("link is required", 400, "invalid_request"), "fallback"))
      .toBe("invalid_input: link is required")
    expect(nodeRequestErrorClipboardText(new Error("boom"), { scope: "import-parse", tag: "hk" })).toBe([
      "scope: import-parse",
      "tag: hk",
      "code: unknown",
      "error: boom",
    ].join("\n"))
    expect(nodeRequestErrorClipboardText(new Error("  "))).toBe("")
    expect(nodeRequestErrorClipboardText(new Error("boom"), {
      scope: "", group: " group ", tag: " tag ",
    })).toBe(["group: group", "tag: tag", "code: unknown", "error: boom"].join("\n"))
    expect(formatNodeRequestErrorToast(new Error("  "), "fallback")).toBe("fallback")
    expect(formatNodeRequestErrorToast("offline", "fallback")).toBe("network: fallback")
  })
})
