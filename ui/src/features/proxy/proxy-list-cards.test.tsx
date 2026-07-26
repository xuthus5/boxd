import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { InboundCards, OutboundCards } from "@/features/proxy/proxy-list-cards"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function renderOutboundCards() {
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <OutboundCards items={[]} onEdit={vi.fn()} onDelete={vi.fn()} />
    </QueryClientProvider>,
  )
}

describe("proxy list cards", () => {
  it("shows a loading skeleton while outbound dependencies load", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)))
    const view = renderOutboundCards()
    expect(document.querySelector('[data-slot="skeleton"]')).toBeInTheDocument()
    view.unmount()
  })

  it("reports outbound dependency errors and retries both queries", async () => {
    const calls = new Map<string, number>()
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      calls.set(path, (calls.get(path) ?? 0) + 1)
      return Promise.reject(new Error(`${path} unavailable`))
    }))
    const user = userEvent.setup()
    renderOutboundCards()
    expect(await screen.findByTestId("page-load-error")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重试" }))
    await waitFor(() => {
      expect(calls.get("/api/subscriptions/")).toBe(2)
      expect(calls.get("/api/nodes/groups")).toBe(2)
    })
  })

  it("builds configured, runtime, and independent outbound sections", async () => {
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const path = String(input)
      if (path === "/api/subscriptions/") {
        return Promise.resolve(new Response(JSON.stringify([
          { name: "fallback", outbounds: [{ tag: "node" }] },
          { name: "runtime", outbounds: [{ tag: "node" }] },
          { name: "runtime-unknown", outbounds: [{ tag: "node" }] },
          { name: "no-members", outbounds: [] },
          { name: "missing", outbounds: [{ tag: "node" }] },
        ])))
      }
      return Promise.resolve(new Response(JSON.stringify({ groups: [
        { type: "selector", tag: "runtime", now: "node", all: ["node"] },
        { type: "selector", tag: "runtime-unknown", now: "node", all: ["node"] },
      ] })))
    }))
    renderApp(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <OutboundCards
          items={[
            { index: 0, item: { tag: "fallback", type: "selector", outbounds: ["node"] } },
            { index: 1, item: { tag: "runtime", type: "custom" } },
            { index: 2, item: { tag: "runtime-unknown", type: 1 } },
            { index: 3, item: { tag: "independent", type: "direct" } },
          ]}
          onEdit={vi.fn()}
          onDelete={vi.fn()}
        />
      </QueryClientProvider>,
    )
    expect(await screen.findByRole("heading", { name: "independent" })).toBeInTheDocument()
    expect(screen.getAllByRole("combobox").length).toBeGreaterThanOrEqual(2)
  })

  it("forwards inbound quick patches", async () => {
    const onPatch = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <InboundCards
        items={[{ index: 0, item: { tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 } }]}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onPatch={onPatch}
      />,
    )
    await user.click(screen.getByRole("switch", { name: "设置系统代理" }))
    expect(onPatch).toHaveBeenCalledWith(0, expect.objectContaining({ set_system_proxy: true }))
  })
})
