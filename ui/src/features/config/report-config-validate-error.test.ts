import { describe, expect, it, vi } from "vitest"

vi.mock("@/features/config/config-save-error-actions", () => ({
  reportConfigSaveErrorToast: vi.fn(),
}))

import { reportConfigSaveErrorToast } from "@/features/config/config-save-error-actions"
import { reportConfigValidateError } from "@/features/config/report-config-validate-error"
import { ApiError } from "@/lib/api/client"

describe("reportConfigValidateError", () => {
  it("densifies pathful runtime failures", () => {
    const error = new ApiError("inbounds[0].listen_port: invalid", 400, "config_invalid_runtime")
    const state = reportConfigValidateError(error)
    expect(state.path).toBe("inbounds[0].listen_port")
    expect(state.code).toBe("config_invalid")
    expect(reportConfigSaveErrorToast).toHaveBeenCalledWith(expect.objectContaining({
      path: "inbounds[0].listen_port",
      code: "config_invalid",
    }))
  })

  it("falls back for plain errors", () => {
    const state = reportConfigValidateError(new Error("boom"))
    expect(state.message).toContain("boom")
    expect(reportConfigSaveErrorToast).toHaveBeenCalled()
  })
})
