import { screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("proxy configuration pages", () => {
  it("renders each inbound configuration as a card", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      inbounds: [
        { tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 },
        { tag: "tun-in", type: "tun", interface_name: "tun0" },
      ],
      outbounds: [],
    }))))

    renderApp(<App />, "/proxy/inbounds")

    expect(await screen.findByText("mixed-in")).toBeInTheDocument()
    expect(screen.getByText("tun-in")).toBeInTheDocument()
    expect(screen.getAllByRole("article")).toHaveLength(2)
    expect(screen.queryByRole("table")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "新增入站" })).toBeInTheDocument()
  })
})
