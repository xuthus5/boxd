import { fireEvent, screen, waitFor } from "@testing-library/react"
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

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

interface SetupResponses {
  put?: { body: unknown; status?: number }
  validate?: { body: unknown; status?: number }
  configFailure?: { body: unknown; status?: number }
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
    return Promise.resolve(json({}))
  })
  vi.stubGlobal("fetch", fetchMock)
  return { fetchMock, user: userEvent.setup(), view: renderApp(<App />, "/advanced/certificate") }
}

const defaultConfig: SingBoxConfig = {
  log: { level: "info" },
  certificate: {
    store: "system",
    certificate: "PEM DATA",
    certificate_path: "/etc/boxd/ca.pem",
    future_option: { enabled: true },
  },
}

describe("certificate page", () => {
  it("renders the default store and normalizes scalar Listable values", async () => {
    setup(defaultConfig)
    expect(await screen.findByRole("heading", { name: "证书信任库" })).toBeInTheDocument()
    expect(screen.getByRole("combobox", { name: "默认信任库" })).toHaveTextContent("system")
    expect(screen.getByLabelText("内置 PEM 证书")).toHaveValue("PEM DATA")
    expect(screen.getByLabelText("证书文件路径")).toHaveValue("/etc/boxd/ca.pem")
    expect(screen.queryByText("当前信任库为空")).not.toBeInTheDocument()
  })

  it("uses the system trust store when the section is absent", async () => {
    const { fetchMock, user } = setup({ log: { level: "info" } })
    expect(await screen.findByRole("heading", { name: "证书信任库" })).toBeInTheDocument()
    expect(screen.getByRole("combobox", { name: "默认信任库" })).toHaveTextContent("system")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeEnabled()
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" })))
    const putCall = fetchMock.mock.calls.find(([url, init]) => String(url) === "/api/config/" && (init as RequestInit | undefined)?.method === "PUT")
    const body = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as SingBoxConfig
    expect(body).not.toHaveProperty("certificate")
  })

  it("warns when only custom certificate sources are trusted", async () => {
    setup({ certificate: { store: "none", certificate_path: "/etc/boxd/ca.pem" } })
    await screen.findByRole("heading", { name: "证书信任库" })
    expect(screen.getByText("仅使用自定义 CA")).toBeInTheDocument()
    expect(screen.queryByText("当前信任库为空")).not.toBeInTheDocument()
  })

  it("treats whitespace-only certificate sources as empty", async () => {
    setup({ certificate: { store: "none", certificate_path: ["", "  "] } })
    await screen.findByRole("heading", { name: "证书信任库" })
    expect(screen.getByText("当前信任库为空")).toBeInTheDocument()
    expect(screen.queryByText("仅使用自定义 CA")).not.toBeInTheDocument()
  })

  it("warns about an empty trust store and saves prepared fields", async () => {
    const { user, fetchMock } = setup({
      ...defaultConfig,
      certificate: { store: "none", future_option: { enabled: true } },
    })
    await screen.findByRole("heading", { name: "证书信任库" })
    expect(screen.getByText("当前信任库为空")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "默认信任库" }))
    await user.click(await screen.findByRole("option", { name: "system" }))
    const pem = "-----BEGIN CERTIFICATE-----\nPEM DATA\n-----END CERTIFICATE-----"
    fireEvent.change(screen.getByLabelText("内置 PEM 证书"), { target: { value: pem } })
    const paths = screen.getByLabelText("证书文件路径")
    fireEvent.change(paths, { target: { value: "/etc/boxd/ca.pem\n/etc/boxd/extra.pem" } })
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({ method: "PUT" })))
    const putCall = fetchMock.mock.calls.find(([url, init]) => String(url) === "/api/config/" && (init as RequestInit | undefined)?.method === "PUT")
    const body = JSON.parse(String((putCall?.[1] as RequestInit | undefined)?.body ?? "{}")) as SingBoxConfig
    expect(body.certificate).toEqual({
      certificate: [pem],
      certificate_path: ["/etc/boxd/ca.pem", "/etc/boxd/extra.pem"],
      future_option: { enabled: true },
    })
  })

  it("uses the certificate-specific dry-run source", async () => {
    const { user, fetchMock } = setup(defaultConfig)
    await screen.findByRole("heading", { name: "证书信任库" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/config/validate?source=validate_certificate",
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
          error: { code: "config_invalid_runtime", message: "certificate.store: invalid" },
          meta: {},
        },
      },
    })
    await screen.findByRole("heading", { name: "证书信任库" })
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    expect(await screen.findByTestId("config-save-error")).toHaveTextContent("certificate.store")
    expect(screen.getByRole("tab", { name: "高级 JSON" })).toHaveAttribute("aria-selected", "true")
  })

  it("reports save rollbacks and configuration load failures", async () => {
    const rollback = setup(defaultConfig, {
      put: { body: { status: "rolled_back", data: null, error: null, meta: {} } },
    })
    await screen.findByRole("heading", { name: "证书信任库" })
    await rollback.user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("配置保存未生效，后端已回滚。")).toBeInTheDocument()
    rollback.view.unmount()

    setup(defaultConfig, {
      configFailure: { body: { code: "internal_error", message: "certificate load failed" } },
    })
    expect(await screen.findByText("certificate load failed", {}, { timeout: 5000 })).toBeInTheDocument()
  })

  it("blocks malformed certificate Listable JSON", async () => {
    const { user } = setup(defaultConfig)
    await screen.findByRole("heading", { name: "证书信任库" })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = await screen.findByLabelText("证书信任库配置 JSON")
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste('{"certificate":{}}')
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "可视化配置" }))
    expect(screen.getByText("证书信任库结构无效")).toBeInTheDocument()
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}not-json")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })
})
