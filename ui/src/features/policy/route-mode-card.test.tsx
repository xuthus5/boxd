import type { ReactElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { I18nextProvider } from "react-i18next"
import { toast } from "sonner"
import { afterEach, describe, expect, it, vi } from "vitest"

import { RouteModeCard } from "@/features/policy/route-mode-card"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function wrap(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}>{ui}</QueryClientProvider></I18nextProvider>)
}

function clashModeFetch(modes: string[], mode = "Rule") {
  return vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname)
    if (path.includes("/api/runtime/clash-mode") && init?.method === "PUT") {
      const body = JSON.parse(String(init.body)) as { mode: string }
      return Promise.resolve(new Response(JSON.stringify({ mode: body.mode, mode_list: modes })))
    }
    return Promise.resolve(new Response(JSON.stringify({ mode, mode_list: modes })))
  })
}

afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks() })

describe("RouteModeCard", () => {
  it("switches the route mode through the runtime api", async () => {
    const fetchMock = clashModeFetch(["Rule", "Global", "Direct"])
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    wrap(<RouteModeCard enabled />)

    expect(await screen.findByRole("button", { name: "规则模式", pressed: true })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "全局直连" }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/runtime/clash-mode",
        expect.objectContaining({ method: "PUT", body: JSON.stringify({ mode: "Direct" }) }),
      )
    })
    expect(await screen.findByRole("button", { name: "全局直连", pressed: true })).toBeInTheDocument()
    expect(toast.success).toHaveBeenCalledWith("路由模式已切换为全局直连")
  })

  it("hints to install the default route when clash_mode rules are missing", async () => {
    vi.stubGlobal("fetch", clashModeFetch(["Rule"]))
    wrap(<RouteModeCard enabled />)

    expect(await screen.findByText(/缺少 clash_mode 规则/)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "全局直连" })).not.toBeInTheDocument()
  })

  it("asks to start the kernel when it is not running", () => {
    wrap(<RouteModeCard enabled={false} />)

    expect(screen.getByText("启动内核后可切换路由模式。")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "全局代理" })).not.toBeInTheDocument()
  })

  it("reports clash_api as unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "invalid_request", message: "feature not enabled" }, meta: null,
    }), { status: 400 }))))
    wrap(<RouteModeCard enabled />)

    expect(await screen.findByText(/未启用 clash_api/)).toBeInTheDocument()
  })
})
