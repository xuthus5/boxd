import type { ReactElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { RuntimeGroupCard, RuntimeGroupsCard } from "@/features/nodes/runtime-groups-card"
import { copyText } from "@/features/proxy/copy-tag-button"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }))
vi.mock("@/features/proxy/copy-tag-button", () => ({ copyText: vi.fn() }))

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
    vi.mocked(copyText).mockResolvedValue(undefined)
    wrap(<RuntimeGroupCard group={{ type: "selector", tag: "proxy", now: "a", all: ["a", "b"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("combobox", { name: "proxy" }))
    await user.click(await screen.findByRole("option", { name: "b" }))
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled()
      const [message, options] = vi.mocked(toast.error).mock.calls[0]
      expect(String(message)).toContain("select failed")
      expect(options).toEqual(expect.objectContaining({
        description: expect.any(String),
        action: expect.objectContaining({ label: expect.any(String) }),
      }))
    })
    const [, options] = vi.mocked(toast.error).mock.calls[0] ?? []
    options?.action?.onClick?.()
    await waitFor(() => expect(toast.success).toHaveBeenCalled())
  })

  it("runs urltest and renders delays", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ a: 12, b: 34 }))))
    wrap(<RuntimeGroupCard group={{ type: "urltest", tag: "auto", now: "a", all: ["a", "b"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "运行 auto URLTest" }))
    expect(await screen.findByText("a")).toBeInTheDocument()
    expect(screen.getByText("12 ms")).toBeInTheDocument()
    expect(screen.getByText("b")).toBeInTheDocument()
    expect(screen.getByText("34 ms")).toBeInTheDocument()
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith(expect.stringContaining("URLTest：2/2 成功，0 失败")))
  })

  it("renders zero, invalid, and missing urltest delays with a warning", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ a: 0, b: -1 }))))
    wrap(<RuntimeGroupCard group={{ type: "urltest", tag: "auto", now: "", all: ["a", "b", "c"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "运行 auto URLTest" }))
    expect(await screen.findByText("0 ms")).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
    await waitFor(() => {
      expect(toast.warning).toHaveBeenCalledWith(expect.stringContaining("URLTest：2/3 成功，1 失败"), expect.anything())
    })
    expect(screen.queryByRole("link", { name: /查看连接/ })).not.toBeInTheDocument()
  })

  it("treats a null urltest payload as missing members", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("null")))
    wrap(<RuntimeGroupCard group={{ type: "urltest", tag: "auto", now: "a", all: ["a"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "运行 auto URLTest" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("URLTest：0/1 成功，1 失败"), expect.anything()))
  })

  it("reports urltest failures", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "internal_error", message: "urltest failed" }), { status: 500 })))
    vi.mocked(copyText).mockResolvedValue(undefined)
    wrap(<RuntimeGroupCard group={{ type: "urltest", tag: "auto", now: "a", all: ["a"] }} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "运行 auto URLTest" }))
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled()
      const [message, options] = vi.mocked(toast.error).mock.calls[0]
      expect(String(message)).toContain("urltest failed")
      expect(options).toEqual(expect.objectContaining({
        description: expect.any(String),
        action: expect.objectContaining({ label: expect.any(String) }),
      }))
    })
    const [, options] = vi.mocked(toast.error).mock.calls[0] ?? []
    options?.action?.onClick?.()
    await waitFor(() => expect(toast.success).toHaveBeenCalled())
  })

  it("shows config vs runtime type mismatch", () => {
    wrap(<RuntimeGroupCard group={{ type: "selector", tag: "xuthus5", now: "a", all: ["a", "b"] }} configType="urltest" />)
    expect(screen.getByText("配置与运行时不一致")).toBeInTheDocument()
    expect(screen.getByText("配置与运行时类型不一致")).toBeInTheDocument()
    expect(screen.getByText(/配置为 urltest/)).toBeInTheDocument()
  })

  it("omits current-member links when the group has no current value", () => {
    wrap(<RuntimeGroupCard group={{ type: "selector", tag: "empty", now: "", all: ["a"] }} />)
    expect(screen.queryByRole("link", { name: /查看连接/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: /查看日志/ })).not.toBeInTheDocument()
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

  it("densifies groups query failure with retry", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "unavailable", message: "groups unavailable" }, meta: null,
    }), { status: 503 }))
    vi.stubGlobal("fetch", fetchMock)
    wrap(<RuntimeGroupsCard />)
    const alert = await screen.findByTestId("card-query-error")
    expect(alert).toHaveAttribute("data-error-code", "unavailable")
    expect(screen.getByText("groups unavailable")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()
    expect(screen.getByText("运行时分组")).toBeInTheDocument()
    await userEvent.setup().click(screen.getByRole("button", { name: "重试" }))
    await waitFor(() => expect(fetchMock.mock.calls.length).toBeGreaterThan(1))
  })

  it("shows a skeleton while runtime groups are loading", async () => {
    let resolve: (response: Response) => void = () => undefined
    const pending = new Promise<Response>((next) => { resolve = next })
    vi.stubGlobal("fetch", vi.fn(() => pending))
    wrap(<RuntimeGroupsCard />)
    expect(document.querySelector('[data-slot="skeleton"]')).not.toBeNull()
    resolve(new Response(JSON.stringify({ groups: [] })))
    await waitFor(() => expect(screen.queryByText("运行时分组")).not.toBeInTheDocument())
  })
})
