import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { ConfigHistoryPage } from "@/features/advanced/config-history-page"
import * as logExport from "@/features/observability/log-export"
import type { ConfigApplyEvent } from "@/lib/api/types"
import { renderApp } from "@/test/render"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function response(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function requestPath(input: string | URL | Request) {
  if (typeof input === "string") return input.split("?")[0]
  if (input instanceof URL) return input.pathname
  return new URL(input.url).pathname
}

function event(overrides: Partial<ConfigApplyEvent> = {}): ConfigApplyEvent {
  return {
    id: "event-1",
    source: "update",
    status: "applied",
    hash: "abcdef0123456789",
    size: 128,
    applied_at: "2026-07-26T00:00:00Z",
    ...overrides,
  }
}

function renderPage(events: ConfigApplyEvent[], historyFailure = false, deferRefresh = false) {
  let historyCalls = 0
  let releaseRefresh: (() => void) | undefined
  const fetchMock = vi.fn((input: string | URL | Request) => {
    const path = requestPath(input)
    if (path === "/api/config/apply-history") {
      historyCalls += 1
      if (historyFailure && historyCalls === 1) {
        return Promise.resolve(response({
          status: "error",
          data: null,
          error: { code: "internal_error", message: "history unavailable" },
          meta: null,
        }, 500))
      }
      if (deferRefresh && historyCalls === 2) {
        return new Promise<Response>((resolve) => { releaseRefresh = () => resolve(response({ events })) })
      }
      return Promise.resolve(response({ events }))
    }
    if (path === "/api/config/") return Promise.resolve(response({ log: { level: "info" } }))
    return Promise.resolve(response({}))
  })
  vi.stubGlobal("fetch", fetchMock)
  const view = renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ConfigHistoryPage />
    </QueryClientProvider>,
  )
  return { fetchMock, getHistoryCalls: () => historyCalls, releaseRefresh: () => releaseRefresh?.(), user: userEvent.setup(), view }
}

afterEach(() => {
  vi.clearAllMocks()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe("config history page", () => {
  it("renders every retained record and exposes restore actions", async () => {
    const events = Array.from({ length: 10 }, (_, index) => event({
      id: `event-${index}`,
      hash: `hash-${index}`,
      restorable: index === 0,
    }))
    renderPage(events)

    expect(await screen.findByRole("heading", { name: "配置历史" })).toBeInTheDocument()
    expect(screen.getByText("审计记录")).toBeInTheDocument()
    expect(screen.getAllByRole("listitem")).toHaveLength(10)
    expect(screen.getByText("显示 10 / 10 条记录")).toBeInTheDocument()
    expect(screen.getByText("总记录")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "可恢复" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "恢复此配置" })).toBeInTheDocument()
  })

  it("exports the currently visible records as JSON", async () => {
    const downloadSpy = vi.spyOn(logExport, "downloadTextFile").mockImplementation(() => {})
    const { user } = renderPage([event({ error: "line one\nline two" })])

    const fullError = await screen.findByText(/line one/)
    expect(fullError).toHaveClass("max-h-40", "whitespace-pre-wrap")
    expect(document.querySelector('time[datetime="2026-07-26T00:00:00Z"]')).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "导出 JSON" }))

    expect(downloadSpy).toHaveBeenCalledTimes(1)
    const [filename, payload] = downloadSpy.mock.calls[0] ?? []
    expect(filename).toMatch(/^boxd-config-history-.*\.json$/)
    expect(JSON.parse(String(payload))).toMatchObject({ count: 1, filter: "all", records: [{ id: "event-1" }] })
    expect(toast.success).toHaveBeenCalledWith("已导出 1 条记录")
  })

  it("reports export failures with a retry-safe error toast", async () => {
    vi.spyOn(logExport, "downloadTextFile").mockImplementation(() => { throw new Error("createObjectURL failed") })
    const { user } = renderPage([event()])
    await screen.findByText("显示 1 / 1 条记录")
    await user.click(screen.getByRole("button", { name: "导出 JSON" }))
    expect(toast.error).toHaveBeenCalledWith(
      "download_failed: createObjectURL failed",
      expect.objectContaining({ description: expect.any(String) }),
    )
  })

  it("filters by status, restorable state, and search text", async () => {
    const events = [
      event({ id: "applied", source: "raw", hash: "raw-hash", restorable: true }),
      event({ id: "validated", source: "validate_dns", status: "validated", hash: "dns-hash" }),
      event({ id: "rolled-back", status: "rolled_back", error: "restart failed" }),
      event({ id: "validate-failed", source: "validate_raw", status: "validate_failed", error_code: "config_invalid" }),
    ]
    const { user } = renderPage(events)

    await screen.findByText("显示 4 / 4 条记录")
    await user.click(screen.getByRole("button", { name: "失败 / 回滚" }))
    expect(screen.getByText("显示 2 / 4 条记录")).toBeInTheDocument()
    expect(screen.getAllByRole("listitem")).toHaveLength(2)
    await user.click(screen.getByRole("button", { name: "可恢复" }))
    expect(screen.getByText("显示 1 / 4 条记录")).toBeInTheDocument()
    expect(screen.getAllByRole("listitem")).toHaveLength(1)
    await user.type(screen.getByRole("textbox", { name: "搜索历史记录" }), "validate_dns")
    expect(await screen.findByText("没有匹配记录")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "清除筛选" })[0]!)
    expect(screen.getByText("显示 4 / 4 条记录")).toBeInTheDocument()
  })

  it("renders an empty state and refreshes the history request", async () => {
    const { user, fetchMock, getHistoryCalls } = renderPage([])
    expect(await screen.findByText("暂无配置历史")).toBeInTheDocument()
    expect(screen.getByText("保存或校验配置后，记录会显示在这里。")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "导出 JSON" })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "刷新历史" }))
    await waitFor(() => expect(getHistoryCalls()).toBe(2))
    expect(fetchMock).toHaveBeenCalledWith("/api/config/apply-history", expect.anything())
  })

  it("shows refresh progress while a history request is pending", async () => {
    const { user, releaseRefresh } = renderPage([], false, true)
    await screen.findByText("暂无配置历史")
    await user.click(screen.getByRole("button", { name: "刷新历史" }))
    expect(await screen.findByRole("button", { name: "刷新中…" })).toBeDisabled()
    releaseRefresh()
    expect(await screen.findByRole("button", { name: "刷新历史" })).toBeEnabled()
  })

  it("clears a no-match filter and retries a failed load", async () => {
    const { user } = renderPage([event()], true)
    expect(await screen.findByTestId("page-load-error")).toBeInTheDocument()
    expect(screen.getByText("history unavailable")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重试" }))
    expect(await screen.findByText("显示 1 / 1 条记录")).toBeInTheDocument()
    await user.type(screen.getByRole("textbox", { name: "搜索历史记录" }), "not-found")
    expect(await screen.findByText("没有匹配记录")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "清除筛选" })[0]!)
    expect(screen.getByText("显示 1 / 1 条记录")).toBeInTheDocument()
  })
})
