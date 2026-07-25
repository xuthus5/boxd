import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifySettingsRequestError,
  formatSettingsRequestErrorToast,
  settingsRequestErrorClipboardText,
  settingsRequestErrorHintKey,
  settingsRequestErrorMessage,
} from "@/features/settings/settings-request-error"

describe("settings request error diagnostics", () => {
  it("maps codes to hints", () => {
    expect(settingsRequestErrorHintKey("invalid_input")).toBe("settings.errorHintInvalid")
    expect(settingsRequestErrorHintKey("nope")).toBe("settings.errorHintUnknown")
  })

  it("classifies API and message failures", () => {
    expect(classifySettingsRequestError(new ApiError("bad password", 400, "invalid_request"))).toBe("invalid_input")
    expect(classifySettingsRequestError(new ApiError("failed to create backup", 500, "internal_error"))).toBe("internal")
    expect(classifySettingsRequestError(new ApiError("nope", 401, "unauthorized"))).toBe("unauthorized")
    expect(classifySettingsRequestError(new Error("failed to fetch"))).toBe("network")
    expect(classifySettingsRequestError(new Error("i/o timeout"))).toBe("timeout")
  })

  it("formats toast and clipboard payloads", () => {
    expect(formatSettingsRequestErrorToast(new ApiError("boom", 500, "internal_error"), "fallback")).toBe("internal: boom")
    expect(formatSettingsRequestErrorToast(new Error("mystery"), "fallback")).toBe("mystery")
    const payload = settingsRequestErrorClipboardText(new ApiError("bad", 400, "invalid_request"), {
      scope: "password",
      field: "newPassword",
    })
    expect(payload).toContain("scope: password")
    expect(payload).toContain("code: invalid_input")
    expect(payload).toContain("error: bad")
  })

  it("covers fallback codes, validation phrases, and empty payloads", () => {
    expect(settingsRequestErrorHintKey()).toBe("settings.errorHintUnknown")
    expect(classifySettingsRequestError(new ApiError("down", 503, "other"))).toBe("unavailable")
    expect(classifySettingsRequestError(new ApiError("slow", 408, "other"))).toBe("timeout")
    expect(classifySettingsRequestError(new ApiError("slow", 504, "other"))).toBe("timeout")
    expect(classifySettingsRequestError(new ApiError("down", 418, "bad_gateway"))).toBe("network")
    expect(classifySettingsRequestError(new ApiError("down", 418, "request_failed"))).toBe("network")
    expect(classifySettingsRequestError(new ApiError("mystery", 418, "other"))).toBe("unknown")
    expect(classifySettingsRequestError(new ApiError("mystery", 418, ""))).toBe("unknown")
    for (const [message, code] of [
      ["current password is wrong", "unauthorized"], ["wrong password", "unauthorized"],
      ["required field", "invalid_input"], ["must be stronger", "invalid_input"],
      ["too short", "invalid_input"], ["too long", "invalid_input"], ["parse failed", "invalid_input"],
      ["deadline exceeded", "timeout"], ["not available", "unavailable"],
      ["network offline", "network"], ["connection refused", "network"],
      ["internal failure", "internal"], ["failed to save", "internal"],
    ] as const) expect(classifySettingsRequestError(new Error(message))).toBe(code)
    expect(classifySettingsRequestError(null)).toBe("unknown")
    expect(settingsRequestErrorMessage(new Error("  "), "fallback")).toBe("fallback")
    expect(settingsRequestErrorMessage("text", "fallback")).toBe("fallback")
    expect(formatSettingsRequestErrorToast(null, "fallback")).toBe("fallback")
    expect(settingsRequestErrorClipboardText(null)).toBe("")
    expect(settingsRequestErrorClipboardText(new Error("boom"), { scope: "  ", field: "  " })).toContain("error: boom")
  })
})
