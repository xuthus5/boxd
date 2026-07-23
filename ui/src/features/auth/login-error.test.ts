import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyLoginError,
  formatLoginErrorTitle,
  loginErrorClipboardText,
  loginErrorFromError,
  loginErrorHintKey,
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
})
