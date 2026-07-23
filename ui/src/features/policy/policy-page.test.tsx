import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { PolicyPage, type PolicyVisualEditorProps } from "@/features/policy/policy-page"
import type { APIEnvelope, JsonValue, SingBoxConfig } from "@/lib/api/types"
import { renderApp } from "@/test/render"

const okEnvelope: APIEnvelope<JsonValue> = {
  status: "ok",
  data: null,
  error: null,
  meta: {},
}

afterEach(() => vi.unstubAllGlobals())

function renderPolicy(renderVisual: (props: PolicyVisualEditorProps) => React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  renderApp(
    <QueryClientProvider client={queryClient}>
      <PolicyPage
        section="route"
        title="路由"
        installLabel="安装默认路由"
        install={() => Promise.resolve(okEnvelope)}
        renderVisual={renderVisual}
      />
    </QueryClientProvider>,
  )
}

function stubConfig(config: SingBoxConfig) {
  const fetchMock = vi.fn((_input: string | URL | Request, init?: RequestInit) => (
    Promise.resolve(new Response(JSON.stringify(init?.method === "PUT" ? okEnvelope : config)))
  ))
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

describe("PolicyPage editor shell", () => {
  it("opens the visual editor before the advanced JSON tab", async () => {
    const config = { route: { final: "proxy" } }
    stubConfig(config)
    renderPolicy(({ object }) => <p>可视化内容：{String(object.final)}</p>)

    expect(await screen.findByText("可视化内容：proxy")).toBeInTheDocument()
    const tabs = screen.getAllByRole("tab")
    expect(tabs.map((tab) => tab.textContent)).toEqual(["可视化配置", "高级 JSON"])
    expect(tabs[0]).toHaveAttribute("aria-selected", "true")
    expect(tabs[0].closest('[data-slot="tabs-list"]')).toHaveClass("overflow-x-auto")
  })

  it("disables saving when advanced JSON is not an object", async () => {
    const config = { route: { final: "proxy" } }
    stubConfig(config)
    const user = userEvent.setup()
    renderPolicy(() => null)

    await user.click(await screen.findByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "流量策略 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}[BracketLeft][BracketRight]")

    expect(screen.getByRole("button", { name: "保存配置" })).toHaveClass("h-8")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })

  it("updates the visual editor object and revision after JSON edits", async () => {
    const config = { route: { final: "proxy" } }
    stubConfig(config)
    const user = userEvent.setup()
    renderPolicy(({ object, revision }) => (
      <p>revision:{revision}; keys:{Object.keys(object).length}</p>
    ))

    await user.click(await screen.findByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "流量策略 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste("{}")
    await user.click(screen.getByRole("tab", { name: "可视化配置" }))

    expect(screen.getByText(/revision:[1-9]\d*; keys:0/)).toBeInTheDocument()
  })

  it("disables saving while a structured field is invalid", async () => {
    const config = { route: { final: "proxy" } }
    stubConfig(config)
    const user = userEvent.setup()
    renderPolicy(({ onFieldValidityChange }) => (
      <>
        <button onClick={() => onFieldValidityChange("rules.0.action", false)}>标记无效</button>
        <button onClick={() => onFieldValidityChange("rules.0.action", true)}>恢复有效</button>
      </>
    ))

    const saveButton = await screen.findByRole("button", { name: "保存配置" })
    await user.click(screen.getByRole("button", { name: "标记无效" }))
    expect(saveButton).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "恢复有效" }))
    expect(saveButton).toBeEnabled()
  })

  it("clears structured invalid fields when advanced JSON replaces the section", async () => {
    const config = { route: { headers: { invalid: true } } }
    stubConfig(config)
    const user = userEvent.setup()
    renderPolicy(({ onFieldValidityChange }) => (
      <button onClick={() => onFieldValidityChange("headers", false)}>标记 Headers 无效</button>
    ))

    const saveButton = await screen.findByRole("button", { name: "保存配置" })
    await user.click(screen.getByRole("button", { name: "标记 Headers 无效" }))
    expect(saveButton).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "流量策略 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste("{}")

    expect(saveButton).toBeEnabled()
  })

  it("replaces only the edited section in the saved config", async () => {
    const config = {
      route: { final: "proxy", custom: { retained: true } },
      dns: { servers: [{ tag: "local", address: "local" }] },
      log: { level: "info" },
    }
    const fetchMock = stubConfig(config)
    const user = userEvent.setup()
    renderPolicy(({ object, onChange }) => (
      <button onClick={() => onChange({ ...object, final: "direct" })}>修改路由</button>
    ))

    await user.click(await screen.findByRole("button", { name: "修改路由" }))
    await user.click(screen.getByRole("button", { name: "保存配置" }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({
      method: "PUT",
      body: JSON.stringify({
        route: { final: "direct", custom: { retained: true } },
        dns: config.dns,
        log: config.log,
      }),
    })))
  })
})
