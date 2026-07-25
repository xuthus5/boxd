import { beforeEach, describe, expect, it, vi } from "vitest"

import { ApiError } from "@/lib/api/client"

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { toast } from "sonner"

import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"

describe("settings request error actions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("reports densified request failures with clipboard action", () => {
    const t = (key: string) => key
    reportSettingsRequestError(new ApiError("failed to create backup", 500, "internal_error"), t, {
      scope: "backup-export",
      fallback: "settings.backupExportFailed",
    })
    expect(toast.error).toHaveBeenCalled()
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toContain("internal")
    expect(options).toEqual(expect.objectContaining({
      description: "settings.errorHintInternal",
      action: expect.objectContaining({ label: "settings.copyRequestError" }),
    }))
  })

  it("uses the default fallback and omits an empty diagnostic action", () => {
    reportSettingsRequestError(new Error(""), (key) => key)
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(message).toBe("settings.requestFailed")
    expect(options?.action).toBeUndefined()
  })
})
