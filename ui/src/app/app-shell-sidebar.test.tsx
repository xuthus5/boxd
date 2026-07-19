import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => {
  sessionStore.clear()
  vi.unstubAllGlobals()
})

describe("collapsed sidebar hit targets", () => {
  it("keeps navigation icons clickable after collapsing", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    installMockAPI()
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 1400 })
    vi.stubGlobal("matchMedia", vi.fn().mockImplementation((query: string) => ({
      matches: true,
      media: query,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
    })))
    renderApp(<App />, "/dashboard")
    await screen.findByRole("heading", { name: "仪表盘" })
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: /toggle sidebar/i }))

    const labels = Array.from(document.querySelectorAll('[data-sidebar="group-label"]'))
    expect(labels.length).toBeGreaterThan(0)
    for (const label of labels) {
      expect((label as HTMLElement).className).toContain("pointer-events-none")
    }

    // Use aria-label from NavLink after collapse.
    const nodes = screen.getByRole("link", { name: "节点" })
    expect(nodes).toBeInTheDocument()
    await user.click(nodes)
    expect(await screen.findByRole("heading", { name: "节点" })).toBeInTheDocument()
  })
})
