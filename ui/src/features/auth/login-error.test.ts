import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyLoginError,
  formatLoginErrorTitle,
  loginErrorClipboardText,
  loginErrorFromError,
  loginErrorHintKey,
  loginErrorMessage,
} from "@/features/auth/login-error"

describe("login-error", () => {
  it("classifies unauthorized, rate limit, network, and invalid input", () => {
    expect(classifyLoginError(new ApiError("invalid credentials", 401, "unauthorized"))).toBe("unauthorized")
    expect(classifyLoginError(new ApiError("too many login attempts", 429, "rate_limited"))).toBe("rate_limited")
    expect(classifyLoginError(new ApiError("invalid request body", 400, "invalid_request"))).toBe("invalid_input")
    expect(classifyLoginError(new Error("Failed to fetch"))).toBe("network")
    expect(classifyLoginError(new Error("timeout waiting"))).toBe("timeout")
    expect(classifyLoginError(new ApiError("token generation failed", 500, "internal_error"))).toBe("internal")
  })

  it("builds densified state, hint, title, and clipboard", () => {
    const state = loginErrorFromError(new ApiError("invalid credentials", 401, "unauthorized"), "登录失败")
    expect(state).toEqual({ message: "invalid credentials", code: "unauthorized" })
    expect(loginErrorHintKey(state.code)).toBe("auth.errorHintUnauthorized")
    expect(loginErrorHintKey("mystery")).toBe("auth.errorHintUnknown")
    expect(formatLoginErrorTitle((key, values) => {
      if (key === "auth.failedWithCode") return `failed:${values?.code}`
      return key
    }, state)).toBe("failed:unauthorized")
    expect(loginErrorClipboardText(state)).toBe("code: unauthorized\nerror: invalid credentials")
    expect(loginErrorClipboardText(null)).toBe("")
  })

  it("covers fallback codes, message patterns, and empty diagnostics", () => {
    expect(loginErrorHintKey()).toBe("auth.errorHintUnknown")
    expect(classifyLoginError(new ApiError("unauthorized", 418, "other"))).toBe("unauthorized")
    expect(classifyLoginError(new ApiError("rate", 429, "other"))).toBe("rate_limited")
    expect(classifyLoginError(new ApiError("bad", 400, "invalid_response"))).toBe("invalid_input")
    expect(classifyLoginError(new ApiError("slow", 408, "other"))).toBe("timeout")
    expect(classifyLoginError(new ApiError("slow", 504, "other"))).toBe("timeout")
    expect(classifyLoginError(new ApiError("down", 503, "other"))).toBe("network")
    expect(classifyLoginError(new ApiError("down", 418, "bad_gateway"))).toBe("network")
    expect(classifyLoginError(new ApiError("down", 418, "request_failed"))).toBe("network")
    expect(classifyLoginError(new ApiError("boom", 500, "other"))).toBe("internal")
    expect(classifyLoginError(new ApiError("mystery", 418, "other"))).toBe("unknown")
    expect(classifyLoginError(new ApiError("mystery", 418, ""))).toBe("unknown")
    for (const [message, code] of [
      ["rate limit", "rate_limited"], ["too many", "rate_limited"],
      ["unauthorized", "unauthorized"], ["wrong password", "unauthorized"],
      ["invalid json", "invalid_input"], ["required field", "invalid_input"],
      ["deadline exceeded", "timeout"], ["connection refused", "network"],
      ["offline", "network"], ["internal failure", "internal"],
    ] as const) expect(classifyLoginError(new Error(message))).toBe(code)
    expect(classifyLoginError(null)).toBe("unknown")
    expect(loginErrorMessage(new Error("  "), "fallback")).toBe("fallback")
    expect(loginErrorMessage("message", "fallback")).toBe("fallback")
    expect(formatLoginErrorTitle((key) => key, { message: "x", code: "unknown" })).toBe("auth.failed")
    expect(loginErrorClipboardText(undefined)).toBe("")
    expect(loginErrorClipboardText({ message: "  ", code: "unknown" })).toBe("")
    expect(loginErrorClipboardText({ message: "x", code: "" as never })).toBe("error: x")
  })
})
