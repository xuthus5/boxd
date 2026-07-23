import type { ReactElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { RuntimeGroupCard, RuntimeGroupsCard } from "@/features/nodes/runtime-groups-card"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function wrap(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}><MemoryRouter>{ui}</MemoryRouter></QueryClientProvider></I18nextProvider>)
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("RuntimeGroupCard", () => {
  it("selects a selector member and refreshes groups", async () => {
    const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = typeof input === "string" ? input : input.toString()
      if (init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify({ selected: "b" })))
      }
      if (path.includes("/api/nodes/groups")) {
        return Promise.resolve(new Response(JSON.stringify({ groups: [{ type: "selector", tag: "proxy", now: "a", all: ["a", "b"] }] })))
      }
      return Promise.resolve(new Response("{}"))
    })
    vi.stubGlobal("fetch", fetchMock)
    wrap(<RuntimeGroupCard group={{ type: "selector", tag: "proxy", now: "a", all: ["a", "b"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("combobox", { name: "proxy" }))
    await user.click(await screen.findByRole("option", { name: "b" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("/api/nodes/selectors/proxy/select"), expect.objectContaining({ method: "POST" })))
  })

  it("deep-links current member to connections and logs", () => {
    wrap(<RuntimeGroupCard group={{ type: "selector", tag: "proxy", now: "a", all: ["a", "b"] }} />)
    expect(screen.getByRole("link", { name: "查看连接: a" })).toHaveAttribute(
      "href",
      "/observability/connections?outbound=a",
    )
    expect(screen.getByRole("link", { name: "查看日志: a" })).toHaveAttribute(
      "href",
      "/observability/logs?q=a",
    )
  })

  it("reports selector selection errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "internal_error", message: "select failed" }), { status: 500 })))
    wrap(<RuntimeGroupCard group={{ type: "selector", tag: "proxy", now: "a", all: ["a", "b"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("combobox", { name: "proxy" }))
    await user.click(await screen.findByRole("option", { name: "b" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("select failed"))
  })

  it("runs urltest and renders delays", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ a: 12, b: 34 }))))
    wrap(<RuntimeGroupCard group={{ type: "urltest", tag: "auto", now: "a", all: ["a", "b"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "运行 auto URLTest" }))
    expect(await screen.findByText("a: 12 ms")).toBeInTheDocument()
    expect(screen.getByText("b: 34 ms")).toBeInTheDocument()
  })

  it("reports urltest failures", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "internal_error", message: "urltest failed" }), { status: 500 })))
    wrap(<RuntimeGroupCard group={{ type: "urltest", tag: "auto", now: "a", all: ["a"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "运行 auto URLTest" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("urltest failed"))
  })
})

describe("RuntimeGroupsCard", () => {
  it("renders nothing when no runtime groups exist", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ groups: [] }))))
    wrap(<RuntimeGroupsCard />)
    await waitFor(() => expect(screen.queryByText("运行时分组")).not.toBeInTheDocument())
  })

  it("lists runtime groups when present", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      groups: [{ type: "selector", tag: "proxy", now: "a", all: ["a"] }],
    }))))
    wrap(<RuntimeGroupsCard />)
    expect(await screen.findByText("运行时分组")).toBeInTheDocument()
    expect(screen.getByText("proxy")).toBeInTheDocument()
  })
})

  it("shows config vs runtime type mismatch", () => {
    wrap(<RuntimeGroupCard group={{ type: "selector", tag: "xuthus5", now: "a", all: ["a", "b"] }} configType="urltest" />)
    expect(screen.getByText("配置与运行时不一致")).toBeInTheDocument()
    expect(screen.getByText("配置与运行时类型不一致")).toBeInTheDocument()
    expect(screen.getByText(/配置为 urltest/)).toBeInTheDocument()
  })

