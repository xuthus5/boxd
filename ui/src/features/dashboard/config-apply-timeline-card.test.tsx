import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ConfigApplyTimelineCard } from "@/features/dashboard/config-apply-timeline-card"
import { AuthProvider } from "@/features/auth/auth-context"
import { PreferencesProvider } from "@/features/preferences/preferences-provider"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

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
    expect(await screen.findByText("完整配置保存")).toBeInTheDocument()
    expect(screen.getByText("结构化保存")).toBeInTheDocument()
    expect(screen.getByText("已回滚")).toBeInTheDocument()
    expect(screen.getByText("已应用")).toBeInTheDocument()
    expect(screen.getByText("restart failed after config save")).toBeInTheDocument()
    expect(screen.getByText(/abcdef01/)).toBeInTheDocument()
  })

  it("shows load failure description", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response("nope", { status: 500 }))))
    renderCard()
    expect(await screen.findByText("无法加载配置应用记录")).toBeInTheDocument()
  })
})
