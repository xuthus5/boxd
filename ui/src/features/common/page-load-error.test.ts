import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyPageLoadError,
  formatPageLoadErrorTitle,
  pageLoadErrorClipboardText,
  pageLoadErrorHintKey,
  pageLoadErrorMessage,
} from "@/features/common/page-load-error"

describe("page load error diagnostics", () => {
  it("maps codes to hints", () => {
    expect(pageLoadErrorHintKey("network")).toBe("common.errorHintNetwork")
    expect(pageLoadErrorHintKey("timeout")).toBe("common.errorHintTimeout")
    expect(pageLoadErrorHintKey("nope")).toBe("common.errorHintUnknown")
  })

  it("classifies api and network failures", () => {
    expect(classifyPageLoadError(new ApiError("nope", 401, "unauthorized"))).toBe("unauthorized")
    expect(classifyPageLoadError(new ApiError("down", 503, "unavailable"))).toBe("unavailable")
    expect(classifyPageLoadError(new ApiError("slow", 504, "timeout"))).toBe("timeout")
    expect(classifyPageLoadError(new ApiError("boom", 500, "internal_error"))).toBe("internal")
    expect(classifyPageLoadError(new Error("failed to fetch"))).toBe("network")
    expect(classifyPageLoadError(new Error("mystery"))).toBe("unknown")
  })

  it("formats message, title, and clipboard payload", () => {
    expect(pageLoadErrorMessage(new Error("  boom  "), "fallback")).toBe("boom")
    expect(pageLoadErrorMessage({}, "fallback")).toBe("fallback")
    const t = (key: string, values?: Record<string, string | number>) =>
      key === "common.loadFailedWithCode" ? `fail(${values?.code})` : key
    expect(formatPageLoadErrorTitle(t, "network")).toBe("fail(network)")
    expect(formatPageLoadErrorTitle(t, "unknown")).toBe("common.loadFailed")
    const payload = pageLoadErrorClipboardText(new ApiError("boom", 500, "internal_error"), {
      scope: "dashboard",
      path: "/api/service/status",
    })
    expect(payload).toContain("scope: dashboard")
    expect(payload).toContain("path: /api/service/status")
    expect(payload).toContain("code: internal")
    expect(payload).toContain("error: boom")
  })
})
