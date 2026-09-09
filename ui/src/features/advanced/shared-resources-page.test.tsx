import { fireEvent, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import type { SingBoxConfig } from "@/lib/api/types"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

const ok = { status: "ok", data: null, error: null }

function setup(path: string, config: SingBoxConfig = {}, failure?: "save" | "validate" | "rollback" | "load") {
  const fallback = installMockAPI()
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetch = vi.fn((input: string | URL | Request, init?: RequestInit) => {
    const url = String(input)
    const operation = url.includes("/api/config/validate") ? "validate" : init?.method === "PUT" ? "save" : "load"
    if (!url.startsWith("/api/config/") || url.includes("history")) return fallback(input)
    const error = { status: "error", data: null, error: { code: "config_invalid", message: "http_clients[0].version: invalid version" } }
    if (operation === failure) return Promise.resolve(new Response(JSON.stringify(error), { status: 400 }))
    if (operation === "save" && failure === "rollback") return Promise.resolve(new Response(JSON.stringify({ ...error, status: "rolled_back" })))
    return Promise.resolve(new Response(JSON.stringify(operation === "load" ? config : ok)))
  })
  vi.stubGlobal("fetch", fetch)
  return { fetch, user: userEvent.setup(), view: renderApp(<App />, path) }
}

async function select(label: string, value: string) {
  await userEvent.click(screen.getByRole("combobox", { name: label }))
  await userEvent.click(await screen.findByRole("option", { name: value, exact: true }))
}

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

describe("shared resource configuration pages", () => {
  it("adds a namespace, validates it, and saves the whole config without dropping siblings", async () => {
    const initial = { log: { level: "info" }, dns: { servers: [{ type: "local", tag: "local" }] } }
    const { fetch, user } = setup("/advanced/network-namespaces", initial)
    await user.click(await screen.findByRole("button", { name: "添加命名空间" }))
    const dialog = screen.getByRole("dialog")
    expect(within(dialog).getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(within(dialog).getByLabelText("标签"), { target: { value: "isolated" } })
    await select("类型", "unshare")
    fireEvent.change(screen.getByLabelText("命名空间进程 PID 文件"), { target: { value: "/run/boxd/ns.pid" } })
    await user.click(within(dialog).getByRole("button", { name: "保存" }))
    await user.click(screen.getByRole("button", { name: "校验配置" }))
    await waitFor(() => expect(fetch.mock.calls.some(([url]) => String(url).includes("/api/config/validate"))).toBe(true))
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    const request = fetch.mock.calls.find(([, init]) => init?.method === "PUT")
    expect(JSON.parse(String(request?.[1]?.body))).toEqual({ ...initial, network_namespaces: [{ type: "unshare", tag: "isolated", pid_file: "/run/boxd/ns.pid" }] })
  })

  it("edits HTTP/3 options and blocks malformed JSON across tabs", async () => {
    const { user, fetch } = setup("/advanced/http-clients", { http_clients: [{ tag: "downloads", version: 2 }], outbounds: [{ type: "direct", tag: "direct" }] })
    await user.click(await screen.findByRole("button", { name: "编辑 downloads" }))
    await select("HTTP 版本", "3")
    fireEvent.change(screen.getByLabelText("初始数据包大小"), { target: { value: "1350" } })
    fireEvent.change(screen.getByLabelText("HTTP TLS 参数"), { target: { value: "{" } })
    await user.click(within(screen.getByRole("dialog")).getByRole("tab", { name: "高级 JSON" }))
    expect(within(screen.getByRole("dialog")).getByRole("button", { name: "保存" })).toBeDisabled()
    await user.click(within(screen.getByRole("dialog")).getByRole("tab", { name: "可视化配置" }))
    fireEvent.change(screen.getByLabelText("HTTP TLS 参数"), { target: { value: '{"enabled":true,"server_name":"example.org"}' } })
    await select("前置出站", "direct")
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "保存" }))
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    const request = fetch.mock.calls.find(([, init]) => init?.method === "PUT")
    expect(JSON.parse(String(request?.[1]?.body)).http_clients[0]).toMatchObject({ version: 3, initial_packet_size: 1350, detour: "direct", tls: { enabled: true, server_name: "example.org" } })
  })

  it("adds certificate providers with structured challenge JSON and preserves it in the editor", async () => {
    const { user } = setup("/advanced/certificate-providers")
    await user.click(await screen.findByRole("button", { name: "添加证书提供者" }))
    fireEvent.change(screen.getByLabelText("标签"), { target: { value: "managed" } })
    fireEvent.change(screen.getByLabelText("证书域名"), { target: { value: "example.org" } })
    fireEvent.change(screen.getByLabelText("DNS-01 验证配置"), { target: { value: '{"provider":"cloudflare","api_token":"test-token"}' } })
    fireEvent.change(within(screen.getByRole("dialog")).getByLabelText("HTTP 客户端"), { target: { value: '"downloads"' } })
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "保存" }))
    await user.click(screen.getByRole("button", { name: "编辑 managed" }))
    expect(JSON.parse((screen.getByLabelText("DNS-01 验证配置") as HTMLTextAreaElement).value)).toEqual({ provider: "cloudflare", api_token: "test-token" })
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "取消" }))
    await user.click(screen.getByRole("button", { name: "删除 managed" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(screen.queryByRole("button", { name: "编辑 managed" })).not.toBeInTheDocument()
  })

  it.each(["save", "validate", "rollback"] as const)("reports %s failures", async (failure) => {
    const { user } = setup("/advanced/http-clients", { http_clients: [{ tag: "bad", version: 9 }] }, failure)
    await screen.findByRole("button", { name: "编辑 bad" })
    await user.click(screen.getByRole("button", { name: failure === "validate" ? "校验配置" : "保存配置" }))
    expect(await screen.findByText(/invalid version/)).toBeInTheDocument()
  })

  it("keeps malformed resource lists visible for correction and blocks save", async () => {
    setup("/advanced/network-namespaces", { network_namespaces: { invalid: true } })
    expect(await screen.findByRole("alert")).toHaveTextContent("此配置必须是对象数组")
    expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled()
  })

  it("opens a diagnostic deep link at the new section JSON", async () => {
    setup("/advanced/http-clients?path=http_clients%5B0%5D.version", { http_clients: [{ tag: "downloads", version: 2 }] })
    expect(await screen.findByRole("textbox", { name: "共享资源列表 JSON" })).toHaveTextContent('"version": 2')
  })

  it("reports load errors", async () => {
    setup("/advanced/http-clients", {}, "load")
    expect(await screen.findByRole("alert")).toBeInTheDocument()
  })
})
