import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, renderHook } from "@testing-library/react"
import { createElement, type ReactNode } from "react"
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

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return createElement(QueryClientProvider, { client }, children)
}


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
    }), { wrapper })
    let ok = false
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(true)
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      inbounds: [{ tag: "mixed-in", type: "mixed", listen_port: 2080 }],
    }), { source: "validate_inbounds" })
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
    }), { wrapper })
    await act(async () => {
      await result.current.validate()
    })
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      outbounds: [
        { tag: "proxy", type: "direct" },
        { tag: "new", type: "direct" },
      ],
    }), { source: "validate_outbounds" })
  })

  it("skips when object or config is missing", async () => {
    const { result } = renderHook(() => useProxyItemValidate({
      kind: "inbounds",
      index: 0,
      object: null,
      reportError: vi.fn(),
    }), { wrapper })
    let ok = true
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(false)
    expect(api.config.validate).not.toHaveBeenCalled()
  })

  it("passes validate source for inbounds and outbounds", async () => {
    vi.mocked(api.config.validate).mockResolvedValue({
      status: "ok", data: { valid: true }, error: null, meta: { validated: true, applied: false },
    })
    const inbound = renderHook(() => useProxyItemValidate({
      kind: "inbounds",
      index: 0,
      object: { type: "mixed", tag: "in" },
    }), { wrapper })
    await act(async () => { await inbound.result.current.validate() })
    expect(api.config.validate).toHaveBeenLastCalledWith(expect.any(Object), { source: "validate_inbounds" })

    const outbound = renderHook(() => useProxyItemValidate({
      kind: "outbounds",
      index: 0,
      object: { type: "direct", tag: "out" },
    }), { wrapper })
    await act(async () => { await outbound.result.current.validate() })
    expect(api.config.validate).toHaveBeenLastCalledWith(expect.any(Object), { source: "validate_outbounds" })
  })

  it("densifies failures without custom reporter", async () => {
    vi.mocked(api.config.validate).mockRejectedValue(new Error("inbounds[0].listen_port: invalid"))
    const { result } = renderHook(() => useProxyItemValidate({
      kind: "inbounds",
      index: 0,
      object: { tag: "mixed-in", type: "mixed", listen_port: 1080 },
    }), { wrapper })
    let ok = true
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(false)
    const { toast } = await import("sonner")
    expect(toast.error).toHaveBeenCalled()
    const [message] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toMatch(/listen_port|config_invalid|invalid/)
  })
})
