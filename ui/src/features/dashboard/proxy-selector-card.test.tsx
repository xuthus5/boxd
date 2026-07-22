import type { ReactElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { ProxySelectorCard } from "@/features/dashboard/proxy-selector-card"
import { i18n } from "@/i18n"

function wrap(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}>{ui}</QueryClientProvider></I18nextProvider>)
}

afterEach(() => vi.unstubAllGlobals())

describe("ProxySelectorCard", () => {
  it("selects a preferred proxy group member", async () => {
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname)
      if (path.includes("/api/nodes/groups")) {
        return Promise.resolve(new Response(JSON.stringify({
          groups: [{ tag: "proxy", type: "selector", now: "a", all: ["a", "b"] }],
        })))
      }
      if (init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify({ selected: "b" })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    })
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    wrap(<ProxySelectorCard />)
    expect(await screen.findByText("当前出口")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "当前出口" }))
    await user.click(await screen.findByRole("option", { name: "b" }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("/api/nodes/selectors/proxy/select"),
        expect.objectContaining({ method: "POST" }),
      )
    })
  })

  it("shows empty state when no selector exists", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ groups: [] }))))
    wrap(<ProxySelectorCard />)
    expect(await screen.findByText("内核未运行或没有可用的 selector 分组。")).toBeInTheDocument()
  })
})
