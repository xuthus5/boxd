import { screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { sessionStore.clear(); vi.unstubAllGlobals() })

describe("application routing", () => {
  it("redirects unauthenticated users to login", () => {
    renderApp(<App />, "/dashboard")
    expect(screen.getByRole("heading", { name: "boxd" })).toBeInTheDocument()
  })

  it("renders the dashboard and approved navigation for a valid session", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    renderApp(<App />, "/dashboard")

    expect(await screen.findByRole("heading", { name: "仪表盘" })).toBeInTheDocument()
    expect(screen.getByText("sing-box control plane")).toBeInTheDocument()
    expect(within(screen.getByRole("banner")).getByText("仪表盘")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "日志" })).toBeInTheDocument()
    expect(screen.queryByRole("link", { name: "内核日志" })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: "应用日志" })).not.toBeInTheDocument()
  })

  it("logs out from the sidebar", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installMockAPI()
    const user = userEvent.setup()
    renderApp(<App />, "/dashboard")
    await screen.findByText("运行中")
    await user.click(screen.getByRole("button", { name: "退出登录" }))
    expect(await screen.findByRole("heading", { name: "boxd" })).toBeInTheDocument()
  })

  it("handles logout request failures after clearing the local session", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = installMockAPI()
    fetchMock.mockImplementation((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/auth/logout") return Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "logout failed" }), { status: 500 }))
      return Promise.resolve(new Response("{}"))
    })
    const user = userEvent.setup()
    renderApp(<App />, "/advanced/endpoints")
    await screen.findByRole("heading", { name: "端点" })
    await user.click(screen.getByRole("button", { name: "退出登录" }))
    expect(await screen.findByRole("heading", { name: "boxd" })).toBeInTheDocument()
    expect(await screen.findByText("logout failed")).toBeInTheDocument()
  })
})
