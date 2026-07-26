import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

describe("proxy editor save failures", () => {
  it.each([
    {
      name: "request failure",
      status: 500,
      response: {
        status: "error",
        data: null,
        error: { code: "config_invalid_runtime", message: "outbounds[0].server: invalid" },
        meta: {},
      },
    },
    {
      name: "restart rollback",
      status: 200,
      response: {
        status: "rolled_back",
        data: null,
        error: { code: "config_restart_failed", message: "restart failed after config save" },
        meta: { rolled_back: true },
      },
    },
  ])("keeps the outbound draft after $name", async ({ response, status }) => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const config = {
      inbounds: [],
      outbounds: [{ tag: "proxy", type: "vless", server: "old.example.com", server_port: 443 }],
    }
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request, init?: RequestInit) => {
      const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
      if (path === "/api/config/" && init?.method === "PUT") {
        return Promise.resolve(new Response(JSON.stringify(response), { status }))
      }
      if (path === "/api/subscriptions/") return Promise.resolve(new Response(JSON.stringify([])))
      if (path === "/api/nodes/groups") return Promise.resolve(new Response(JSON.stringify({ groups: [] })))
      return Promise.resolve(new Response(JSON.stringify(config)))
    }))

    const user = userEvent.setup()
    renderApp(<App />, "/proxy/outbounds")
    await screen.findByText("proxy")
    await user.click(screen.getByRole("button", { name: "编辑" }))
    const server = await screen.findByLabelText("服务器地址")
    await user.clear(server)
    await user.type(server, "draft.example.com")
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(await screen.findByTestId("config-save-error")).toBeInTheDocument()
    expect(screen.getByRole("dialog")).toBeInTheDocument()
    expect(screen.getByLabelText("服务器地址")).toHaveValue("draft.example.com")
  })
})
