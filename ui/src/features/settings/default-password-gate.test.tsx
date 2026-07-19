import { screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { i18n } from "@/i18n"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

const timeout = 5_000

beforeEach(async () => {
  sessionStorage.clear()
  localStorage.clear()
  sessionStore.clear()
  await i18n.changeLanguage("zh")
})

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
  sessionStorage.clear()
})

function requestPath(input: string | URL | Request) {
  const raw = typeof input === "string"
    ? input
    : input instanceof URL
      ? input.href
      : input.url
  try {
    return new URL(raw, "http://localhost").pathname
  } catch {
    return String(raw).split("?")[0] ?? ""
  }
}

function json(data: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  }))
}

function stub(defaultPassword: boolean) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = vi.fn(async (input: string | URL | Request) => {
    const path = requestPath(input)
    if (path === "/api/settings/password") {
      return json({ defaultPassword })
    }
    if (path === "/api/settings/jwt-secret") {
      return json({ masked: "ab**cd", present: true, length: 32 })
    }
    if (path === "/api/settings/url-test") {
      return json({ url: "https://cp.cloudflare.com/" })
    }
    if (path === "/api/settings/urltest-defaults") {
      return json({ enabled: true, url: "https://cp.cloudflare.com/", interval: "3m", tolerance: 50 })
    }
    if (path === "/api/settings/kernel-autostart") {
      return json({ enabled: true })
    }
    if (path === "/api/config/rule-sets/auto-update") {
      return json({ enabled: false, interval: "24h" })
    }
    if (path === "/api/service/status") {
      return json({ running: true, uptime: "1m" })
    }
    if (path === "/api/runtime/memory") {
      return json({ alloc: 1, total: 1, sys: 1, num_gc: 0, heap_inuse: 1, stack_inuse: 1, num_goroutine: 1 })
    }
    if (path === "/api/runtime/version") {
      return json({ version: "dev", kernel_version: "1.13.14" })
    }
    if (path === "/api/stats/traffic/history") {
      return json({ points: [] })
    }
    if (path.startsWith("/api/stats/")) {
      return new Response(new ReadableStream({
        start(controller) {
          controller.enqueue(new TextEncoder().encode("data: {}\n\n"))
          controller.close()
        },
      }), { headers: { "Content-Type": "text/event-stream" } })
    }
    return json({})
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

describe("default password gate", () => {
  it("redirects non-settings pages to settings when default password is active", async () => {
    const fetchMock = stub(true)
    renderApp(<App />, "/dashboard")

    await waitFor(() => {
      expect(fetchMock.mock.calls.some((call) => requestPath(call[0] as string | URL | Request) === "/api/settings/password")).toBe(true)
    }, { timeout })

    expect(await screen.findByText("请先轮换默认管理员密码后，才能使用其他面板功能。", {}, { timeout })).toBeInTheDocument()
    expect(await screen.findByRole("heading", { name: "应用设置" }, { timeout })).toBeInTheDocument()
    // dashboard should no longer be the active page heading once redirected
    expect(screen.queryByRole("heading", { name: "仪表盘" })).not.toBeInTheDocument()
  })

  it("allows dashboard when default password is not active", async () => {
    stub(false)
    renderApp(<App />, "/dashboard")
    expect(await screen.findByRole("heading", { name: "仪表盘" }, { timeout })).toBeInTheDocument()
    expect(screen.queryByText("请先轮换默认管理员密码后，才能使用其他面板功能。")).not.toBeInTheDocument()
  })
})
