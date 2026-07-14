import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("policy alternate states", () => {
  it("uses an empty object when the policy section is absent", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response("{}"))))
    renderApp(<App />, "/policy/dns")
    expect(await screen.findByRole("button", { name: "保存配置" })).toBeEnabled()
  })

  it("shows a policy query error", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({
      code: "internal_error", message: "config failed",
    }), { status: 500 }))))
    renderApp(<App />, "/policy/route")
    expect(await screen.findByText("config failed", {}, { timeout: 3000 })).toBeInTheDocument()
  })

  it("reports default installer failures", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "POST") return Promise.resolve(new Response(JSON.stringify({ code: "bad_gateway", message: "download failed" }), { status: 502 }))
      return Promise.resolve(new Response(JSON.stringify({ dns: {} })))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/policy/dns")
    await user.click(await screen.findByRole("button", { name: "安装默认 DNS" }))
    expect(await screen.findByText("download failed")).toBeInTheDocument()
  })
})

describe("policy save and rollback states", () => {
  it("reports policy save failures", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ code: "internal_error", message: "save route failed" }), { status: 500 }))
      return Promise.resolve(new Response(JSON.stringify({ route: {} })))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/policy/route")
    await user.click(await screen.findByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("save route failed")).toBeInTheDocument()
  })

  it("reports a rolled back policy save", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "PUT") return Promise.resolve(new Response(JSON.stringify({ status: "rolled_back", data: null, error: null, meta: {} })))
      return Promise.resolve(new Response(JSON.stringify({ route: {} })))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/policy/route")
    await user.click(await screen.findByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("配置已回滚")).toBeInTheDocument()
  })

  it("reports a rolled back default installer", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "POST") return Promise.resolve(new Response(JSON.stringify({ status: "rolled_back", data: null, error: null, meta: {} })))
      return Promise.resolve(new Response(JSON.stringify({ dns: {} })))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/policy/dns")
    await user.click(await screen.findByRole("button", { name: "安装默认 DNS" }))
    expect(await screen.findByText("配置已回滚")).toBeInTheDocument()
  })

  it("stops route installation when rule sets roll back", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "POST") return Promise.resolve(new Response(JSON.stringify({ status: "rolled_back", data: null, error: null, meta: {} })))
      return Promise.resolve(new Response(JSON.stringify({ route: {} })))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/policy/route")
    await user.click(await screen.findByRole("button", { name: "安装默认路由" }))
    await screen.findByText("配置已回滚")
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === "POST")).toHaveLength(1)
  })
})
