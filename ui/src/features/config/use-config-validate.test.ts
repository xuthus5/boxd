import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))
vi.mock("@/lib/api/endpoints", () => ({
  api: {
    config: {
      validate: vi.fn(),
    },
  },
}))

import { toast } from "sonner"

import { api } from "@/lib/api/endpoints"
import { useConfigValidate } from "@/features/config/use-config-validate"

describe("useConfigValidate", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("toasts success when dry-run validate passes", async () => {
    vi.mocked(api.config.validate).mockResolvedValue({
      status: "ok",
      data: { valid: true },
      error: null,
      meta: null,
    })
    const reportError = vi.fn()
    const { result } = renderHook(() => useConfigValidate({
      buildConfig: () => ({ log: { level: "info" } }),
      reportError,
    }))
    let ok = false
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(true)
    expect(api.config.validate).toHaveBeenCalled()
    expect(toast.success).toHaveBeenCalled()
    expect(reportError).not.toHaveBeenCalled()
  })

  it("reports densified errors and optional path callback", async () => {
    vi.mocked(api.config.validate).mockRejectedValue(new Error("route.final: missing"))
    const reportError = vi.fn().mockReturnValue({
      message: "route.final: missing",
      path: "route.final",
      code: "config_invalid",
    })
    const onReportedError = vi.fn()
    const clearSaveError = vi.fn()
    const { result } = renderHook(() => useConfigValidate({
      buildConfig: () => ({ route: {} }),
      reportError,
      clearSaveError,
      onReportedError,
    }))
    let ok = true
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(false)
    expect(clearSaveError).toHaveBeenCalled()
    expect(reportError).toHaveBeenCalled()
    expect(onReportedError).toHaveBeenCalledWith(expect.objectContaining({ path: "route.final" }))
  })

  it("skips when buildConfig returns null", async () => {
    const reportError = vi.fn()
    const { result } = renderHook(() => useConfigValidate({
      buildConfig: () => null,
      reportError,
    }))
    let ok = true
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(false)
    expect(api.config.validate).not.toHaveBeenCalled()
  })
})
