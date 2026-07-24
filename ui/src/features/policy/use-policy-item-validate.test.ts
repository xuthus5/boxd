import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, renderHook } from "@testing-library/react"
import { createElement, type ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/lib/api/endpoints", () => ({
  api: { config: { validate: vi.fn() } },
}))

vi.mock("@/features/config/config-hooks", () => ({
  useConfigQuery: vi.fn(),
}))

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), message: vi.fn() },
}))

import { useConfigQuery } from "@/features/config/config-hooks"
import { api } from "@/lib/api/endpoints"
import {
  policyItemErrorRelativePath,
  usePolicyItemValidate,
} from "@/features/policy/use-policy-item-validate"

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return createElement(QueryClientProvider, { client }, children)
}

describe("policyItemErrorRelativePath", () => {
  it("maps matching item paths and rejects others", () => {
    expect(policyItemErrorRelativePath("route.rules[0].outbound", "route", "rules", 0)).toBe("outbound")
    expect(policyItemErrorRelativePath("route.rules[1].outbound", "route", "rules", 0)).toBeUndefined()
    expect(policyItemErrorRelativePath("dns.servers[0].tag", "dns", "servers", -1)).toBe("tag")
    expect(policyItemErrorRelativePath("route.final", "route", "rules", 0)).toBeUndefined()
  })
})

describe("usePolicyItemValidate", () => {
  beforeEach(() => {
    vi.mocked(api.config.validate).mockReset()
    vi.mocked(useConfigQuery).mockReturnValue({
      data: {
        route: { final: "proxy", rules: [{ action: "reject" }], rule_set: [] },
        dns: { servers: [{ tag: "local" }], rules: [] },
      },
    } as ReturnType<typeof useConfigQuery>)
  })

  it("replaces an existing route rule in full config", async () => {
    vi.mocked(api.config.validate).mockResolvedValue({
      status: "ok", data: { valid: true }, error: null, meta: {},
    } as never)
    const { result } = renderHook(() => usePolicyItemValidate({
      section: "route",
      kind: "rules",
      index: 0,
      object: { action: "route", outbound: "proxy" },
    }), { wrapper })
    await act(async () => { await result.current.validate() })
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      route: expect.objectContaining({
        rules: [{ action: "route", outbound: "proxy" }],
      }),
    }), { source: "validate_route" })
  })

  it("appends a draft dns server when index is negative", async () => {
    vi.mocked(api.config.validate).mockResolvedValue({
      status: "ok", data: { valid: true }, error: null, meta: {},
    } as never)
    const { result } = renderHook(() => usePolicyItemValidate({
      section: "dns",
      kind: "servers",
      index: -1,
      object: { tag: "new", type: "local" },
    }), { wrapper })
    await act(async () => { await result.current.validate() })
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      dns: expect.objectContaining({
        servers: [{ tag: "local" }, { tag: "new", type: "local" }],
      }),
    }), { source: "validate_dns" })
  })

  it("skips when object is missing", async () => {
    const { result } = renderHook(() => usePolicyItemValidate({
      section: "route",
      kind: "rules",
      index: 0,
      object: null,
    }), { wrapper })
    let ok = true
    await act(async () => { ok = await result.current.validate() })
    expect(ok).toBe(false)
    expect(api.config.validate).not.toHaveBeenCalled()
  })

  it("passes validate source for route and dns", async () => {
    vi.mocked(api.config.validate).mockResolvedValue({
      status: "ok", data: { valid: true }, error: null, meta: { validated: true, applied: false },
    })
    const route = renderHook(() => usePolicyItemValidate({
      section: "route",
      kind: "rules",
      index: 0,
      object: { outbound: "direct" },
    }), { wrapper })
    await act(async () => { await route.result.current.validate() })
    expect(api.config.validate).toHaveBeenLastCalledWith(expect.any(Object), { source: "validate_route" })

    const dns = renderHook(() => usePolicyItemValidate({
      section: "dns",
      kind: "servers",
      index: 0,
      object: { tag: "local", address: "local" },
    }), { wrapper })
    await act(async () => { await dns.result.current.validate() })
    expect(api.config.validate).toHaveBeenLastCalledWith(expect.any(Object), { source: "validate_dns" })
  })

  it("densifies failures without custom reporter", async () => {
    vi.mocked(api.config.validate).mockRejectedValue(new Error("route.rules[0].outbound: missing"))
    const { result } = renderHook(() => usePolicyItemValidate({
      section: "route",
      kind: "rules",
      index: 0,
      object: { action: "route", outbound: "proxy" },
    }), { wrapper })
    let ok = true
    await act(async () => {
      ok = await result.current.validate()
    })
    expect(ok).toBe(false)
    const { toast } = await import("sonner")
    expect(toast.error).toHaveBeenCalled()
    const [message] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toMatch(/outbound|config_invalid|missing/)
  })
})
