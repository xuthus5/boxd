import type { ReactElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { ClashModeCard } from "@/features/dashboard/clash-mode-card"
import { i18n } from "@/i18n"

function wrap(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}>{ui}</QueryClientProvider></I18nextProvider>)
}

afterEach(() => { vi.unstubAllGlobals() })

describe("ClashModeCard", () => {
  it("switches clash mode", async () => {
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname)
      if (path.includes("/api/runtime/clash-mode") && init?.method === "PUT") {
        return Promise.resolve(new Response(JSON.stringify({ mode: "Global", mode_list: ["Rule", "Global", "Direct"] })))
      }
      return Promise.resolve(new Response(JSON.stringify({ mode: "Rule", mode_list: ["Rule", "Global", "Direct"] })))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    wrap(<ClashModeCard enabled />)
    expect(await screen.findByText("当前")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Global" }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/runtime/clash-mode",
        expect.objectContaining({ method: "PUT" }),
      )
    })
    expect(await screen.findByRole("button", { name: "Global", pressed: true })).toBeInTheDocument()
  })

  it("shows disabled hint when clash_api missing", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "invalid_request", message: "feature not enabled" }, meta: null,
    }), { status: 400 }))))
    wrap(<ClashModeCard enabled />)
    expect(await screen.findByText(/clash_api/)).toBeInTheDocument()
  })

  it("shows densified query failure diagnostics", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "internal_error", message: "clash mode boom" }, meta: null,
    }), { status: 500 }))))
    wrap(<ClashModeCard enabled />)
    const alert = await screen.findByTestId("card-query-error")
    expect(alert).toHaveAttribute("data-error-code", "internal")
    expect(screen.getByText("clash mode boom")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制加载错误" })).toBeInTheDocument()
  })


})
