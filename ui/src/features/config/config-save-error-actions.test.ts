import { beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { toast } from "sonner"

import {
  formatConfigSaveErrorToast,
  reportConfigSaveErrorToast,
} from "@/features/config/config-save-error-actions"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"

describe("config save error actions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("formats toast message with code when present", () => {
    expect(formatConfigSaveErrorToast({
      message: "inbounds[0].tag: missing",
      code: "config_invalid",
    })).toBe("config_invalid: inbounds[0].tag: missing")
    expect(formatConfigSaveErrorToast({ message: "plain" })).toBe("plain")
    expect(formatConfigSaveErrorToast({ message: "plain", code: "unknown" })).toBe("plain")
  })

  it("reports densified toast with hint and clipboard action", () => {
    const state: ConfigSaveErrorState = {
      message: "inbounds[0].listen_port: invalid",
      path: "inbounds[0].listen_port",
      code: "config_invalid",
      section: "inbounds",
    }
    reportConfigSaveErrorToast(state)
    expect(toast.error).toHaveBeenCalled()
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toContain("config_invalid")
    expect(options).toEqual(expect.objectContaining({
      description: expect.any(String),
      action: expect.objectContaining({ label: expect.any(String) }),
    }))
  })

  it("uses an empty action when the diagnostic payload is unavailable", () => {
    reportConfigSaveErrorToast({ message: "", code: "unknown" })
    const [, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(options?.action).toBeUndefined()
  })
})
