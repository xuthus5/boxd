import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { ConfigApplyTimelineCard } from "@/features/dashboard/config-apply-timeline-card"
import { AuthProvider } from "@/features/auth/auth-context"
import { PreferencesProvider } from "@/features/preferences/preferences-provider"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"
import * as copy from "@/features/proxy/copy-tag-button"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function renderCard() {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <AuthProvider>
        <PreferencesProvider>
          <ConfigApplyTimelineCard />
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  )
}

describe("ConfigApplyTimelineCard", () => {
  it("renders empty state", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({ events: [] })))))
    renderCard()
    expect(await screen.findByText("配置应用记录")).toBeInTheDocument()
    expect(screen.getByText("暂无应用记录")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "打开完整配置" })).toHaveAttribute("href", "/advanced/raw")
  })

  it("renders applied and rolled back events", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      events: [
        {
          id: "1",
          source: "raw",
          status: "rolled_back",
          hash: "abcdef0123456789",
          size: 2048,
          error: "restart failed after config save",
          applied_at: "2026-07-23T12:00:00Z",
        },
        {
          id: "2",
          source: "update",
          status: "applied",
          hash: "1111222233334444",
          size: 512,
          applied_at: "2026-07-23T11:00:00Z",
        },
      ],
    })))))
    renderCard()
    expect(await screen.findByRole("link", { name: "完整配置保存" })).toHaveAttribute("href", "/advanced/raw")
    expect(screen.getByRole("link", { name: "结构化保存" })).toHaveAttribute("href", "/advanced/raw")
    expect(screen.getByText("结构化保存")).toBeInTheDocument()
    expect(screen.getByText("已回滚")).toBeInTheDocument()
    expect(screen.getByText("已应用")).toBeInTheDocument()
    expect(screen.getByText("restart failed after config save")).toBeInTheDocument()
    expect(screen.getByText("restart_failed")).toBeInTheDocument()
    expect(screen.getByText(/abcdef01/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制错误: 完整配置保存" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "打开来源: 完整配置保存" })).toHaveAttribute("href", "/advanced/raw")
  })

  it("labels restored snapshots distinctly", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      events: [{
        id: "restore-1",
        source: "restore",
        status: "applied",
        hash: "abcdef0123456789",
        size: 512,
        applied_at: "2026-07-23T12:00:00Z",
      }],
    })))))
    renderCard()
    expect(await screen.findByRole("link", { name: "历史配置恢复" })).toHaveAttribute("href", "/advanced/raw")
  })

  it("marks the current config without offering a no-op restore", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      events: [{
        id: "current-1",
        source: "update",
        status: "applied",
        hash: "abcdef0123456789",
        size: 512,
        current: true,
        restorable: true,
        applied_at: "2026-07-23T12:00:00Z",
      }],
    })))))
    renderCard()
    expect(await screen.findByText("当前配置")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "恢复此配置" })).not.toBeInTheDocument()
  })

  it("shows densified load failure diagnostics", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "internal_error", message: "history unavailable" }, meta: null,
    }), { status: 500 }))))
    renderCard()
    expect(await screen.findByText("无法加载配置应用记录")).toBeInTheDocument()
    const alert = await screen.findByTestId("card-query-error")
    expect(alert).toHaveAttribute("data-error-code", "internal")
    expect(screen.getByText("history unavailable")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制加载错误" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
  })

  it("copies apply error details", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      events: [
        {
          id: "1",
          source: "raw",
          status: "rolled_back",
          hash: "abcdef0123456789",
          size: 2048,
          error: "restart failed after config save",
          applied_at: "2026-07-23T12:00:00Z",
        },
      ],
    })))))
    renderCard()
    await screen.findByText("restart failed after config save")
    await userEvent.setup().click(screen.getByRole("button", { name: "复制错误: 完整配置保存" }))
    await waitFor(() => expect(spy).toHaveBeenCalled())
    expect(spy.mock.calls[0][0]).toContain("restart failed after config save")
    expect(spy.mock.calls[0][0]).toContain("source: raw")
    expect(spy.mock.calls[0][0]).toContain("code: restart_failed")
    expect(toast.success).toHaveBeenCalledWith("应用错误已复制")
  })

  it("deep-links pathful apply errors to section editors", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      events: [
        {
          id: "1",
          source: "raw",
          status: "rolled_back",
          hash: "abcdef0123456789",
          size: 2048,
          error: "inbounds[0].listen_port: invalid",
          applied_at: "2026-07-23T12:00:00Z",
        },
      ],
    })))))
    renderCard()
    expect(await screen.findByText("inbounds[0].listen_port: invalid")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "跳转到路径: inbounds[0].listen_port" })).toHaveAttribute(
      "href",
      "/advanced/raw?path=inbounds%5B0%5D.listen_port",
    )
    expect(screen.getByRole("link", { name: "打开来源: 完整配置保存" })).toHaveAttribute(
      "href",
      "/advanced/raw?path=inbounds%5B0%5D.listen_port",
    )
    expect(screen.getByRole("link", { name: "打开对应分区: inbounds[0].listen_port" })).toHaveAttribute(
      "href",
      "/proxy/inbounds",
    )
  })

  it("renders validate-only events", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      events: [
        {
          id: "v1",
          source: "validate",
          status: "validate_failed",
          hash: "deadbeefcafebabe",
          size: 128,
          error: "inbounds[0].type: required",
          error_code: "config_invalid",
          applied_at: "2026-07-24T12:00:00Z",
        },
        {
          id: "v2",
          source: "validate",
          status: "validated",
          hash: "feedfacefeedface",
          size: 256,
          applied_at: "2026-07-24T11:00:00Z",
        },
      ],
    })))))
    renderCard()
    expect((await screen.findAllByRole("link", { name: "Dry-run 校验" })).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText("校验失败")).toBeInTheDocument()
    expect(screen.getByText("校验通过")).toBeInTheDocument()
    expect(screen.getByText("inbounds[0].type: required")).toBeInTheDocument()
  })


})
