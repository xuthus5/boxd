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
    expect(await screen.findAllByText("download failed")).not.toHaveLength(0)
  })

  it("installs route rule sets before route defaults and refetches", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const requests: string[] = []
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === "POST") {
        requests.push(url)
        return Promise.resolve(new Response(JSON.stringify({ status: "ok", data: null, error: null, meta: {} })))
      }
      return Promise.resolve(new Response(JSON.stringify({ route: {} })))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/policy/route")
    await user.click(await screen.findByRole("button", { name: "安装默认路由" }))

    expect(await screen.findByText("默认配置已安装")).toBeInTheDocument()
    expect(requests).toEqual([
      "/api/config/rule-sets/defaults",
      "/api/config/route/defaults",
    ])
    const reads = fetchMock.mock.calls.filter(([, init]) => init?.method === undefined).map(([input]) => String(input))
    expect(reads.filter((url) => url === "/api/config/")).toHaveLength(2)
    expect(reads.filter((url) => url === "/api/config/route/rule-metadata")).toHaveLength(2)
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
    expect(await screen.findAllByText("save route failed")).not.toHaveLength(0)
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
    expect(await screen.findAllByText("配置已回滚")).not.toHaveLength(0)
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
    expect(await screen.findAllByText("配置已回滚")).not.toHaveLength(0)
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
    await screen.findAllByText("配置已回滚")
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === "POST")).toHaveLength(1)
  })
})
