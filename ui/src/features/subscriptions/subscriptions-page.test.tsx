import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("SubscriptionsPage", () => {
  it("lists subscriptions and exposes refresh actions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/urltest-defaults")
        ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
        : path.endsWith("/nodes/") ? [] : [{
          id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
          last_updated: "2026-01-01T00:00:00Z", outbounds: [],
        }]
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    const { container } = renderApp(<App />, "/subscriptions")

    expect(await screen.findByText("主订阅")).toBeInTheDocument()
    expect(container.querySelector("[data-slot=card] [data-slot=card]")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "新增订阅" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "刷新全部" })).toBeInTheDocument()
  })

  it("treats an empty URLTest override object as global inheritance", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      const data = path.endsWith("/urltest-defaults")
        ? { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
        : path.endsWith("/nodes/") ? [] : [{
          id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
          last_updated: "2026-01-01T00:00:00Z", outbounds: [], urltest: {},
        }]
      return Promise.resolve(new Response(JSON.stringify(data)))
    }))
    renderApp(<App />, "/subscriptions")

    expect(await screen.findByText("URLTest：继承全局")).toBeInTheDocument()
    expect(screen.queryByText("URLTest：自定义")).not.toBeInTheDocument()
  })
})
