import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), message: vi.fn() }, Toaster: () => null }))

afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks(); sessionStore.clear() })

describe("default proxy installation", () => {
  it.each([
    ["inbounds", "/proxy/inbounds", "安装默认入站"],
    ["outbounds", "/proxy/outbounds", "安装默认出站"],
    ["dns", "/policy/dns", "安装默认 DNS"],
    ["rule-sets", "/policy/route", "安装默认路由"],
    ["route", "/policy/route", "安装默认路由"],
    ["experimental", "/advanced/experimental", "启用 Clash API"],
  ])("shows rollback instead of success for %s", async (section, route, label) => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(input)
      if (init?.method === "POST" && path === `/api/config/${section}/defaults`) return Promise.resolve(new Response(JSON.stringify({
        status: "rolled_back", data: [], error: { code: "config_restart_failed", message: "TUN unavailable; previous config restored" }, meta: { rolled_back: true },
      })))
      if (init?.method === "POST") return Promise.resolve(new Response(JSON.stringify({ status: "ok", data: [], error: null, meta: null })))
      const body = path.includes("preferences") ? { theme: "system", language: "zh", minimumLogLevel: "all" }
        : path.includes("password") ? { defaultPassword: false } : { inbounds: [], outbounds: [], route: {}, dns: {}, experimental: {} }
      return Promise.resolve(new Response(JSON.stringify(body)))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, route)
    await user.click(await screen.findByRole("button", { name: label }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("配置已回滚")
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    expect(toast.success).not.toHaveBeenCalled()
    if (section === "rule-sets") expect(fetchMock.mock.calls.some(([url]) => String(url) === "/api/config/route/defaults")).toBe(false)
  })
})
