import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import type { SingBoxConfig } from "@/lib/api/types"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

const okEnvelope = { status: "ok", data: null, error: null, meta: {} }

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function setup(config: SingBoxConfig = {
  log: { level: "info" },
  experimental: {
    cache_file: { enabled: true, path: "/var/lib/boxd/cache.db" },
  },
  outbounds: [{ type: "direct", tag: "direct" }, { type: "selector", tag: "proxy" }],
  inbounds: [{ type: "mixed", tag: "mixed-in" }],
}) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
    if (path === "/api/config/" && init?.method === "PUT") {
      return Promise.resolve(new Response(JSON.stringify(okEnvelope)))
    }
    if (path === "/api/config/" || path === "/api/config/raw") {
      return Promise.resolve(new Response(JSON.stringify(config)))
    }
    if (path === "/api/network/interfaces") {
      return Promise.resolve(new Response(JSON.stringify({ interfaces: [] })))
    }
    return Promise.resolve(new Response(JSON.stringify({})))
  })
  vi.stubGlobal("fetch", fetchMock)
  return { user: userEvent.setup(), fetchMock, view: renderApp(<App />, "/advanced/experimental") }
}

describe("experimental page", () => {
  it("renders visual and advanced tabs", async () => {
    setup()
    expect(await screen.findByRole("heading", { name: "实验特性" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "可视化配置" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toBeInTheDocument()
    expect(screen.getByText("缓存文件")).toBeInTheDocument()
    expect(screen.getByText("Clash API")).toBeInTheDocument()
    expect(screen.getByText("V2Ray API")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存配置" })).toHaveClass("h-8")
    expect(screen.getByRole("button", { name: "启用 Clash API" })).toHaveClass("h-8")
  })

  it("saves experimental changes from the visual form", async () => {
    const { user, fetchMock } = setup()
    await screen.findByRole("heading", { name: "实验特性" })
    const path = await screen.findByLabelText("缓存路径")
    await user.clear(path)
    await user.type(path, "/tmp/experimental-cache.db")
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" }))
    })
    const putCall = fetchMock.mock.calls.find(([url, init]) => String(url) === "/api/config/" && (init as RequestInit | undefined)?.method === "PUT")
    const body = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as {
      experimental?: { cache_file?: { path?: string; enabled?: boolean } }
    }
    expect(body.experimental?.cache_file?.enabled).toBe(true)
    expect(body.experimental?.cache_file?.path).toBe("/tmp/experimental-cache.db")
  })

  it("switches to advanced JSON and keeps save enabled for objects", async () => {
    const { user } = setup()
    await screen.findByRole("heading", { name: "实验特性" })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expect(await screen.findByLabelText("实验特性配置 JSON")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存配置" })).toBeEnabled()
  })

  it("disables save when advanced JSON is not an object", async () => {
    const { user } = setup()
    await screen.findByRole("heading", { name: "实验特性" })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = await screen.findByLabelText("实验特性配置 JSON")
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}[BracketLeft][BracketRight]")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })

  it("surfaces densified save errors with code and clipboard actions", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
      if (path === "/api/config/" && init?.method === "PUT") {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error",
          data: null,
          error: { code: "config_invalid_runtime", message: "experimental.clash_api.secret: required" },
          meta: {},
        }), { status: 400 }))
      }
      if (path === "/api/config/" || path === "/api/config/raw") {
        return Promise.resolve(new Response(JSON.stringify({
          log: { level: "info" },
          experimental: { cache_file: { enabled: true, path: "/tmp/x.db" } },
        })))
      }
      if (path === "/api/network/interfaces") {
        return Promise.resolve(new Response(JSON.stringify({ interfaces: [] })))
      }
      if (path === "/api/settings/password") {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/advanced/experimental")
    await screen.findByRole("heading", { name: "实验特性" })
    const save = screen.getByRole("button", { name: "保存配置" })
    expect(save).toBeEnabled()
    await user.click(save)
    expect(await screen.findByTestId("config-save-error")).toBeInTheDocument()
    expect(screen.getByText("config_invalid")).toBeInTheDocument()
    expect(screen.getAllByText(/experimental\.clash_api\.secret/).length).toBeGreaterThan(0)
    await user.click(screen.getByRole("button", { name: "跳转到路径" }))
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toHaveAttribute("aria-selected", "true")
  })

})
