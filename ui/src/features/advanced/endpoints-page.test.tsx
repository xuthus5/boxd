import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import type { SingBoxConfig } from "@/lib/api/types"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

const okEnvelope = { status: "ok", data: null, error: null, meta: {} }

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function setup(config: SingBoxConfig = {
  log: { level: "info" },
  endpoints: [{
    type: "wireguard",
    tag: "wg-home",
    address: ["10.0.0.2/32"],
    private_key: "private",
    peers: [{ public_key: "peer", allowed_ips: ["0.0.0.0/0"] }],
  }],
  outbounds: [{ type: "direct", tag: "direct" }],
  dns: { servers: [{ tag: "local", type: "local" }] },
}) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const path = String(typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url).split("?")[0]
    if (path === "/api/config/" && init?.method === "PUT") {
      return Promise.resolve(new Response(JSON.stringify(okEnvelope)))
    }
    if (path === "/api/config/" || path === "/api/config/raw") {
      return Promise.resolve(new Response(JSON.stringify(config)))
    }
    if (path === "/api/network/interfaces") {
      return Promise.resolve(new Response(JSON.stringify({ interfaces: [{ name: "eth0", ips: ["10.0.0.2"] }] })))
    }
    return Promise.resolve(new Response(JSON.stringify({})))
  })
  vi.stubGlobal("fetch", fetchMock)
  return { user: userEvent.setup(), fetchMock, view: renderApp(<App />, "/advanced/endpoints") }
}

describe("endpoints page", () => {
  it("renders visual and advanced tabs with endpoint cards", async () => {
    setup()
    expect(await screen.findByRole("heading", { name: "Endpoints" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "可视化配置" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "wg-home" })).toBeInTheDocument()
    expect(screen.getByText("wireguard")).toBeInTheDocument()
  })

  it("adds a wireguard endpoint from the visual editor and saves", async () => {
    const { user, fetchMock } = setup({ log: { level: "info" }, endpoints: [] })
    await screen.findByRole("heading", { name: "Endpoints" })
    await user.click(screen.getAllByRole("button", { name: "新增 Endpoint" })[0])
    const dialog = await screen.findByRole("dialog")
    await user.type(within(dialog).getByLabelText("Tag"), "wg-new")
    await user.type(within(dialog).getByLabelText("本端地址"), "10.8.0.2/32")
    await user.type(within(dialog).getByLabelText("私钥"), "secret-key")
    await user.click(within(dialog).getByRole("button", { name: "保存" }))
    expect(await screen.findByRole("heading", { name: "wg-new" })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" }))
    })
    const putCall = fetchMock.mock.calls.find(([url, init]) => String(url) === "/api/config/" && (init as RequestInit | undefined)?.method === "PUT")
    const body = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as {
      endpoints?: Array<{ tag?: string; type?: string; private_key?: string }>
    }
    expect(body.endpoints?.[0]?.tag).toBe("wg-new")
    expect(body.endpoints?.[0]?.type).toBe("wireguard")
    expect(body.endpoints?.[0]?.private_key).toBe("secret-key")
  })

  it("switches to advanced JSON and disables save for non-array values", async () => {
    const { user } = setup()
    await screen.findByRole("heading", { name: "Endpoints" })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = await screen.findByLabelText("Endpoints 配置 JSON")
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}{Backspace}{{}")
    await user.keyboard("{Shift>}}{/Shift}")
    // invalid JSON or object should disable save; write a plain object
    await user.keyboard("{Control>}a{/Control}{Backspace}")
    await user.paste("{}")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })
})
