import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { DNSRuleDialog } from "@/features/policy/dns-rule-dialog"
import { DNSServerDialog } from "@/features/policy/dns-server-dialog"
import type { JsonObject } from "@/features/policy/policy-form-model"
import { renderApp } from "@/test/render"

function renderDNS(ui: React.ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}

afterEach(() => vi.restoreAllMocks())

const dialerLabels = [
  "前置出站", "绑定接口", "IPv4 绑定地址", "IPv6 绑定地址", "绑定地址时忽略端口",
  "保护路径", "路由标记", "复用地址", "网络命名空间", "连接超时", "TCP Fast Open",
  "TCP MultiPath", "禁用 TCP Keep Alive", "TCP Keep Alive", "TCP Keep Alive 间隔", "UDP 分片",
  "域名解析服务器", "域名解析策略", "禁用解析缓存", "解析重写 TTL", "解析客户端子网",
  "网络策略", "网络类型", "回退网络类型", "回退延迟",
] as const

const tlsLabels = [
  "启用 TLS", "禁用 SNI", "TLS 服务器名称", "跳过证书验证", "TLS ALPN", "CA 证书内容", "CA 证书路径",
] as const

const ruleTabs = {
  "基础与网络": ["规则类型", "入站", "IP 版本", "查询类型", "网络", "认证用户", "协议", "反向匹配"],
  "域名与地址": ["域名", "域名后缀", "域名关键字", "域名正则", "源 IP CIDR", "源 IP 为私有地址", "目标 IP CIDR", "目标 IP 为私有地址"],
  "端口与环境": [
    "源端口", "源端口范围", "目标端口", "目标端口范围", "进程名", "进程路径", "进程路径正则",
    "应用包名", "用户", "用户 ID", "出站", "Clash 模式", "规则集", "规则集 IP 匹配源地址",
    "网络类型", "计费网络", "受限网络", "Wi-Fi SSID", "Wi-Fi BSSID",
  ],
} as const

function tabPanel(name: string): HTMLElement {
  const tab = screen.getByRole("tab", { name })
  const panel = document.getElementById(tab.getAttribute("aria-controls") ?? "")
  expect(panel).not.toBeNull()
  return panel!
}

function expectLabels(panel: HTMLElement, labels: readonly string[]) {
  for (const label of labels) {
    expect(panel.querySelector(`[aria-label="${label}"]`), `${label} should be in selected panel`).not.toBeNull()
  }
}

function serverItem(type: string): JsonObject {
  if (type === "legacy") return { tag: "dns", address: "local" }
  if (["udp", "tcp", "tls", "quic", "https", "h3"].includes(type)) {
    return { type, tag: "dns", server: "dns.example" }
  }
  if (type === "fakeip") return { type, tag: "dns", inet4_range: "198.18.0.0/15" }
  return { type, tag: "dns" }
}

describe("DNS rule field matrix", () => {
  it("places every approved match field in its explicit Tab", () => {
    renderDNS(<DNSRuleDialog open title="编辑 DNS 规则" item={{ action: "reject" }} serverTags={[]}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    for (const [tab, labels] of Object.entries(ruleTabs)) expectLabels(tabPanel(tab), labels)
  })
})

describe("DNS server field matrix", () => {
  it.each([
    ["legacy", "基础", ["服务器类型", "Tag", "旧式地址"]],
    ["legacy", "拨号与解析", ["旧式地址解析器", "旧式地址策略", "旧式地址回退延迟", "域名策略", "前置出站", "客户端子网"]],
    ["local", "拨号与解析", dialerLabels],
    ["local", "类型专属", ["优先 Go 解析器"]],
    ["dhcp", "拨号与解析", dialerLabels],
    ["dhcp", "类型专属", ["优先 Go 解析器", "网络接口"]],
    ["udp", "基础", ["服务器地址", "服务器端口"]],
    ["udp", "拨号与解析", dialerLabels],
    ["tcp", "基础", ["服务器地址", "服务器端口"]],
    ["tcp", "拨号与解析", dialerLabels],
    ["tls", "基础", ["服务器地址", "服务器端口"]],
    ["tls", "拨号与解析", dialerLabels],
    ["tls", "TLS 与 HTTP", tlsLabels],
    ["quic", "基础", ["服务器地址", "服务器端口"]],
    ["quic", "拨号与解析", dialerLabels],
    ["quic", "TLS 与 HTTP", tlsLabels],
    ["https", "基础", ["服务器地址", "服务器端口"]],
    ["https", "拨号与解析", dialerLabels],
    ["https", "TLS 与 HTTP", [...tlsLabels, "HTTP 路径", "HTTP Method", "HTTP Headers"]],
    ["h3", "基础", ["服务器地址", "服务器端口"]],
    ["h3", "拨号与解析", dialerLabels],
    ["h3", "TLS 与 HTTP", [...tlsLabels, "HTTP 路径", "HTTP Method", "HTTP Headers"]],
    ["hosts", "类型专属", ["Hosts 路径", "预定义 Hosts"]],
    ["fakeip", "类型专属", ["FakeIP IPv4 范围", "FakeIP IPv6 范围"]],
  ] as const)("renders %s approved fields in %s", (type, tab, labels) => {
    renderApp(<DNSServerDialog open title="编辑 DNS 服务器" item={serverItem(type)}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    expectLabels(tabPanel(tab), labels)
  })

  it("keeps an unknown current type and its Advanced JSON", async () => {
    renderApp(<DNSServerDialog open title="编辑 DNS 服务器" item={{ type: "future", tag: "dns", payload: true }}
      onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await userEvent.click(screen.getByRole("combobox", { name: "服务器类型" }))
    expect(await screen.findByRole("option", { name: "future" })).toBeInTheDocument()
    await userEvent.keyboard("{Escape}")
    expect(tabPanel("高级 JSON").querySelector('[aria-label="编辑 DNS 服务器 JSON"]')).not.toBeNull()
  })
})

async function replaceJSON(label: string, value: string) {
  const editor = screen.getByRole("textbox", { name: label })
  await userEvent.click(editor)
  await userEvent.keyboard("{Control>}a{/Control}")
  await userEvent.paste(value)
}

describe("DNS Advanced JSON resilience", () => {
  it.each([
    ["server", <DNSServerDialog open title="编辑 DNS 服务器" item={{ type: "https", tag: "dns", server: "dns.example" }} onOpenChange={vi.fn()} onSave={vi.fn()} />, "编辑 DNS 服务器 JSON", "基础", "Tag", '{"type":"https","tag":"dns","server":"new.example"}'],
    ["rule", <DNSRuleDialog open title="编辑 DNS 规则" item={{ action: "reject", domain: ["example.com"] }} serverTags={[]} onOpenChange={vi.fn()} onSave={vi.fn()} />, "编辑 DNS 规则 JSON", "域名与地址", "域名", '{"action":"reject","domain":["new.example"]}'],
  ] as const)("keeps %s Tabs mounted while JSON is temporarily invalid", async (_kind, dialog, label, otherTab, structuredLabel, validJSON) => {
    renderDNS(dialog)
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await replaceJSON(label, "[")
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    expect(screen.getByRole("tab", { name: otherTab })).toBeInTheDocument()
    await userEvent.click(screen.getByRole("tab", { name: otherTab }))
    expect(screen.getByLabelText(structuredLabel)).toBeInTheDocument()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expect(screen.getByRole("textbox", { name: label })).toHaveTextContent("[")
    await replaceJSON(label, validJSON)
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  }, 15_000)

  it("replaces invalid raw JSON when a structured field is edited", async () => {
    renderApp(<DNSServerDialog open title="编辑 DNS 服务器"
      item={{ type: "https", tag: "dns", server: "dns.example" }} onOpenChange={vi.fn()} onSave={vi.fn()} />)
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    await replaceJSON("编辑 DNS 服务器 JSON", "[")
    await userEvent.click(screen.getByRole("tab", { name: "基础" }))
    fireEvent.change(screen.getByLabelText("Tag"), { target: { value: "recovered" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    await userEvent.click(screen.getByRole("tab", { name: "高级 JSON" }))
    expect(screen.getByRole("textbox", { name: "编辑 DNS 服务器 JSON" })).toHaveTextContent('"recovered"')
  })
})