import { fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("outbound editor", () => {
  it("edits server fields instead of inbound listen fields", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((_input: string | URL | Request, init?: RequestInit) => Promise.resolve(new Response(JSON.stringify(
      init?.method === "PUT" ? { status: "ok", data: null, error: null, meta: null } : { outbounds: [{ tag: "proxy", type: "vless", server: "old.example.com", server_port: 443 }] },
    ))))
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderApp(<App />, "/proxy/outbounds")
    await user.click(await screen.findByRole("button", { name: "编辑" }))
    expect(screen.getByLabelText("服务器地址")).toHaveValue("old.example.com")
    fireEvent.change(screen.getByLabelText("服务器地址"), { target: { value: "new.example.com" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({
      method: "PUT",
      body: expect.stringContaining('"server":"new.example.com"'),
    })))
  })
})
