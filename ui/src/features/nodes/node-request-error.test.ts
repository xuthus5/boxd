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
  })
})
