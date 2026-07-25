import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, renderHook, waitFor } from "@testing-library/react"
import { createElement, type ReactNode } from "react"
import { I18nextProvider } from "react-i18next"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }))
vi.mock("@/lib/api/endpoints", () => ({
  api: { nodes: { groups: vi.fn(), select: vi.fn() } },
}))
vi.mock("@/features/proxy/copy-tag-button", () => ({ copyText: vi.fn() }))
vi.mock("@/features/dashboard/proxy-delay", () => ({
  delayBatchFailureClipboardText: vi.fn(),
  delayBatchToastTone: vi.fn(),
  delayErrorHintKey: vi.fn(),
  delayFailureFromError: vi.fn(),
  delayRequestErrorClipboardText: vi.fn(),
  formatDelayBatchMessage: vi.fn(),
  formatDelayRequestErrorToast: vi.fn(),
  measureGroupDelays: vi.fn(),
  pickPrimaryGroup: vi.fn(),
  summarizeDelays: vi.fn(),
}))

import { toast } from "sonner"

import { useProxySelector } from "@/features/dashboard/use-proxy-selector"
import * as delay from "@/features/dashboard/proxy-delay"
import { copyText } from "@/features/proxy/copy-tag-button"
import { i18n } from "@/i18n"
import { api } from "@/lib/api/endpoints"

const selector = { type: "selector", tag: "proxy", now: "a", all: ["a", "b"] }

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return createElement(
    I18nextProvider,
    { i18n },
    createElement(QueryClientProvider, { client }, children),
  )
}

function configureHelpers() {
  vi.mocked(delay.pickPrimaryGroup).mockImplementation((groups) => groups.find((item) => item.tag === "proxy") ?? null)
  vi.mocked(delay.summarizeDelays).mockImplementation((values) => {
    const entries = Object.values(values)
    const failed = entries.filter((value) => typeof value !== "number").length
    return {
      total: entries.length,
      ok: entries.length - failed,
      failed,
      failedSamples: failed ? [{ tag: "b", error: "failed", code: "timeout" }] : [],
    }
  })
  vi.mocked(delay.formatDelayBatchMessage).mockImplementation((summary) => `batch ${summary.ok}/${summary.total}`)
  vi.mocked(delay.delayBatchToastTone).mockImplementation((summary) => {
    if (summary.failed > 0 && summary.ok === 0) return "error"
    if (summary.failed > 0) return "warning"
    return "success"
  })
  vi.mocked(delay.delayBatchFailureClipboardText).mockImplementation((summary) => summary.failed ? "delay payload" : "")
  vi.mocked(delay.delayErrorHintKey).mockReturnValue("dashboard.delayHint")
  vi.mocked(delay.delayFailureFromError).mockReturnValue({ failed: true, error: "probe failed", code: "timeout" })
  vi.mocked(delay.delayRequestErrorClipboardText).mockReturnValue("request payload")
  vi.mocked(delay.formatDelayRequestErrorToast).mockImplementation((_error, fallback) => fallback)
}

afterEach(() => {
  vi.clearAllMocks()
})

beforeEach(() => {
  configureHelpers()
  vi.mocked(api.nodes.groups).mockResolvedValue({ groups: [selector] })
  vi.mocked(api.nodes.select).mockResolvedValue({ selected: "b" })
  vi.mocked(delay.measureGroupDelays).mockResolvedValue({ a: 12, b: 24 })
  vi.mocked(copyText).mockResolvedValue(undefined)
})

describe("useProxySelector", () => {
  it("reports missing-group mutations and still handles an empty probe", async () => {
    vi.mocked(api.nodes.groups).mockResolvedValue({ groups: [] })
    const { result } = renderHook(() => useProxySelector(), { wrapper })
    await waitFor(() => expect(result.current.query.isSuccess).toBe(true))
    expect(result.current.group).toBeNull()
    expect(result.current.members).toEqual([])

    act(() => result.current.select("a"))
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    const [, selectOptions] = vi.mocked(toast.error).mock.calls[0] ?? []
    expect(selectOptions?.action).toBeDefined()
    act(() => selectOptions?.action?.onClick?.())
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("请求错误已复制"))
    vi.mocked(copyText).mockRejectedValueOnce(new Error("clipboard unavailable"))
    act(() => selectOptions?.action?.onClick?.())
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("复制请求错误失败"))

    act(() => result.current.probe())
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("batch 0/0"))
    expect(delay.measureGroupDelays).not.toHaveBeenCalled()
  })

  it("selects a member and reports a successful delay probe", async () => {
    const { result } = renderHook(() => useProxySelector(), { wrapper })
    await waitFor(() => expect(result.current.group?.tag).toBe("proxy"))

    act(() => result.current.select("b"))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("已切换当前出口"))
    expect(api.nodes.select).toHaveBeenCalledWith("proxy", "b")

    act(() => result.current.probe())
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("batch 2/2"))
    expect(result.current.delays).toEqual({ a: 12, b: 24 })
    expect(delay.measureGroupDelays).toHaveBeenCalledWith("proxy", ["a", "b"])
  })

  it("shows warning and error delay tones with copy actions", async () => {
    vi.mocked(delay.measureGroupDelays)
      .mockResolvedValueOnce({ a: 12, b: { failed: true, error: "timeout", code: "timeout" } })
      .mockResolvedValueOnce({ a: { failed: true, error: "timeout", code: "timeout" } })
    vi.mocked(delay.summarizeDelays)
      .mockReturnValueOnce({ total: 2, ok: 1, failed: 1, failedSamples: [{ tag: "b", error: "timeout", code: "timeout" }] })
      .mockReturnValueOnce({ total: 1, ok: 0, failed: 1, failedSamples: [{ tag: "a", error: "timeout", code: "timeout" }] })
    vi.mocked(delay.formatDelayBatchMessage).mockReturnValueOnce("warning batch").mockReturnValueOnce("error batch")
    vi.mocked(delay.delayBatchToastTone).mockReturnValueOnce("warning").mockReturnValueOnce("error")
    const { result } = renderHook(() => useProxySelector(), { wrapper })
    await waitFor(() => expect(result.current.group).not.toBeNull())

    act(() => result.current.probe())
    await waitFor(() => expect(toast.warning).toHaveBeenCalledWith("warning batch", expect.anything()))
    const [, warningOptions] = vi.mocked(toast.warning).mock.calls[0] ?? []
    act(() => warningOptions?.action?.onClick?.())
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("延迟错误已复制"))
    vi.mocked(copyText).mockRejectedValueOnce(new Error("clipboard unavailable"))
    act(() => warningOptions?.action?.onClick?.())
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("复制延迟错误失败"))

    act(() => result.current.probe())
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("error batch", expect.anything()))
  })

  it("handles mutation failures and exposes pending state", async () => {
    let resolveSelect: (value: { selected: string }) => void = () => undefined
    vi.mocked(api.nodes.select).mockImplementation(() => new Promise((resolve) => { resolveSelect = resolve }))
    let rejectProbe: (error: Error) => void = () => undefined
    vi.mocked(delay.measureGroupDelays).mockImplementation(() => new Promise((_resolve, reject) => { rejectProbe = reject }))
    const { result } = renderHook(() => useProxySelector(), { wrapper })
    await waitFor(() => expect(result.current.group).not.toBeNull())

    act(() => result.current.select("b"))
    await waitFor(() => expect(result.current.selecting).toBe(true))
    resolveSelect({ selected: "b" })
    await waitFor(() => expect(result.current.selecting).toBe(false))

    act(() => result.current.probe())
    await waitFor(() => expect(result.current.probing).toBe(true))
    rejectProbe(new Error("probe failed"))
    await waitFor(() => expect(result.current.probing).toBe(false))
    expect(toast.error).toHaveBeenCalledWith("超时", expect.anything())
    const [, options] = vi.mocked(toast.error).mock.calls.at(-1) ?? []
    act(() => options?.action?.onClick?.())
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("延迟错误已复制"))
    vi.mocked(copyText).mockRejectedValueOnce(new Error("clipboard unavailable"))
    act(() => options?.action?.onClick?.())
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("复制延迟错误失败"))
  })
})
