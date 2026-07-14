import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function authenticate() {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
}

describe("proxy alternate states", () => {
  it("shows an empty inbound state and closes the add dialog", async () => {
    authenticate()
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ inbounds: [] })))))
    const user = userEvent.setup()
    renderApp(<App />, "/proxy/inbounds")
    expect(await screen.findByText("暂无配置")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "新增入站" })[0])
    await user.click(screen.getByRole("button", { name: "取消" }))
  })

  it("shows a configuration query error", async () => {
    authenticate()
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "internal_error", message: "failed" }, meta: null,
    }), { status: 500 }))))
    renderApp(<App />, "/proxy/inbounds")
    expect(await screen.findByText("加载失败", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("reports a rolled back save", async () => {
    authenticate()
    vi.stubGlobal("fetch", vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ status: "rolled_back", data: null, error: null, meta: {} })))
      return Promise.resolve(new Response(JSON.stringify({ inbounds: [{ tag: "mixed", type: "mixed" }] })))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/proxy/inbounds")
    await user.click(await screen.findByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(await screen.findByText("配置已回滚")).toBeInTheDocument()
  })
})
