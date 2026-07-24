import { act, renderHook } from "@testing-library/react"
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
    }))
    await act(async () => { await result.current.validate() })
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      route: expect.objectContaining({
        rules: [{ action: "route", outbound: "proxy" }],
      }),
    }))
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
    }))
    await act(async () => { await result.current.validate() })
    expect(api.config.validate).toHaveBeenCalledWith(expect.objectContaining({
      dns: expect.objectContaining({
        servers: [{ tag: "local" }, { tag: "new", type: "local" }],
      }),
    }))
  })

  it("skips when object is missing", async () => {
    const { result } = renderHook(() => usePolicyItemValidate({
      section: "route",
      kind: "rules",
      index: 0,
      object: null,
    }))
    let ok = true
    await act(async () => { ok = await result.current.validate() })
    expect(ok).toBe(false)
    expect(api.config.validate).not.toHaveBeenCalled()
  })
})
