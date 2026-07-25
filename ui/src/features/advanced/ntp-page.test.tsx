import { screen, waitFor } from "@testing-library/react"
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

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

const defaultConfig: SingBoxConfig = {
  log: { level: "info" },
  ntp: {
    enabled: true,
    server: "time.google.com",
    server_port: 123,
    interval: "30m",
    write_to_system: true,
    detour: "direct",
    domain_resolver: { server: "dns-local" },
  },
  dns: { servers: [{ type: "local", tag: "dns-local" }] },
  outbounds: [{ type: "direct", tag: "direct" }],
}

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
    return Promise.resolve(json({}))
  })
  vi.stubGlobal("fetch", fetchMock)
  return { user: userEvent.setup(), fetchMock, view: renderApp(<App />, "/advanced/ntp") }
}

describe("NTP page", () => {
  it("renders visual and advanced configuration with the system-time warning", async () => {
    setup()
    expect(await screen.findByRole("heading", { name: "NTP 时间同步" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "可视化配置" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toBeInTheDocument()
    expect(screen.getByText("NTP 基本设置")).toBeInTheDocument()
    expect(screen.getByText("网络拨号")).toBeInTheDocument()
    expect(screen.getByText("写入系统时间需要 CAP_SYS_TIME 权限")).toBeInTheDocument()
  })

  it("saves visual edits with the prepared NTP section", async () => {
    const { user, fetchMock } = setup()
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    const server = screen.getByLabelText("NTP 服务器")
    await user.clear(server)
    await user.type(server, "time.cloudflare.com")
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" })))
    const putCall = fetchMock.mock.calls.find(([url, init]) => String(url) === "/api/config/" && (init as RequestInit | undefined)?.method === "PUT")
    const body = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as {
      ntp?: { server?: string; enabled?: boolean; domain_resolver?: { server?: string } }
    }
    expect(body.ntp?.enabled).toBe(true)
    expect(body.ntp?.server).toBe("time.cloudflare.com")
    expect(body.ntp?.domain_resolver?.server).toBe("dns-local")
  })

  it("records the NTP dry-run source", async () => {
    const { user, fetchMock } = setup()
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/config/validate?source=validate_ntp",
      expect.objectContaining({ method: "POST" }),
    ))
  })

  it("hides dialer options while disabled and restores them when enabled", async () => {
    const { user } = setup({ ntp: { enabled: false } })
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    expect(screen.queryByText("网络拨号")).not.toBeInTheDocument()
    expect(screen.queryByText("写入系统时间需要 CAP_SYS_TIME 权限")).not.toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "启用 NTP 时间同步" }))
    expect(await screen.findByText("网络拨号")).toBeInTheDocument()
    expect(screen.queryByText("写入系统时间需要 CAP_SYS_TIME 权限")).not.toBeInTheDocument()
  })

  it("uses an empty disabled draft when the NTP section is absent", async () => {
    setup({ log: { level: "info" } })
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    expect(screen.getByRole("switch", { name: "启用 NTP 时间同步" })).not.toBeChecked()
    expect(screen.getByRole("button", { name: "保存配置" })).toBeEnabled()
  })

  it("blocks invalid numeric drafts and recovers after correction", async () => {
    const { user } = setup()
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    const port = screen.getByLabelText("服务端口")
    await user.clear(port)
    await user.type(port, "65536")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
    expect(screen.getByText("请输入支持范围内的有效值。")).toBeInTheDocument()
    await user.clear(port)
    await user.type(port, "123")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeEnabled()
  })

  it("rejects malformed resolver structures in the visual editor", async () => {
    setup({ ntp: { enabled: true, domain_resolver: 42 } })
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    expect(screen.getByText("NTP 结构无效")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })

  it("surfaces validation failures without changing tabs when no path is available", async () => {
    const { user } = setup(defaultConfig, {
      validate: {
        status: 400,
        body: { status: "error", data: null, error: { code: "config_invalid_runtime", message: "NTP validation failed" }, meta: {} },
      },
    })
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("NTP validation failed")
    expect(screen.getByRole("tab", { name: "可视化配置" })).toHaveAttribute("aria-selected", "true")
  })

  it("reports save rollbacks", async () => {
    const { user } = setup(defaultConfig, {
      put: { body: { status: "rolled_back", data: null, error: null, meta: {} } },
    })
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("配置保存未生效，后端已回滚。")).toBeInTheDocument()
  })

  it("jumps from pathful save errors to advanced JSON", async () => {
    const { user } = setup(defaultConfig, {
      put: {
        status: 400,
        body: {
          status: "error",
          data: null,
          error: { code: "config_invalid_runtime", message: "ntp.server_port: invalid" },
          meta: {},
        },
      },
    })
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("ntp.server_port")
    await user.click(screen.getByRole("button", { name: "跳转到路径" }))
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toHaveAttribute("aria-selected", "true")
  })

  it("shows configuration load failures", async () => {
    setup(defaultConfig, { configFailure: { body: { code: "internal_error", message: "ntp load failed" } } })
    expect(await screen.findByText("ntp load failed", {}, { timeout: 5000 })).toBeInTheDocument()
  })

  it("disables saving for an invalid JSON root", async () => {
    const { user } = setup()
    await screen.findByRole("heading", { name: "NTP 时间同步" })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = await screen.findByLabelText("NTP 配置 JSON")
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}[BracketLeft][BracketRight]")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
    await user.keyboard("{Control>}a{/Control}not-json")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })
})
