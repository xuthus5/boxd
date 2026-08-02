import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, cleanup, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { RuleSetHealthCard } from "@/features/dashboard/rule-set-health-card"
import { copyText } from "@/lib/clipboard"
import type { RuleSetStatusItem } from "@/lib/api/types"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), warning: vi.fn(), error: vi.fn() },
}))
vi.mock("@/lib/clipboard", () => ({
  copyText: vi.fn().mockResolvedValue(undefined),
}))

// 相对当前时间的更新时间：模块加载时取 1 小时前，避免因真实时钟推移而过期（stale）。
const recentTimestamp = () => new Date(Date.now() - 60 * 60 * 1000).toISOString()

const status: RuleSetStatusItem[] = [
  {
    tag: "geo-cn",
    type: "remote",
    builtin: true,
    updatable: true,
    update_interval: "24h",
    file_size: 2048,
    last_updated: recentTimestamp(),
  },
  {
    tag: "missing-set",
    type: "local",
    builtin: true,
    updatable: true,
    file_size: 0,
  },
]

const autoUpdate = { enabled: true, interval: "24h" }

function renderCard(options: {
  logs?: { level: string; message: string; timestamp?: string }[]
  failStatus?: boolean
  failAutoUpdate?: boolean
  statusItems?: RuleSetStatusItem[]
  updateMode?: "success" | "partial" | "error"
  appLogError?: string
  appLogStatus?: string
} = {}) {
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const path = typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname
    if (path === "/api/config/rule-sets/status") {
      if (options.failStatus) {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error", data: null, error: { code: "internal_error", message: "status unavailable" }, meta: null,
        }), { status: 500 }))
      }
      return Promise.resolve(new Response(JSON.stringify(options.statusItems ?? status)))
    }
    if (path === "/api/config/rule-sets/auto-update") {
      if (options.failAutoUpdate) {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error", data: null, error: { code: "internal_error", message: "auto update unavailable" }, meta: null,
        }), { status: 500 }))
      }
      return Promise.resolve(new Response(JSON.stringify(autoUpdate)))
    }
    if (path === "/api/config/rule-sets/update" && init?.method === "POST") {
      if (options.updateMode === "error") {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error", data: null, error: { code: "bad_gateway", message: "upstream failed" }, meta: null,
        }), { status: 502 }))
      }
      const partial = options.updateMode === "partial"
      return Promise.resolve(new Response(JSON.stringify({
        status: partial ? "partial" : "ok",
        data: {
          results: partial
            ? [{ tag: "geo-cn", type: "remote", ok: true }, { tag: "missing-set", type: "local", ok: false, error: "unexpected status 500", error_code: "http_status" }]
            : [{ tag: "geo-cn", type: "remote", ok: true }],
          updated_count: 1,
          failed_count: partial ? 1 : 0,
          skipped_count: 0,
          restarted: false,
        },
        error: null,
        meta: null,
      })))
    }
    return Promise.resolve(new Response(JSON.stringify({})))
  })
  vi.stubGlobal("fetch", fetchMock)
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RuleSetHealthCard
        appLogs={options.logs}
        appLogError={options.appLogError}
        appLogStatus={options.appLogStatus ?? "open"}
        onReconnectAppLogs={vi.fn()}
      />
    </QueryClientProvider>,
    "/dashboard",
  )
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
  sessionStore.clear()
})

describe("RuleSetHealthCard", () => {
  it("shows freshness, automatic update failures, and deep links", async () => {
    const fetchMock = renderCard({ logs: [{
      level: "warn",
      message: 'WARN ruleset auto update finished updated=1 failed=2 skipped=0 failed_details=[{"tag":"loyalsoldier-proxy","code":"network"},{"tag":"loyalsoldier-reject","code":"http_status"}]',
      timestamp: "2026-07-26T11:00:00Z",
    }] })

    expect(await screen.findByText("规则集运行异常")).toBeInTheDocument()
    expect(screen.getByText("2 个规则集")).toBeInTheDocument()
    expect(screen.getByText("自动更新失败 2 个")).toBeInTheDocument()
    expect(screen.getByText("loyalsoldier-proxy")).toBeInTheDocument()
    expect(screen.getByText("network")).toBeInTheDocument()
    expect(screen.getByText("无法拉取规则集，请检查网络、DNS 或 download_detour。")).toBeInTheDocument()
    expect(screen.getByText(/最近文件\/缓存/)).toBeInTheDocument()
    expect(screen.getByText("missing-set")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "打开规则集 missing-set" })).toHaveAttribute(
      "href",
      "/policy/route?path=route.rule_set%5B1%5D",
    )
    expect(screen.getByRole("link", { name: "查看应用日志" })).toHaveAttribute("href", "/observability/logs?tab=application")
    await userEvent.setup().click(screen.getByRole("button", { name: "刷新规则集状态" }))
    await waitFor(() => {
      const paths = fetchMock.mock.calls.map(([input]) => typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname)
      expect(paths.filter((path) => path === "/api/config/rule-sets/status")).toHaveLength(2)
      expect(paths.filter((path) => path === "/api/config/rule-sets/auto-update")).toHaveLength(2)
    })
  })

  it("updates all rule sets and refreshes query data", async () => {
    const fetchMock = renderCard()
    const user = userEvent.setup()
    await screen.findByRole("button", { name: "更新可更新规则集" })
    await user.click(screen.getByRole("button", { name: "更新可更新规则集" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/rule-sets/update", expect.objectContaining({ method: "POST" })))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith(expect.stringContaining("已更新 1 个规则集")))
  })

  it("reports partial and request-level update failures", async () => {
    const user = userEvent.setup()
    renderCard({ updateMode: "partial" })
    await user.click(await screen.findByRole("button", { name: "更新可更新规则集" }))
    await waitFor(() => expect(toast.warning).toHaveBeenCalledWith(expect.stringContaining("失败 1 个"), expect.any(Object)))
    const [, warningOptions] = vi.mocked(toast.warning).mock.calls.at(-1) ?? []
    act(() => warningOptions?.action?.onClick?.())
    await waitFor(() => expect(copyText).toHaveBeenCalled())
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("更新错误已复制"))
    vi.mocked(copyText).mockRejectedValueOnce(new Error("clipboard unavailable"))
    act(() => warningOptions?.action?.onClick?.())
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("复制更新错误失败"))

    cleanup()
    vi.clearAllMocks()
    renderCard({ updateMode: "error" })
    await user.click(await screen.findByRole("button", { name: "更新可更新规则集" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("upstream failed"), expect.any(Object)))
  })

  it("keeps rule-set status visible when the app-log stream reconnects", async () => {
    renderCard({ appLogError: "SSE request failed with status 503", appLogStatus: "reconnecting" })
    expect(await screen.findByText("需要关注")).toBeInTheDocument()
    expect(screen.getByText("SSE request failed with status 503")).toBeInTheDocument()
  })

  it("renders healthy, empty, and auto-update query states", async () => {
    renderCard({
      statusItems: status.slice(0, 1),
      logs: [{ level: "info", message: "ruleset auto update finished updated=1 failed=0 skipped=0", timestamp: "invalid-time" }],
    })
    expect(await screen.findByText("规则集正常")).toBeInTheDocument()
    expect(screen.getByText("所有可更新规则集都有可用文件或缓存。")).toBeInTheDocument()
    expect(screen.getByText("invalid-time")).toBeInTheDocument()

    cleanup()
    renderCard({ statusItems: [] })
    expect(await screen.findByText("尚未配置规则集")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "更新可更新规则集" })).toBeDisabled()

    cleanup()
    renderCard({ failAutoUpdate: true })
    expect(await screen.findByText("auto update unavailable")).toBeInTheDocument()
  })

  it("renders loading and query error states", async () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)))
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const loadingView = renderApp(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <RuleSetHealthCard />
      </QueryClientProvider>,
      "/dashboard",
    )
    expect(screen.getByTestId("rule-set-health-card").querySelector('[data-slot="skeleton"]')).toBeInTheDocument()
    loadingView.unmount()

    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "internal_error", message: "status unavailable" }, meta: null,
    }), { status: 500 })))
    renderCard({ failStatus: true })
    expect(await screen.findByText("status unavailable")).toBeInTheDocument()
  })
})
