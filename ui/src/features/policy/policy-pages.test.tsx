import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("policy pages", () => {
  it("shows route configuration and the default installer", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      route: { final: "proxy", rules: [] },
    }))))

    renderApp(<App />, "/policy/route")

    expect(await screen.findByRole("heading", { name: "路由" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "安装默认路由" })).toBeInTheDocument()
  })
})
