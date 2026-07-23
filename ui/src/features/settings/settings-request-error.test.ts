import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifySettingsRequestError,
  formatSettingsRequestErrorToast,
  settingsRequestErrorClipboardText,
  settingsRequestErrorHintKey,
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
})
