import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/lib/api/endpoints", () => ({
  api: {
    config: {
      validate: vi.fn(),
    },
  },
}))

vi.mock("@/features/config/config-hooks", () => ({
  useConfigQuery: vi.fn(),
}))

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), message: vi.fn() },
}))

import { useConfigQuery } from "@/features/config/config-hooks"
import { api } from "@/lib/api/endpoints"
import { useProxyItemValidate } from "@/features/proxy/use-proxy-item-validate"

describe("useProxyItemValidate", () => {
  beforeEach(() => {
    vi.mocked(api.config.validate).mockReset()
    vi.mocked(useConfigQuery).mockReturnValue({
      data: {
        inbounds: [{ tag: "mixed-in", type: "mixed", listen_port: 1080 }],
        outbounds: [{ tag: "proxy", type: "direct" }],
      },
    } as ReturnType<typeof useConfigQuery>)
  })

  it("builds full config with draft item replaced at index", async () => {
    vi.mocked(api.config.validate).mockResolvedValue({
      status: "ok",
      data: { valid: true },
      error: null,
      meta: { validated: true, applied: false },
    } as never)
    const reportError = vi.fn()
    const { result } = renderHook(() => useProxyItemValidate({
      kind: "inbounds",
      index: 0,
      object: { tag: "mixed-in", type: "mixed", listen_port: 2080 },
      reportError,
    }))
    let ok = false
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(true)
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      inbounds: [{ tag: "mixed-in", type: "mixed", listen_port: 2080 }],
    }))
  })

  it("appends draft when index is negative", async () => {
    vi.mocked(api.config.validate).mockResolvedValue({
      status: "ok",
      data: { valid: true },
      error: null,
      meta: {},
    } as never)
    const { result } = renderHook(() => useProxyItemValidate({
      kind: "outbounds",
      index: -1,
      object: { tag: "new", type: "direct" },
      reportError: vi.fn(),
    }))
    await act(async () => {
      await result.current.validate()
    })
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      outbounds: [
        { tag: "proxy", type: "direct" },
        { tag: "new", type: "direct" },
      ],
    }))
  })

  it("skips when object or config is missing", async () => {
    const { result } = renderHook(() => useProxyItemValidate({
      kind: "inbounds",
      index: 0,
      object: null,
      reportError: vi.fn(),
    }))
    let ok = true
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(false)
    expect(api.config.validate).not.toHaveBeenCalled()
  })
})
