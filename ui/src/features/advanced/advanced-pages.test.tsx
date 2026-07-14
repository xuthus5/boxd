import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("advanced configuration", () => {
  it("opens the full configuration editor", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ log: { level: "info" } }))))
    renderApp(<App />, "/advanced/raw")
    expect(await screen.findByRole("heading", { name: "完整配置" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存完整配置" })).toBeInTheDocument()
  })
})
