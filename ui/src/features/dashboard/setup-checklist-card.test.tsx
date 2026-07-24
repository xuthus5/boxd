import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { AuthProvider } from "@/features/auth/auth-context"
import { SetupChecklistCard } from "@/features/dashboard/setup-checklist-card"
import { PreferencesProvider } from "@/features/preferences/preferences-provider"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function renderCard(status = { running: true, uptime: "1m" }) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <AuthProvider>
        <PreferencesProvider>
          <SetupChecklistCard status={status} />
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  )
}

describe("SetupChecklistCard", () => {
  it("deep-links to failed subscriptions while setup is complete", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/config")) {
        return Promise.resolve(new Response(JSON.stringify({
          inbounds: [{ type: "mixed", tag: "mixed-in" }],
          outbounds: [{ type: "selector", tag: "proxy", outbounds: ["hk"] }],
          route: { rules: [{ outbound: "proxy" }] },
          experimental: { clash_api: { external_controller: "127.0.0.1:9090" } },
        })))
      }
      if (path.includes("/api/subscriptions")) {
        return Promise.resolve(new Response(JSON.stringify([
          {
            id: "ok",
            name: "正常订阅",
            url: "https://example.com/ok",
            interval_min: 60,
            last_updated: "2026-01-01T00:00:00Z",
            outbounds: [],
          },
          {
            id: "bad",
            name: "失败订阅",
            url: "https://example.com/bad",
            interval_min: 60,
            last_updated: "2026-01-01T00:00:00Z",
            outbounds: [],
            error: "timeout",
          },
        ])))
      }
      if (path.includes("/api/settings/preferences")) {
        return Promise.resolve(new Response(JSON.stringify({
          theme: "system", language: "zh", minimumLogLevel: "all",
        })))
      }
      if (path.includes("/api/settings/password")) {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))

    renderCard()
    expect(await screen.findByText("1 个订阅刷新失败")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看失败订阅" })).toHaveAttribute(
      "href",
      "/subscriptions?status=error",
    )
    expect(screen.getByText("失败订阅")).toBeInTheDocument()
    expect(screen.getAllByText("timeout").length).toBeGreaterThan(0)
    expect(screen.getByRole("link", { name: "查看: 失败订阅" })).toHaveAttribute(
      "href",
      "/subscriptions?q=%E5%A4%B1%E8%B4%A5%E8%AE%A2%E9%98%85&status=error",
    )
    // setup complete: incomplete steps list is hidden
    expect(screen.queryByText("去配置")).not.toBeInTheDocument()
  })

  it("hides the card when setup is complete and no subscription failures", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/config")) {
        return Promise.resolve(new Response(JSON.stringify({
          inbounds: [{ type: "mixed", tag: "mixed-in" }],
          outbounds: [{ type: "selector", tag: "proxy", outbounds: ["hk"] }],
          route: { rules: [{ outbound: "proxy" }] },
          experimental: { clash_api: { external_controller: "127.0.0.1:9090" } },
        })))
      }
      if (path.includes("/api/subscriptions")) {
        return Promise.resolve(new Response(JSON.stringify([
          {
            id: "ok",
            name: "正常订阅",
            url: "https://example.com/ok",
            interval_min: 60,
            last_updated: "2026-01-01T00:00:00Z",
            outbounds: [],
          },
        ])))
      }
      if (path.includes("/api/settings/preferences")) {
        return Promise.resolve(new Response(JSON.stringify({
          theme: "system", language: "zh", minimumLogLevel: "all",
        })))
      }
      if (path.includes("/api/settings/password")) {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))

    const view = renderCard()
    // wait a tick for queries
    await screen.findByText("快速上手").catch(() => null)
    // card should unmount content / return null - title may not exist
    expect(view.container.querySelector("[data-slot=card]")).toBeNull()
  })

  it("densifies subscription query failure with retry", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path.includes("/api/config")) {
        return Promise.resolve(new Response(JSON.stringify({
          inbounds: [{ type: "mixed", tag: "mixed-in" }],
          outbounds: [{ type: "selector", tag: "proxy", outbounds: ["hk"] }],
          route: { rules: [{ outbound: "proxy" }] },
          experimental: { clash_api: { external_controller: "127.0.0.1:9090" } },
        })))
      }
      if (path.includes("/api/subscriptions")) {
        return Promise.resolve(new Response(JSON.stringify({ error: "boom" }), { status: 500 }))
      }
      if (path.includes("/api/settings/preferences")) {
        return Promise.resolve(new Response(JSON.stringify({
          theme: "system", language: "zh", minimumLogLevel: "all",
        })))
      }
      if (path.includes("/api/settings/password")) {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    renderCard()
    expect(await screen.findByTestId("card-query-error")).toBeInTheDocument()
    expect(document.querySelector('[data-error-code]')).not.toBeNull()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
  })
})
