import { fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import type { SingBoxConfig } from "@/lib/api/types"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

const okEnvelope = { status: "ok", data: null, error: null, meta: {} }
const defaultConfig: SingBoxConfig = {
  log: {
    level: "info",
    output: "/var/log/sing-box.log",
    timestamp: true,
    future_option: { enabled: true },
  },
}

interface SetupResponses {
  put?: { body: unknown; status?: number }
  validate?: { body: unknown; status?: number }
  configFailure?: { body: unknown; status?: number }
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function requestPath(input: string | URL | Request) {
  const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url
  return new URL(raw, "http://boxd.test")
}

function setup(config: SingBoxConfig = defaultConfig, responses: SetupResponses = {}) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const url = requestPath(input)
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
    return Promise.resolve(json({}))
  })
  vi.stubGlobal("fetch", fetchMock)
  return { fetchMock, user: userEvent.setup(), view: renderApp(<App />, "/advanced/log") }
}

function savedConfig(fetchMock: ReturnType<typeof vi.fn>) {
  const call = fetchMock.mock.calls.find(([url, init]) => (
    String(url) === "/api/config/" && (init as RequestInit | undefined)?.method === "PUT"
  ))
  return JSON.parse(String((call?.[1] as RequestInit | undefined)?.body ?? "{}")) as SingBoxConfig
}

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

describe("log page", () => {
  it("renders structured log settings", async () => {
    setup()
    expect(await screen.findByRole("heading", { name: "内核日志配置" })).toBeInTheDocument()
    expect(screen.getByRole("switch", { name: "禁用内核日志" })).not.toBeChecked()
    expect(screen.getByRole("combobox", { name: "日志级别" })).toHaveTextContent("info")
    expect(screen.getByLabelText("输出目标")).toHaveValue("/var/log/sing-box.log")
    expect(screen.getByRole("switch", { name: "完整时间戳" })).toBeChecked()
  })

  it("keeps an absent log section absent when saved unchanged", async () => {
    const { fetchMock, user } = setup({ outbounds: [] })
    await screen.findByRole("heading", { name: "内核日志配置" })
    expect(screen.getByRole("combobox", { name: "日志级别" })).toHaveTextContent("未设置")
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" })))
    expect(savedConfig(fetchMock)).toEqual({ outbounds: [] })
  })

  it("saves prepared fields and preserves unknown JSON", async () => {
    const { fetchMock, user } = setup()
    await screen.findByRole("heading", { name: "内核日志配置" })
    await user.click(screen.getByRole("switch", { name: "禁用内核日志" }))
    expect(screen.getByText("内核日志已禁用")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "日志级别" }))
    await user.click(await screen.findByRole("option", { name: "warn" }))
    fireEvent.change(screen.getByLabelText("输出目标"), { target: { value: " /tmp/kernel.log " } })
    await user.click(screen.getByRole("switch", { name: "完整时间戳" }))
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" })))
    expect(savedConfig(fetchMock).log).toEqual({
      disabled: true,
      level: "warn",
      output: "/tmp/kernel.log",
      future_option: { enabled: true },
    })
  })

  it("uses the log-specific dry-run source", async () => {
    const { fetchMock, user } = setup()
    await screen.findByRole("heading", { name: "内核日志配置" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/config/validate?source=validate_log",
      expect.objectContaining({ method: "POST" }),
    ))
  })

  it("reveals pathful validation failures in advanced JSON", async () => {
    const { user } = setup(defaultConfig, {
      validate: {
        status: 400,
        body: {
          status: "error",
          data: null,
          error: { code: "config_invalid_runtime", message: "log.level: unknown log level" },
          meta: {},
        },
      },
    })
    await screen.findByRole("heading", { name: "内核日志配置" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("log.level")
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toHaveAttribute("aria-selected", "true")
  })

  it("blocks malformed log JSON and known field types", async () => {
    const { user } = setup()
    await screen.findByRole("heading", { name: "内核日志配置" })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = await screen.findByLabelText("内核日志配置 JSON")
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste('{"level":"verbose"}')
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "可视化配置" }))
    expect(screen.getByText("日志配置结构无效")).toBeInTheDocument()
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}not-json")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })

  it("reports rollbacks, save failures, and load failures", async () => {
    const rollback = setup(defaultConfig, {
      put: { body: { status: "rolled_back", data: null, error: null, meta: {} } },
    })
    await screen.findByRole("heading", { name: "内核日志配置" })
    await rollback.user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("配置保存未生效，后端已回滚。")).toBeInTheDocument()
    rollback.view.unmount()

    const failed = setup(defaultConfig, {
      put: { status: 500, body: { code: "internal_error", message: "log save failed" } },
    })
    await screen.findByRole("heading", { name: "内核日志配置" })
    await failed.user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("log save failed")
    failed.view.unmount()

    setup(defaultConfig, {
      configFailure: { body: { code: "internal_error", message: "log load failed" } },
    })
    expect(await screen.findByText("log load failed", {}, { timeout: 5000 })).toBeInTheDocument()
  })
})
