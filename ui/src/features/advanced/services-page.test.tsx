import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import type { SingBoxConfig } from "@/lib/api/types"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

const okEnvelope = { status: "ok", data: null, error: null, meta: {} }

interface SetupResponses {
  put?: { body: unknown; status?: number }
  validate?: { body: unknown; status?: number }
  configFailure?: { body: unknown; status?: number }
}

const defaultConfig: SingBoxConfig = {
  log: { level: "info" },
  inbounds: [{ type: "shadowsocks", tag: "ss-in" }],
  outbounds: [{ type: "direct", tag: "direct" }],
  services: [{ type: "resolved", tag: "local-dns" }],
}

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function setup(config: SingBoxConfig = defaultConfig, responses: SetupResponses = {}) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.pathname : input.url
    const url = new URL(raw, "http://boxd.test")
    if (url.pathname === "/api/config/validate") {
      const response = responses.validate
      return Promise.resolve(response ? json(response.body, response.status) : json(okEnvelope))
    }
    if (url.pathname === "/api/config/" && init?.method === "PUT") {
      const response = responses.put
      return Promise.resolve(response ? json(response.body, response.status) : json(okEnvelope))
    }
    if (url.pathname === "/api/config/" || url.pathname === "/api/config/raw") {
      const response = responses.configFailure
      return Promise.resolve(response ? json(response.body, response.status ?? 500) : json(config))
    }
    if (url.pathname === "/api/network/interfaces") return Promise.resolve(json({ interfaces: [] }))
    if (url.pathname === "/api/settings/password") return Promise.resolve(json({ defaultPassword: false }))
    return Promise.resolve(json({}))
  })
  vi.stubGlobal("fetch", fetchMock)
  return { fetchMock, user: userEvent.setup(), view: renderApp(<App />, "/advanced/services") }
}

describe("services page", () => {
  it("renders normalized service cards and visual JSON tabs", async () => {
    setup({
      ...defaultConfig,
      services: [
        { type: "ccm", tag: "claude", listen: "127.0.0.1", listen_port: 8081 },
        { type: "derp", tag: "relay", listen: "::", listen_port: 443, config_path: "derper.key" },
        { type: "ocm", tag: "codex", listen: "127.0.0.1", listen_port: 8082 },
        { type: "resolved", tag: "dns" },
        { type: "ssm-api", tag: "manager", listen: "127.0.0.1", listen_port: 8083, servers: { "/": "ss-in" } },
      ],
    })
    expect(await screen.findByRole("heading", { name: "内核服务" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "可视化配置" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toBeInTheDocument()
    for (const tag of ["claude", "relay", "codex", "dns", "manager"]) {
      expect(screen.getByRole("heading", { name: tag })).toBeInTheDocument()
    }
    expect(screen.getAllByText("127.0.0.53:53").length).toBeGreaterThan(0)
    expect(screen.getByText("1 个 Shadowsocks 映射")).toBeInTheDocument()
  })

  it("adds, edits, deletes, and saves an empty services section", async () => {
    const { fetchMock, user } = setup({ ...defaultConfig, services: [] })
    await screen.findByRole("heading", { name: "内核服务" })
    await user.click(screen.getAllByRole("button", { name: "新增服务" })[0])
    let dialog = await screen.findByRole("dialog")
    expect(within(dialog).getByLabelText("监听地址")).toHaveValue("127.0.0.53")
    await user.type(within(dialog).getByLabelText("Tag"), "resolver")
    await user.click(within(dialog).getByRole("button", { name: "保存" }))
    expect(await screen.findByRole("heading", { name: "resolver" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "编辑" }))
    dialog = await screen.findByRole("dialog")
    const tag = within(dialog).getByLabelText("Tag")
    await user.clear(tag)
    await user.type(tag, "resolver-local")
    await user.click(within(dialog).getByRole("button", { name: "保存" }))
    expect(await screen.findByRole("heading", { name: "resolver-local" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(await screen.findByRole("button", { name: "确认删除" }))
    expect(await screen.findByText("暂无内核服务")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" })))
    const putCall = fetchMock.mock.calls.find(([url, init]) => String(url) === "/api/config/" && (init as RequestInit | undefined)?.method === "PUT")
    const body = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as SingBoxConfig
    expect(body).not.toHaveProperty("services")
  })

  it("uses the services-specific dry-run source", async () => {
    const { fetchMock, user } = setup()
    await screen.findByRole("heading", { name: "内核服务" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/config/validate?source=validate_services",
      expect.objectContaining({ method: "POST" }),
    ))
    expect(fetchMock.mock.calls.some((call) => String(call[0]) === "/api/config/" && call[1]?.method === "PUT")).toBe(false)
  })

  it("reveals pathful validation errors in advanced JSON", async () => {
    const { user } = setup(defaultConfig, {
      validate: {
        status: 400,
        body: {
          status: "error",
          data: null,
          error: { code: "config_invalid_runtime", message: "services[0].listen_port: invalid" },
          meta: {},
        },
      },
    })
    await screen.findByRole("heading", { name: "内核服务" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("services[0].listen_port")
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toHaveAttribute("aria-selected", "true")
  })

  it("reports save rollbacks and configuration load failures", async () => {
    const rollback = setup(defaultConfig, {
      put: { body: { status: "rolled_back", data: null, error: null, meta: {} } },
    })
    await screen.findByRole("heading", { name: "内核服务" })
    await rollback.user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("配置保存未生效，后端已回滚。")).toBeInTheDocument()
    rollback.view.unmount()

    setup(defaultConfig, {
      configFailure: { body: { code: "internal_error", message: "services load failed" } },
    })
    expect(await screen.findByText("services load failed", {}, { timeout: 5000 })).toBeInTheDocument()
  })

  it("surfaces save request failures", async () => {
    const { user } = setup(defaultConfig, {
      put: { status: 500, body: { code: "internal_error", message: "services save failed" } },
    })
    await screen.findByRole("heading", { name: "内核服务" })
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("services save failed")
  })

  it("preserves invalid roots for repair and blocks malformed JSON", async () => {
    const invalidConfig = { ...defaultConfig, services: { type: "resolved" } } as unknown as SingBoxConfig
    const { user } = setup(invalidConfig)
    await screen.findByRole("heading", { name: "内核服务" })
    expect(screen.getByText("Services 结构无效")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()

    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = await screen.findByLabelText("内核服务 JSON")
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste("[]")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeEnabled()
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}not-json")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })
})
