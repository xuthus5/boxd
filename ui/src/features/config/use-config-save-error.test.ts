import { act, renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { toast } from "sonner"

import { useConfigSaveError } from "@/features/config/use-config-save-error"

describe("useConfigSaveError", () => {
  it("reports densified errors and rollbacks", () => {
    const { result } = renderHook(() => useConfigSaveError())
    act(() => {
      result.current.reportError(new Error("inbounds[0].tag: missing"))
    })
    expect(result.current.saveError?.path).toBe("inbounds[0].tag")
    expect(toast.error).toHaveBeenCalled()
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toContain("inbounds[0].tag")
    expect(options).toEqual(expect.objectContaining({
      description: expect.any(String),
      action: expect.objectContaining({ label: expect.any(String) }),
    }))
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
