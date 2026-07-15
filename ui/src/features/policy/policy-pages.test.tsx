import { cleanup, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { i18n } from "@/i18n"
import { sessionStore } from "@/lib/session"
import { preferencesStore } from "@/lib/storage"
import { renderApp } from "@/test/render"

const booleanMatches = { source_ip_is_private: true, rule_set_ip_cidr_match_source: true }
const config = {
  route: {
    final: "proxy",
    rules: [
      { type: "logical", mode: "and", rules: [], action: "reject" },
      { ...booleanMatches, domain_suffix: ["example.com"], action: "route", outbound: "proxy" },
    ],
    rule_set: [{ type: "inline", tag: "geo", rules: [] }],
  },
  dns: {
    final: "legacy",
    fakeip: { enabled: true, inet4_range: "198.18.0.0/15" },
    servers: [
      { tag: "legacy", address: "local", detour: "direct" },
      { type: "https", tag: "remote", server: "dns.example", server_port: 443 },
    ],
    rules: [
      { type: "logical", mode: "or", rules: [], action: "reject" },
      { ...booleanMatches, domain_suffix: ["example.com"], action: "route", server: "remote" },
    ],
  },
}

function stubConfig() {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(config))))
}

async function renderPolicy(path: "/policy/route" | "/policy/dns", heading: string) {
  stubConfig()
  renderApp(<App />, path)
  await screen.findByRole("heading", { name: heading })
}

async function expectPolicyTabs(visual: string, advanced: string) {
  expect(screen.getByRole("tab", { name: visual })).toBeInTheDocument()
  await userEvent.click(screen.getByRole("tab", { name: advanced }))
  expect(screen.getByRole("textbox", { name: /流量策略 JSON|Traffic policy JSON/ })).toBeInTheDocument()
  await userEvent.click(screen.getByRole("tab", { name: visual }))
}

function expectNoRawBooleanKeys() {
  expect(document.body).not.toHaveTextContent(/source_ip_is_private|rule_set_ip_cidr_match_source/)
}

async function expectMenuActions(trigger: string, actions: readonly string[]) {
  await userEvent.click(screen.getByRole("button", { name: trigger }))
  const menu = await screen.findByRole("menu")
  for (const action of actions) expect(within(menu).getByRole("menuitem", { name: action })).toBeInTheDocument()
  await userEvent.keyboard("{Escape}")
}

async function expectChineseRouteCards() {
  for (const title of ["全局路由设置", "路由规则", "路由规则集"]) expect(screen.getByText(title)).toBeInTheDocument()
  expect(screen.getByText("共 2 条规则")).toBeInTheDocument()
  expect(screen.getByText("共 1 个规则集")).toBeInTheDocument()
  expect(screen.getByText("源 IP 为私有地址")).toBeInTheDocument()
  expect(screen.getByText("规则集 IP 匹配源地址")).toBeInTheDocument()
  expectNoRawBooleanKeys()
  await expectMenuActions("更多规则 2", ["复制", "上移", "下移", "删除"])
}

async function expectChineseRouteDialogs() {
  await userEvent.click(screen.getByRole("button", { name: "编辑规则 1" }))
  for (const tab of ["基础与网络", "域名与地址", "端口与进程", "规则集与网络环境", "执行动作", "高级 JSON"]) {
    expect(screen.getByRole("tab", { name: tab })).toBeInTheDocument()
  }
  expect(screen.getByText("逻辑子规则可在当前表单中编辑，其他复杂字段可在高级 JSON 中维护。")).toBeInTheDocument()
  await userEvent.click(screen.getByRole("button", { name: "取消" }))
  await userEvent.click(screen.getByRole("button", { name: "新增规则" }))
  expect(screen.getByRole("alert")).toHaveTextContent("请补全当前规则类型和动作所需的值。")
  await userEvent.click(screen.getByRole("button", { name: "取消" }))
  await userEvent.click(screen.getByRole("button", { name: "编辑规则集 geo" }))
  expect(screen.getByText("复杂 inline 规则内容请在高级 JSON 中维护。")).toBeInTheDocument()
}

async function expectChineseDNSCards() {
  for (const title of ["DNS 全局设置", "旧式 FakeIP", "DNS 服务器", "DNS 规则"]) expect(screen.getByText(title)).toBeInTheDocument()
  expect(screen.getByText("共 2 台服务器")).toBeInTheDocument()
  expect(screen.getByText("共 2 条规则")).toBeInTheDocument()
  expect(screen.getByText(/标签 legacy/)).toBeInTheDocument()
  expect(screen.getByText(/前置出站 direct/)).toBeInTheDocument()
  expect(screen.getByText("源 IP 为私有地址")).toBeInTheDocument()
  expect(screen.getByText("规则集 IP 匹配源地址")).toBeInTheDocument()
  expectNoRawBooleanKeys()
  await expectMenuActions("更多 DNS 服务器 legacy", ["复制", "删除"])
}

async function expectChineseDNSDialogs() {
  await userEvent.click(screen.getByRole("button", { name: "编辑 DNS 服务器 legacy" }))
  for (const tab of ["基础", "拨号与解析", "TLS 与 HTTP", "类型专属", "高级 JSON"]) {
    expect(screen.getByRole("tab", { name: tab })).toBeInTheDocument()
  }
  await userEvent.click(screen.getByRole("button", { name: "取消" }))
  await userEvent.click(screen.getByRole("button", { name: "新增 DNS 服务器" }))
  expect(screen.getByRole("alert")).toHaveTextContent("请填写当前服务器类型所需的 Tag 和地址信息。")
  await userEvent.click(screen.getByRole("button", { name: "取消" }))
  await userEvent.click(screen.getByRole("button", { name: "编辑 DNS 规则 1" }))
  expect(screen.getByText("逻辑子规则请在高级 JSON 中维护。")).toBeInTheDocument()
  await userEvent.click(screen.getByRole("button", { name: "取消" }))
  await userEvent.click(screen.getByRole("button", { name: "删除 DNS 规则 1" }))
  expect(screen.getByRole("alertdialog", { name: "删除 DNS 规则 #1？" })).toHaveTextContent("此操作无法撤销。")
}

function expectEnglishRouteCards() {
  for (const label of ["Global route settings", "Route rules", "Route rule sets", "Add route rule", "Add rule set"]) {
    expect(screen.getByText(label)).toBeInTheDocument()
  }
  expect(screen.getByText("Source IP is private")).toBeInTheDocument()
  expect(screen.getByText("Match rule-set IP against source")).toBeInTheDocument()
  expectNoRawBooleanKeys()
}

function expectEnglishDNSCards() {
  for (const label of ["DNS settings", "Legacy FakeIP", "DNS servers", "DNS rules", "Add DNS server", "Add DNS rule"]) {
    expect(screen.getByText(label)).toBeInTheDocument()
  }
  expect(screen.getByText("Source IP is private")).toBeInTheDocument()
  expect(screen.getByText("Match rule-set IP against source")).toBeInTheDocument()
  expectNoRawBooleanKeys()
}

beforeEach(async () => {
  preferencesStore.set({ language: "zh", theme: "system" })
  await i18n.changeLanguage("zh")
})

afterEach(async () => {
  cleanup()
  vi.unstubAllGlobals()
  sessionStore.clear()
  await i18n.changeLanguage("zh")
})

describe("Chinese Route policy interactions", () => {
  it("resolves cards, boolean summaries, actions, and dialogs", async () => {
    await renderPolicy("/policy/route", "路由")
    await expectPolicyTabs("可视化配置", "高级 JSON")
    await expectChineseRouteCards()
    await expectChineseRouteDialogs()
  }, 20_000)
})

describe("Chinese DNS policy interactions", () => {
  it("resolves cards, boolean summaries, actions, validation, and guidance", async () => {
    await renderPolicy("/policy/dns", "DNS")
    await expectPolicyTabs("可视化配置", "高级 JSON")
    await expectChineseDNSCards()
    await expectChineseDNSDialogs()
  }, 20_000)
})

describe("English policy interactions", () => {
  it("resolves Route cards and validation without Chinese or raw keys", async () => {
    preferencesStore.set({ language: "en", theme: "system" })
    await i18n.changeLanguage("en")
    await renderPolicy("/policy/route", "Route")
    await expectPolicyTabs("Visual editor", "Advanced JSON")
    expectEnglishRouteCards()
    await userEvent.click(screen.getByRole("button", { name: "Add route rule" }))
    expect(screen.getByRole("alert")).toHaveTextContent("Required fields missing")
    expect(document.body).not.toHaveTextContent(/[一-龥]/)
  }, 20_000)

  it("resolves DNS cards and validation without Chinese or raw keys", async () => {
    preferencesStore.set({ language: "en", theme: "system" })
    await i18n.changeLanguage("en")
    await renderPolicy("/policy/dns", "DNS")
    await expectPolicyTabs("Visual editor", "Advanced JSON")
    expectEnglishDNSCards()
    await userEvent.click(screen.getByRole("button", { name: "Add DNS rule" }))
    expect(screen.getByRole("alert")).toHaveTextContent("Required fields missing")
    expect(document.body).not.toHaveTextContent(/[一-龥]/)
  }, 20_000)
})
