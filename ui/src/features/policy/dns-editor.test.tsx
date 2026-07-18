import { useState } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { DNSRuleDialog } from "@/features/policy/dns-rule-dialog"
import { DNSServerDialog } from "@/features/policy/dns-server-dialog"
import { DNSVisualEditor } from "@/features/policy/dns-visual-editor"
import type { JsonObject } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

function renderDNS(ui: React.ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}

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
  return <><DNSVisualEditor {...props} /><output aria-label="dns state">{JSON.stringify(object)}</output>
    <output aria-label="dns rule identity">{String(rules.length > 1 && rules[0] === rules[1])}</output></>
}

function state(): JsonObject {
  return JSON.parse(screen.getByLabelText("dns state").textContent ?? "{}") as JsonObject
}

async function choose(label: string, option: string) {
  const user = userEvent.setup()
  await user.click(screen.getByRole("combobox", { name: label }))
  await user.click(await screen.findByRole("option", { name: option }))
}

function expectEveryFieldGrouped() {
  const fields = [...document.querySelectorAll('[data-slot="field"]')]
  expect(fields.length).toBeGreaterThan(0)
  expect(fields.every((field) => field.closest('[data-slot="field-group"]'))).toBe(true)
}

describe("DNS globals and legacy FakeIP", () => {
  it("edits every global field while retaining unknown keys", async () => {
    renderApp(<EditorHarness initial={{ custom: { keep: true } }} />)
    fireEvent.change(screen.getByLabelText("最终 DNS 服务器"), { target: { value: "remote" } })
    await choose("域名策略", "prefer_ipv4")
    for (const label of ["禁用缓存", "禁用缓存过期", "独立缓存", "反向映射"]) {
      await userEvent.click(screen.getByRole("switch", { name: label }))
    }
    fireEvent.change(screen.getByLabelText("缓存容量"), { target: { value: "4096" } })
    fireEvent.change(screen.getByLabelText("客户端子网"), { target: { value: "192.0.2.0/24" } })
    expect(state()).toMatchObject({
      final: "remote", strategy: "prefer_ipv4", disable_cache: true, disable_expire: true,
      independent_cache: true, cache_capacity: 4096, client_subnet: "192.0.2.0/24",
      reverse_mapping: true, custom: { keep: true },
    })
    expect(state()).not.toHaveProperty("fakeip")
  })

  it("creates FakeIP only after editing and prunes known paths without deleting unknown siblings", async () => {
    renderApp(<EditorHarness initial={{}} />)
    expect(state()).not.toHaveProperty("fakeip")
    await userEvent.click(screen.getByRole("switch", { name: "启用旧式 FakeIP" }))
    fireEvent.change(screen.getByLabelText("FakeIP IPv4 范围"), { target: { value: "198.18.0.0/15" } })
    fireEvent.change(screen.getByLabelText("FakeIP IPv6 范围"), { target: { value: "fc00::/18" } })
    expect(state().fakeip).toEqual({ enabled: true, inet4_range: "198.18.0.0/15", inet6_range: "fc00::/18" })
    await userEvent.click(screen.getByRole("switch", { name: "启用旧式 FakeIP" }))
    fireEvent.change(screen.getByLabelText("FakeIP IPv4 范围"), { target: { value: "" } })
    fireEvent.change(screen.getByLabelText("FakeIP IPv6 范围"), { target: { value: "" } })
    expect(state()).not.toHaveProperty("fakeip")

    renderApp(<EditorHarness initial={{ fakeip: { enabled: true, inet4_range: "198.18.0.0/15", future: 1 } }} />)
    await userEvent.click(screen.getAllByRole("switch", { name: "启用旧式 FakeIP" })[1])
    fireEvent.change(screen.getAllByLabelText("FakeIP IPv4 范围")[1], { target: { value: "" } })
    expect(screen.getAllByLabelText("dns state")[1]).toHaveTextContent('"fakeip":{"future":1}')
  })
})

describe("DNS server dialog", () => {
  it("preserves untouched legacy JSON exactly and changes shape only after type selection", async () => {
    const legacy = { tag: "legacy", address: "https://dns.google/dns-query", address_resolver: "local", custom: { keep: true } }
    const onSave = vi.fn()
    renderApp(<DNSServerDialog open title="编辑 DNS 服务器" item={legacy} onOpenChange={vi.fn()} onSave={onSave} />)
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenLastCalledWith(legacy)
    await choose("服务器类型", "udp")
    expect(screen.queryByLabelText("旧式地址")).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("服务器地址"), { target: { value: "1.1.1.1" } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenLastCalledWith({ tag: "legacy", type: "udp", server: "1.1.1.1", custom: { keep: true } })
  })

  it.each([
    ["local", "优先 Go 解析器"], ["hosts", "Hosts 路径"], ["udp", "服务器地址"],
    ["tcp", "服务器地址"], ["tls", "启用 TLS"], ["quic", "启用 TLS"],
    ["https", "HTTP Headers"], ["h3", "HTTP Headers"], ["dhcp", "网络接口"],
    ["fakeip", "FakeIP IPv4 范围"],
  ])("shows fields for %s and hides unsupported TLS/dialer fields", async (type, label) => {
    renderApp(<DNSServerDialog open title="新增 DNS 服务器" item={{ type, tag: "dns" }} onOpenChange={vi.fn()} onSave={vi.fn()} />)
    const tab = label === "启用 TLS" || label === "HTTP Headers" ? "TLS 与 HTTP"
      : ["优先 Go 解析器", "Hosts 路径", "网络接口", "FakeIP IPv4 范围"].includes(label) ? "类型专属" : "基础"
    if (tab !== "基础") await userEvent.click(screen.getByRole("tab", { name: tab }))
    const booleanLabel = ["优先 Go 解析器", "启用 TLS"].includes(label)
    expect(booleanLabel ? screen.getByRole("switch", { name: label }) : screen.getByLabelText(label)).toBeInTheDocument()
    if (["hosts", "fakeip"].includes(type)) expect(screen.queryByLabelText("前置出站")).not.toBeInTheDocument()
    if (!["tls", "quic", "https", "h3"].includes(type)) expect(screen.queryByLabelText("启用 TLS")).not.toBeInTheDocument()
  })

  it("gates required values, HTTPS headers, and routing mark union with visible invalid feedback", async () => {
    const onSave = vi.fn()
    renderApp(<DNSServerDialog open title="新增 DNS 服务器" item={{ type: "https" }} onOpenChange={vi.fn()} onSave={onSave} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("Tag"), { target: { value: "remote" } })
    fireEvent.change(screen.getByLabelText("服务器地址"), { target: { value: "dns.example" } })
    await userEvent.click(screen.getByRole("tab", { name: "TLS 与 HTTP" }))
    fireEvent.change(screen.getByLabelText("HTTP Headers"), { target: { value: "{" } })
    expect(screen.getByText("请输入有效的 JSON 结构。")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await userEvent.click(screen.getByRole("tab", { name: "基础" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await userEvent.click(screen.getByRole("tab", { name: "TLS 与 HTTP" }))
    fireEvent.change(screen.getByLabelText("HTTP Headers"), { target: { value: '{"X-Test":"1"}' } })
    await userEvent.click(screen.getByRole("tab", { name: "拨号与解析" }))
    fireEvent.change(screen.getByLabelText("路由标记"), { target: { value: "4294967296" } })
    expect(screen.getByLabelText("路由标记")).toHaveAttribute("aria-invalid", "true")
    expect(screen.getByText("请输入有效的 DNS 数值或名称。")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("路由标记"), { target: { value: "0x20" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })

  it.each([
    [{ tag: "legacy" }, "旧式地址", "local"],
    [{ type: "udp", tag: "remote" }, "服务器地址", "1.1.1.1"],
    [{ type: "fakeip", tag: "fake" }, "FakeIP IPv4 范围", "198.18.0.0/15"],
  ] as const)("requires type-specific server values for %j", async (item, label, value) => {
    renderApp(<DNSServerDialog open title="新增 DNS 服务器" item={item} onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    expect(screen.getByText("请填写当前服务器类型所需的 Tag 和地址信息。")).toBeInTheDocument()
    if (["网络接口", "FakeIP IPv4 范围"].includes(label)) {
      await userEvent.click(screen.getByRole("tab", { name: "类型专属" }))
    }
    fireEvent.change(screen.getByLabelText(label), { target: { value } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })

  it("keeps the DHCP interface optional while saving it when provided", async () => {
    const onSave = vi.fn()
    renderApp(<DNSServerDialog open title="新增 DNS 服务器" item={{ type: "dhcp", tag: "lan" }}
      onOpenChange={vi.fn()} onSave={onSave} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    await userEvent.click(screen.getByRole("tab", { name: "类型专属" }))
    expect(screen.getByLabelText("网络接口")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("网络接口"), { target: { value: "eth0" } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ type: "dhcp", tag: "lan", interface: "eth0" })
  })

  it("keeps unknown types in Advanced JSON", async () => {
    const item = { type: "future", tag: "custom", payload: { enabled: true } }
    const onSave = vi.fn()
    renderApp(<DNSServerDialog open title="编辑 DNS 服务器" item={item} onOpenChange={vi.fn()} onSave={onSave} />)
    await userEvent.click(screen.getByRole("combobox", { name: "服务器类型" }))
    expect(await screen.findByRole("option", { name: "future" })).toBeInTheDocument()
    await userEvent.keyboard("{Escape}")
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expect(screen.getByRole("textbox", { name: "编辑 DNS 服务器 JSON" })).toHaveTextContent('"future"')
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(item)
  })
})

describe("DNS server cards", () => {
  it("summarizes tag, type, endpoint, port, detour, and strategy", () => {
    renderApp(<EditorHarness initial={{ servers: [{
      type: "https", tag: "remote", server: "dns.example", server_port: 443,
      detour: "direct", strategy: "prefer_ipv4",
    }] }} />)
    expect(screen.getByText("https", { selector: '[data-slot="badge"]' })).toBeInTheDocument()
    expect(screen.getByText(/dns\.example:443 · 标签 remote · 前置出站 direct · 策略 prefer_ipv4/)).toBeInTheDocument()
  })

  it("adds, summarizes, copies deeply, and deletes after confirmation", async () => {
    const user = userEvent.setup()
    renderApp(<EditorHarness initial={{}} />)
    expect(screen.getByText("暂无 DNS 服务器")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "新增 DNS 服务器" })[0])
    fireEvent.change(screen.getByLabelText("Tag"), { target: { value: "google" } })
    fireEvent.change(screen.getByLabelText("旧式地址"), { target: { value: "https://dns.google/dns-query" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(screen.getAllByText(/https:\/\/dns.google\/dns-query/).length).toBeGreaterThan(0)
    await user.click(screen.getByRole("button", { name: "复制 DNS 服务器 google" }))
    expect(screen.getAllByRole("button", { name: "编辑 DNS 服务器 google" })).toHaveLength(2)
    await user.click(screen.getAllByRole("button", { name: "删除 DNS 服务器 google" })[0])
    expect(state().servers).toHaveLength(2)
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(state().servers).toHaveLength(1)
  })
})

describe("DNS rule dialog and cards", () => {
  it("edits approved matches, route target, logical JSON, and special unions", async () => {
    const onSave = vi.fn()
    renderDNS(<DNSRuleDialog open title="新增 DNS 规则" item={{ action: "route" }} serverTags={["local", "remote"]} onOpenChange={vi.fn()} onSave={onSave} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    expect(screen.getByText("请补全逻辑规则或当前动作的必填值。")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("查询类型"), { target: { value: "A!" } })
    expect(screen.getByLabelText("查询类型")).toHaveAttribute("aria-invalid", "true")
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await userEvent.click(screen.getByRole("tab", { name: "基础与网络" }))
    fireEvent.change(screen.getByLabelText("查询类型"), { target: { value: "A, 28" } })
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    await choose("目标 DNS 服务器", "remote")
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ action: "route", query_type: ["A", 28], server: "remote" })
    await choose("执行动作", "predefined")
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    fireEvent.change(screen.getByLabelText("响应码"), { target: { value: "4096" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("响应码"), { target: { value: "REFUSED" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  }, 15_000)

  it("supports four actions, logical mode/invert, and Advanced JSON child rules", async () => {
    renderDNS(<DNSRuleDialog open title="编辑 DNS 规则" item={{ type: "logical", mode: "or", rules: [], action: "reject" }} serverTags={[]} onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByText("逻辑子规则请在高级 JSON 中维护。")).toBeInTheDocument()
    await choose("逻辑模式", "and")
    await userEvent.click(screen.getByRole("switch", { name: "反向匹配" }))
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    for (const action of ["route", "route-options", "reject", "predefined"]) {
      await userEvent.click(screen.getByRole("combobox", { name: "执行动作" }))
      expect(await screen.findByRole("option", { name: action })).toBeInTheDocument()
      await userEvent.keyboard("{Escape}")
    }
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    const editor = screen.getByRole("textbox", { name: "编辑 DNS 规则 JSON" })
    await userEvent.click(editor)
    await userEvent.keyboard("{Control>}a{/Control}")
    await userEvent.paste("[")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })

  it("copies deeply, moves with boundaries, and deletes after confirmation", async () => {
    const user = userEvent.setup()
    renderApp(<EditorHarness initial={{ servers: [{ tag: "dns", address: "local" }], rules: [
      { domain_suffix: ["one.example"], server: "dns" }, { action: "reject", domain: ["two.example"] },
    ] }} />)
    expect(screen.getByText("one.example")).toBeInTheDocument()
    expect(screen.getByText("route · dns")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "上移 DNS 规则 1" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "下移 DNS 规则 2" })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "复制 DNS 规则 1" }))
    expect(screen.getByLabelText("dns rule identity")).toHaveTextContent("false")
    await user.click(screen.getByRole("button", { name: "下移 DNS 规则 1" }))
    expect((state().rules as JsonObject[])[1]).toMatchObject({ domain_suffix: ["one.example"] })
    await user.click(screen.getByRole("button", { name: "删除 DNS 规则 1" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(state().rules).toHaveLength(2)
  })
})

describe("DNS page integration", () => {
  it("renders the visual editor and PUTs the complete config with unknown data", async () => {
    const config = { dns: { final: "old", custom: { keep: true } }, route: { final: "proxy" }, log: { level: "info" } }
    const fetchMock = vi.fn((_input: string | URL | Request, init?: RequestInit) => Promise.resolve(
      new Response(JSON.stringify(init?.method === "PUT" ? { status: "ok", data: null, error: null, meta: {} } : config)),
    ))
    vi.stubGlobal("fetch", fetchMock)
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    renderApp(<App />, "/policy/dns")
    fireEvent.change(await screen.findByLabelText("最终 DNS 服务器"), { target: { value: "new" } })
    await userEvent.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/config/", expect.objectContaining({
      method: "PUT",
      body: JSON.stringify({ ...config, dns: { final: "new", custom: { keep: true } } }),
    })))
  })

  it("keeps every direct Dialog Field inside a FieldGroup", async () => {
    renderApp(<DNSServerDialog open title="编辑 DNS 服务器" item={{ type: "https", tag: "dns", server: "example.com" }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expectEveryFieldGrouped()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expectEveryFieldGrouped()
    cleanup()

    renderDNS(<DNSRuleDialog open title="编辑 DNS 规则" item={{ action: "reject" }} serverTags={[]}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expectEveryFieldGrouped()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expectEveryFieldGrouped()
  })
})