import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("SubscriptionsPage", () => {
  it("lists subscriptions and exposes refresh actions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify([{
      id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z", outbounds: [],
    }]))))
    renderApp(<App />, "/subscriptions")

    expect(await screen.findByText("主订阅")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "新增订阅" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "刷新全部" })).toBeInTheDocument()
  })
})
