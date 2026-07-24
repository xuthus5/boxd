import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { RawConfigPage } from "@/features/advanced/raw-config-page"
import { i18n } from "@/i18n"

function renderPage() {
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>
        <MemoryRouter initialEntries={["/advanced/raw"]}>
          <RawConfigPage />
        </MemoryRouter>
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

describe("RawConfigPage diff", () => {
  it("renders a no-change panel for the loaded config", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "ok",
      data: { log: { level: "info" } },
      error: null,
      meta: null,
    }))))
    renderPage()
    await waitFor(() => expect(screen.getByTestId("config-diff-panel")).toHaveTextContent("配置无变化"))
    expect(screen.getByLabelText("完整配置 JSON")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "校验配置" })).toBeInTheDocument()
    vi.unstubAllGlobals()
  })

  it("shows path-level diff details after edits", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "ok",
      data: { log: { level: "info" } },
      error: null,
      meta: null,
    }))))
    renderPage()
    const editor = await screen.findByLabelText("完整配置 JSON")
    await userEvent.click(editor)
    await userEvent.keyboard("{Control>}a{/Control}")
    await userEvent.paste(JSON.stringify({ log: { level: "debug" }, dns: { final: "local" } }, null, 2))
    await waitFor(() => {
      expect(screen.getByTestId("config-diff-panel")).toHaveTextContent("log.level")
      expect(screen.getByTestId("config-diff-panel")).toHaveTextContent("dns")
    })
    vi.unstubAllGlobals()
  })

  it("validates configuration without writing when validate succeeds", async () => {
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
      if (path === "/api/config/validate" && init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify({
          status: "ok",
          data: { valid: true },
          error: null,
          meta: { validated: true, applied: false },
        })))
      }
      return Promise.resolve(new Response(JSON.stringify({
        status: "ok",
        data: { log: { level: "info" } },
        error: null,
        meta: null,
      })))
    })
    vi.stubGlobal("fetch", fetchMock)
    renderPage()
    await screen.findByLabelText("完整配置 JSON")
    await userEvent.click(screen.getByRole("button", { name: "校验配置" }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/config/validate?source=validate_raw",
        expect.objectContaining({ method: "POST" }),
      )
    })
    expect(fetchMock.mock.calls.some((call) => String(call[0]).includes("/api/config/raw") && call[1]?.method === "PUT")).toBe(false)
    vi.unstubAllGlobals()
  })

  it("surfaces densified validation errors with path jump", async () => {
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
      if (path === "/api/config/validate" && init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify({
          status: "error",
          data: null,
          error: { code: "config_invalid_runtime", message: "inbounds[0].listen_port: invalid" },
          meta: null,
        }), { status: 400 }))
      }
      return Promise.resolve(new Response(JSON.stringify({
        status: "ok",
        data: { log: { level: "info" }, inbounds: [{ tag: "mixed-in", type: "mixed", listen_port: 1080 }] },
        error: null,
        meta: null,
      })))
    })
    vi.stubGlobal("fetch", fetchMock)
    renderPage()
    await screen.findByLabelText("完整配置 JSON")
    await userEvent.click(screen.getByRole("button", { name: "校验配置" }))
    expect(await screen.findByTestId("config-save-error")).toBeInTheDocument()
    expect(screen.getAllByText(/inbounds\[0\]\.listen_port/).length).toBeGreaterThan(0)
    await userEvent.click(screen.getByRole("button", { name: "跳转到路径" }))
    vi.unstubAllGlobals()
  })
})
