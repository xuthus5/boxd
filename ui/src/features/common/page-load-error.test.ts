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

  it("covers fallback API statuses, message patterns, and optional fields", () => {
    expect(pageLoadErrorHintKey()).toBe("common.errorHintUnknown")
    expect(classifyPageLoadError(new ApiError("unauthorized", 418, "other"))).toBe("unauthorized")
    expect(classifyPageLoadError(new ApiError("down", 503, "other"))).toBe("unavailable")
    expect(classifyPageLoadError(new ApiError("slow", 408, "other"))).toBe("timeout")
    expect(classifyPageLoadError(new ApiError("slow", 504, "other"))).toBe("timeout")
    expect(classifyPageLoadError(new ApiError("down", 418, "bad_gateway"))).toBe("network")
    expect(classifyPageLoadError(new ApiError("down", 418, "request_failed"))).toBe("network")
    expect(classifyPageLoadError(new ApiError("boom", 500, "other"))).toBe("internal")
    expect(classifyPageLoadError(new ApiError("mystery", 418, "other"))).toBe("unknown")
    expect(classifyPageLoadError(new ApiError("mystery", 418, ""))).toBe("unknown")
    for (const [message, code] of [
      ["401 unauthorized", "unauthorized"], ["deadline exceeded", "timeout"],
      ["not available", "unavailable"], ["503 response", "unavailable"],
      ["network offline", "network"], ["connection refused", "network"],
      ["failed to load", "internal"], ["internal failure", "internal"],
    ] as const) expect(classifyPageLoadError(new Error(message))).toBe(code)
    expect(classifyPageLoadError(null)).toBe("unknown")
    expect(pageLoadErrorMessage("  text  ", "fallback")).toBe("text")
    expect(pageLoadErrorMessage(new Error("  "), "fallback")).toBe("fallback")
    expect(formatPageLoadErrorTitle((key) => key, "network", "custom.title")).toBe("custom.title (network)")
    expect(formatPageLoadErrorTitle((key) => key, "unknown", "custom.title")).toBe("custom.title")
    expect(pageLoadErrorClipboardText(new Error("boom"))).toBe("code: unknown\nerror: boom")
    expect(pageLoadErrorClipboardText(new Error("boom"), { scope: "  ", path: "  " })).toBe("code: unknown\nerror: boom")
    expect(pageLoadErrorClipboardText(null)).toBe("")
  })
})
