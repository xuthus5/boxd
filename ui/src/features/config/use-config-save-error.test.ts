import { act, renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { useConfigSaveError } from "@/features/config/use-config-save-error"

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

describe("useConfigSaveError", () => {
  it("reports errors and rollbacks", () => {
    const { result } = renderHook(() => useConfigSaveError())
    act(() => {
      result.current.reportError(new Error("inbounds[0].tag: missing"))
    })
    expect(result.current.saveError?.path).toBe("inbounds[0].tag")
    act(() => {
      result.current.reportRollback({ error: null }, "配置已回滚")
    })
    expect(result.current.saveError?.message).toBe("配置已回滚")
    act(() => {
      result.current.clearSaveError()
    })
    expect(result.current.saveError).toBeNull()
  })
})
