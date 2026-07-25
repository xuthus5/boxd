import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("dashboard alternate states", () => {
  it("shows query failures", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "internal_error", message: "unavailable" }, meta: null,
    }), { status: 500 }))))
    renderApp(<App />, "/dashboard")
    const alert = await screen.findByTestId("page-load-error")
    expect(alert).toHaveAttribute("data-error-code", "internal")
    expect(screen.getByText("unavailable")).toBeInTheDocument()
    expect(screen.getByText(/仪表盘加载失败/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
  })
})
