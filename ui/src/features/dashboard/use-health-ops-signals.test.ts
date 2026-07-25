import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, renderHook, waitFor } from "@testing-library/react"
import { createElement, type ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/lib/api/endpoints", () => ({
  api: {
    subscriptions: { list: vi.fn() },
    nodes: { list: vi.fn(), testHistory: vi.fn() },
    config: { applyHistory: vi.fn() },
  },
}))

import { useHealthOpsSignals } from "@/features/dashboard/use-health-ops-signals"
import { api } from "@/lib/api/endpoints"

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return createElement(QueryClientProvider, { client }, children)
}

function configureSuccess() {
  vi.mocked(api.subscriptions.list).mockResolvedValue([
    { id: "bad", name: "bad", url: "https://example.com", interval_min: 60, last_updated: "now", outbounds: [], error: "timeout" },
  ])
  vi.mocked(api.nodes.list).mockResolvedValue([
    { tag: "bad-node", type: "vless", server: "example.com", port: 443, source: "import" },
  ])
  vi.mocked(api.nodes.testHistory).mockResolvedValue({
    history: { "bad-node": { tcp: [{ timestamp: "1", success: false }] } },
  })
  vi.mocked(api.config.applyHistory).mockResolvedValue({
    events: [{ id: "1", source: "update", status: "validate_failed", hash: "h", size: 1, applied_at: "now" }],
  })
}

beforeEach(() => {
  vi.resetAllMocks()
  configureSuccess()
})

describe("useHealthOpsSignals", () => {
  it("combines all successful query payloads into health signals", async () => {
    const { result } = renderHook(() => useHealthOpsSignals(), { wrapper })
    await waitFor(() => expect(result.current.signals.applyFailures).toBe(1))
    expect(result.current.signals.failedSubscriptions).toBe(1)
    expect(result.current.signals.failedNodes).toBe(1)
    expect(result.current.signals.problemNodes).toBe(1)
    expect(result.current.queryError).toBeUndefined()
    expect(result.current.queryScope).toBeUndefined()
    expect(result.current.onRetry).toBeUndefined()
  })

  it("ignores non-array subscription and apply-event payloads", async () => {
    vi.mocked(api.subscriptions.list).mockResolvedValue({} as never)
    vi.mocked(api.config.applyHistory).mockResolvedValue({ events: {} } as never)
    vi.mocked(api.nodes.list).mockResolvedValue([])
    vi.mocked(api.nodes.testHistory).mockResolvedValue({ history: {} })
    const { result } = renderHook(() => useHealthOpsSignals(), { wrapper })
    await waitFor(() => expect(result.current.queryError).toBeUndefined())
    expect(result.current.signals).toMatchObject({
      failedSubscriptions: 0,
      failedSubscriptionItems: [],
      problemNodes: 0,
      problemNodeItems: [],
      applyFailures: 0,
    })
  })

  it.each([
    ["subscriptions", "health-subscriptions"],
    ["nodes", "health-nodes"],
    ["history", "health-node-history"],
    ["apply", "health-apply-history"],
  ])("reports and retries the %s query", async (kind, scope) => {
    const errors = {
      subscriptions: () => vi.mocked(api.subscriptions.list).mockRejectedValue(new Error("subscriptions failed")),
      nodes: () => vi.mocked(api.nodes.list).mockRejectedValue(new Error("nodes failed")),
      history: () => vi.mocked(api.nodes.testHistory).mockRejectedValue(new Error("history failed")),
      apply: () => vi.mocked(api.config.applyHistory).mockRejectedValue(new Error("apply failed")),
    }
    errors[kind as keyof typeof errors]()
    const { result } = renderHook(() => useHealthOpsSignals(), { wrapper })
    await waitFor(() => expect(result.current.queryScope).toBe(scope))
    expect(result.current.queryError).toBeInstanceOf(Error)
    expect(result.current.onRetry).toBeTypeOf("function")
    const before = {
      subscriptions: vi.mocked(api.subscriptions.list).mock.calls.length,
      nodes: vi.mocked(api.nodes.list).mock.calls.length,
      history: vi.mocked(api.nodes.testHistory).mock.calls.length,
      apply: vi.mocked(api.config.applyHistory).mock.calls.length,
    }
    act(() => result.current.onRetry?.())
    await waitFor(() => {
      const current = {
        subscriptions: vi.mocked(api.subscriptions.list).mock.calls.length,
        nodes: vi.mocked(api.nodes.list).mock.calls.length,
        history: vi.mocked(api.nodes.testHistory).mock.calls.length,
        apply: vi.mocked(api.config.applyHistory).mock.calls.length,
      }
      expect(current[kind as keyof typeof current]).toBeGreaterThan(before[kind as keyof typeof before])
    })
  })

  it("retries every failed query when multiple sources fail", async () => {
    vi.mocked(api.subscriptions.list).mockRejectedValue(new Error("subscriptions failed"))
    vi.mocked(api.nodes.list).mockRejectedValue(new Error("nodes failed"))
    vi.mocked(api.nodes.testHistory).mockRejectedValue(new Error("history failed"))
    vi.mocked(api.config.applyHistory).mockRejectedValue(new Error("apply failed"))
    const { result } = renderHook(() => useHealthOpsSignals(), { wrapper })
    await waitFor(() => expect(result.current.queryScope).toBe("health-subscriptions"))
    act(() => result.current.onRetry?.())
    await waitFor(() => {
      expect(api.subscriptions.list).toHaveBeenCalledTimes(2)
      expect(api.nodes.list).toHaveBeenCalledTimes(2)
      expect(api.nodes.testHistory).toHaveBeenCalledTimes(2)
      expect(api.config.applyHistory).toHaveBeenCalledTimes(2)
    })
  })
})
