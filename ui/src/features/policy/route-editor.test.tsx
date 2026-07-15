import { useState } from "react"
import { fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { RouteRuleSetDialog } from "@/features/policy/route-rule-set-dialog"
import { RouteVisualEditor } from "@/features/policy/route-visual-editor"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
})

function EditorHarness({ initial }: { initial: JsonObject }) {
  const [object, setObject] = useState(initial)
  const props: PolicyVisualEditorProps = {
    object,
    revision: 0,
    onChange: setObject,
    onFieldValidityChange: vi.fn(),
  }
  const rules = Array.isArray(object.rules) ? object.rules : []
  return <><RouteVisualEditor {...props} /><output aria-label="route state">{JSON.stringify(object)}</output>
    <output aria-label="rule identity">{String(rules.length > 1 && rules[0] === rules[1])}</output></>
}

async function choose(label: string, option: string) {
  const user = userEvent.setup()
  await user.click(screen.getByRole("combobox", { name: label }))
  await user.click(await screen.findByRole("option", { name: option }))
}

const actionCases: readonly [string, JsonObject, string | null][] = [
  ["route", { action: "route", outbound: "proxy" }, "目标出站"],
  ["route-options", { action: "route-options" }, "覆盖目标地址"],
  ["direct", { action: "direct" }, "绑定接口"],
  ["bypass", { action: "bypass", outbound: "direct" }, "目标出站"],
  ["reject", { action: "reject" }, "拒绝方式"],
  ["hijack-dns", { action: "hijack-dns" }, null],
  ["sniff", { action: "sniff" }, "嗅探器"],
  ["resolve", { action: "resolve", server: "dns-remote" }, "解析服务器"],
]

const routeFixture = {
  final: "proxy", find_process: false, auto_detect_interface: false,
  override_android_vpn: false, default_interface: "eth0", default_mark: 100,
  default_domain_resolver: {
    server: "dns-old", strategy: "prefer_ipv4", disable_cache: false,
    rewrite_ttl: 60, client_subnet: "192.0.2.0/24", custom: "nested",
  },
  default_network_strategy: "default", default_network_type: ["wifi"],
  default_fallback_network_type: ["cellular"], default_fallback_delay: "300ms",
  rules: [{ action: "reject" }], rule_set: [{ type: "inline", tag: "inline", rules: [] }],
  custom: { retained: true },
}

const configFixture = { route: routeFixture, dns: { final: "dns" }, log: { level: "info" } }

function expectedSavedConfig() {
  return {
    ...configFixture,
    route: {
      ...routeFixture,
      final: "direct", find_process: true, auto_detect_interface: true,
      override_android_vpn: true, default_network_strategy: "hybrid",
      default_network_type: ["wifi", "ethernet"],
      default_fallback_network_type: ["cellular", "wifi"], default_fallback_delay: "350ms",
      default_domain_resolver: {
        ...routeFixture.default_domain_resolver,
        server: "dns-new", strategy: "prefer_ipv6", disable_cache: true,
      },
    },
  }
}

function stubRouteConfig() {
  const fetchMock = vi.fn((_input: string | URL | Request, init?: RequestInit) => Promise.resolve(
    new Response(JSON.stringify(init?.method === "PUT"
      ? { status: "ok", data: null, error: null, meta: {} }
      : configFixture)),
  ))
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

async function changeRouteGlobals() {
  const user = userEvent.setup()
  fireEvent.change(await screen.findByLabelText("最终出站"), { target: { value: "direct" } })
  for (const name of ["查找进程", "自动检测接口", "覆盖 Android VPN", "禁用默认解析缓存"]) {
    await user.click(screen.getByRole("switch", { name }))
  }
  fireEvent.change(screen.getByLabelText("默认域名解析服务器"), { target: { value: "dns-new" } })
  await choose("默认域名解析策略", "prefer_ipv6")
  await choose("默认网络策略", "hybrid")
  fireEvent.change(screen.getByLabelText("默认网络类型"), { target: { value: "wifi\nethernet" } })
  fireEvent.change(screen.getByLabelText("默认回退网络类型"), { target: { value: "cellular\nwifi" } })
  fireEvent.change(screen.getByLabelText("默认回退延迟"), { target: { value: "350ms" } })
  await user.click(screen.getByRole("button", { name: "保存配置" }))
}

describe("route global editor", () => {
  it("saves changed globals while preserving the complete config", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = stubRouteConfig()
    renderApp(<App />, "/policy/route")
    await changeRouteGlobals()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({
      method: "PUT", body: JSON.stringify(expectedSavedConfig()),
    })))
  })
})

describe("route rule dialog", () => {
  it("offers every match tab and supported action with required action values", async () => {
    renderApp(<RouteRuleDialog open item={{}} title="新增规则" onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByRole("dialog")).toHaveClass("sm:max-w-5xl")
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "基础与网络", "域名与地址", "端口与进程", "规则集与网络环境", "执行动作", "高级 JSON",
    ])
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    await userEvent.click(screen.getByRole("combobox", { name: "执行动作" }))
    for (const action of ["route", "route-options", "direct", "bypass", "reject", "hijack-dns", "sniff", "resolve"]) {
      expect(await screen.findByRole("option", { name: action })).toBeInTheDocument()
    }
    await userEvent.click(screen.getByRole("option", { name: "route" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("目标出站"), { target: { value: "proxy" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })

  it("requires the resolver server and exposes action-specific fields", async () => {
    renderApp(<RouteRuleDialog open item={{ action: "resolve" }} title="编辑规则" onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    expect(screen.getByLabelText("解析服务器")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("解析服务器"), { target: { value: "dns-remote" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    await choose("执行动作", "direct")
    expect(screen.getByLabelText("绑定接口")).toBeInTheDocument()
  })

  it.each(actionCases)("edits the %s action", async (action, item, fieldLabel) => {
    renderApp(<RouteRuleDialog open item={item} title={`编辑 ${action}`} onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    expect(screen.getByRole("combobox", { name: "执行动作" })).toHaveTextContent(action)
    if (fieldLabel) expect(screen.getByLabelText(fieldLabel)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })

  it("keeps logical child JSON invalidity blocking save across tabs", async () => {
    renderApp(<RouteRuleDialog open title="编辑规则" item={{
      type: "logical", mode: "and", rules: [{ action: "reject" }], invert: false, action: "reject",
    }} onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByRole("combobox", { name: "逻辑模式" })).toHaveTextContent("and")
    await userEvent.click(screen.getByRole("switch", { name: "反向匹配" }))
    fireEvent.change(screen.getByLabelText("子规则 JSON"), { target: { value: "invalid" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await userEvent.click(screen.getByRole("tab", { name: "基础与网络" }))
    fireEvent.change(screen.getByLabelText("子规则 JSON"), { target: { value: "[{}]" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })

  it("uses the immutable type transition when changing a logical rule to default", async () => {
    const onSave = vi.fn()
    renderApp(<RouteRuleDialog open title="编辑规则" item={{
      type: "logical", mode: "or", rules: [{ action: "reject" }], invert: true,
      action: "reject", custom: "keep",
    }} onOpenChange={vi.fn()} onSave={onSave} />)
    await choose("规则类型", "default")
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ invert: true, action: "reject", custom: "keep" })
  })

  it("blocks save for invalid or non-object advanced JSON", async () => {
    const user = userEvent.setup()
    renderApp(<RouteRuleDialog open item={{ action: "reject" }} title="编辑规则" onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "编辑规则 JSON" })
    await user.click(editor)
    await user.keyboard("{Control>}a{/Control}")
    await user.paste("[")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await user.keyboard("{Control>}a{/Control}")
    await user.paste("[]")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })
})

describe("route rule cards", () => {
  it("adds defaults, summarizes, copies deeply, moves adjacently, and confirms deletion", async () => {
    const user = userEvent.setup()
    renderApp(<EditorHarness initial={{}} />)
    expect(screen.getByText("暂无路由规则")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "新增规则" })[0])
    await user.click(screen.getByRole("tab", { name: "执行动作" }))
    fireEvent.change(screen.getByLabelText("目标出站"), { target: { value: "proxy" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(screen.getByLabelText("route state")).toHaveTextContent('"action":"route"')

    await user.click(screen.getByRole("button", { name: "编辑规则 1" }))
    await user.click(screen.getByRole("tab", { name: "域名与地址" }))
    fireEvent.change(screen.getByLabelText("域名后缀"), { target: { value: "example.com\nexample.org" } })
    await user.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expect(screen.getByRole("textbox", { name: "编辑规则 1 JSON" })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(screen.getByText("example.com")).toBeInTheDocument()
    expect(screen.getByText("example.org")).toBeInTheDocument()
    expect(screen.getByText("proxy")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "复制规则 1" }))
    expect(screen.getAllByText(/规则 #/)).toHaveLength(2)
    expect(screen.getByLabelText("rule identity")).toHaveTextContent("false")
    expect(screen.getByRole("button", { name: "上移规则 1" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "下移规则 2" })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "编辑规则 2" }))
    await user.click(screen.getByRole("tab", { name: "执行动作" }))
    await choose("执行动作", "reject")
    await user.click(screen.getByRole("button", { name: "保存" }))
    await user.click(screen.getByRole("button", { name: "下移规则 1" }))
    const moved = JSON.parse(screen.getByLabelText("route state").textContent ?? "{}")
    expect(moved.rules.map((rule: JsonObject) => rule.action)).toEqual(["reject", "route"])

    await user.click(screen.getByRole("button", { name: "删除规则 1" }))
    expect(screen.getByRole("alertdialog")).toBeInTheDocument()
    const state = document.querySelector('output[aria-label="route state"]')
    expect(JSON.parse(state?.textContent ?? "{}").rules).toHaveLength(2)
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(screen.getAllByRole("button", { name: /编辑规则/ })).toHaveLength(1)
  }, 15_000)
})

describe("route rule-set editor", () => {
  it("validates type-driven fields and preserves unknown keys on transitions", async () => {
    const onSave = vi.fn()
    renderApp(<RouteRuleSetDialog open title="编辑规则集" item={{
      type: "remote", tag: "geo", url: "https://old/r.srs", update_interval: "1d", custom: "keep",
    }} onOpenChange={vi.fn()} onSave={onSave} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    await choose("规则集类型", "local")
    expect(screen.queryByLabelText("远程 URL")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("本地路径"), { target: { value: "/etc/geo.srs" } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ type: "local", tag: "geo", path: "/etc/geo.srs", custom: "keep" })
  })

  it("supports inline and unknown types while rule-set cards never show movement", async () => {
    const user = userEvent.setup()
    renderApp(<EditorHarness initial={{ rule_set: [
      { type: "inline", tag: "inline", rules: [{ domain: ["example.com"] }] },
      { type: "custom", tag: "future", format: "binary", payload: { enabled: true } },
    ] }} />)
    expect(screen.getAllByText("inline").length).toBeGreaterThan(0)
    expect(screen.getAllByText("future").length).toBeGreaterThan(0)
    expect(screen.queryByRole("button", { name: /上移规则集|下移规则集/ })).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "编辑规则集 inline" }))
    expect(screen.getByText("复杂 inline 规则内容请在高级 JSON 中维护。")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "取消" }))
    await user.click(screen.getByRole("button", { name: "复制规则集 inline" }))
    expect(screen.getAllByRole("button", { name: "编辑规则集 inline" })).toHaveLength(2)
  })

  it("adds an inline rule set from Empty and deletes only after confirmation", async () => {
    const user = userEvent.setup()
    renderApp(<EditorHarness initial={{}} />)
    expect(screen.getByText("暂无路由规则集")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "新增规则集" })[1])
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("Tag"), { target: { value: "inline-new" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(screen.getByRole("button", { name: "编辑规则集 inline-new" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "更多规则集 inline-new" }))
    await user.click(await screen.findByRole("menuitem", { name: "复制" }))
    expect(screen.getAllByRole("button", { name: "编辑规则集 inline-new" })).toHaveLength(2)
    await user.click(screen.getAllByRole("button", { name: "删除规则集 inline-new" })[0])
    await user.click(screen.getByRole("button", { name: "取消" }))
    expect(screen.getAllByRole("button", { name: "编辑规则集 inline-new" })).toHaveLength(2)
    await user.click(screen.getAllByRole("button", { name: "删除规则集 inline-new" })[0])
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(screen.getAllByRole("button", { name: "编辑规则集 inline-new" })).toHaveLength(1)
  })
})
