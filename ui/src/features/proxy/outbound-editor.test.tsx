import type { ReactElement } from "react"
import { fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { describe, expect, it, vi } from "vitest"

import { OutboundEditorDialog } from "@/features/proxy/outbound-editor-dialog"
import { renderApp } from "@/test/render"

function renderEditor(ui: ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}

describe("outbound editor", () => {
  it("offers supported types and excludes removed outbounds", async () => {
    const user = userEvent.setup()
    renderEditor(<OutboundEditorDialog title="新增出站" item={{}} onClose={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await user.click(screen.getByRole("combobox", { name: "类型" }))
    for (const type of ["direct", "block", "selector", "urltest", "vless", "hysteria2", "ssh", "tor"]) {
      expect(await screen.findByRole("option", { name: type })).toBeInTheDocument()
    }
    for (const type of ["dns", "wireguard", "shadowsocksr", "mixed"]) {
      expect(screen.queryByRole("option", { name: type })).not.toBeInTheDocument()
    }
  })

  it("edits VLESS TLS, uTLS, Reality, transport, and multiplex fields", async () => {
    const user = userEvent.setup()
    const onSave = vi.fn()
    renderEditor(<OutboundEditorDialog title="编辑" item={{ tag: "vless-out", type: "vless", server: "old.example.com", server_port: 443, custom: "keep" }} onClose={vi.fn()} onSave={onSave} />)

    expect(screen.getByRole("dialog")).toHaveClass("sm:max-w-5xl")
    expect(screen.getByLabelText("服务器地址")).toHaveValue("old.example.com")
    await user.click(screen.getByRole("tab", { name: "协议" }))
    fireEvent.change(screen.getByLabelText("UUID"), { target: { value: "uuid" } })
    await user.click(screen.getByRole("tab", { name: "TLS / uTLS / Reality" }))
    await user.click(screen.getByRole("switch", { name: "启用 TLS" }))
    fireEvent.change(screen.getByLabelText("服务器名称"), { target: { value: "example.com" } })
    await user.click(screen.getByRole("switch", { name: "启用 uTLS" }))
    fireEvent.change(screen.getByLabelText("uTLS 指纹"), { target: { value: "chrome" } })
    await user.click(screen.getByRole("switch", { name: "启用 Reality" }))
    fireEvent.change(screen.getByLabelText("Reality 公钥"), { target: { value: "public-key" } })
    await user.click(screen.getByRole("tab", { name: "传输与复用" }))
    await user.click(screen.getByRole("combobox", { name: "传输类型" }))
    await user.click(await screen.findByRole("option", { name: "ws" }))
    fireEvent.change(screen.getByLabelText("传输路径"), { target: { value: "/ws" } })
    await user.click(screen.getByRole("switch", { name: "启用多路复用" }))
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({
      custom: "keep", uuid: "uuid",
      tls: expect.objectContaining({ enabled: true, server_name: "example.com", utls: { enabled: true, fingerprint: "chrome" }, reality: { enabled: true, public_key: "public-key" } }),
      transport: { type: "ws", path: "/ws" }, multiplex: { enabled: true },
    }))
  })

  it("edits selector and URLTest groups with a subscription ownership warning", async () => {
    const user = userEvent.setup()
    const onSave = vi.fn()
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ outbounds: [{ tag: "a" }, { tag: "b" }, { tag: "c" }] }))))
    renderEditor(<OutboundEditorDialog title="编辑" item={{ tag: "group", type: "selector", outbounds: ["a", "b"] }} onClose={vi.fn()} onSave={onSave} />)
    await user.click(screen.getByRole("tab", { name: "分组" }))
    expect(screen.getByText("订阅自动生成的分组请在订阅页面管理")).toBeInTheDocument()
    expect(screen.getByText("a")).toBeInTheDocument()
    await user.click(await screen.findByRole("combobox", { name: "分组成员" }))
    await user.click(await screen.findByRole("option", { name: "c" }))
    await user.click(screen.getByRole("tab", { name: "基础" }))
    await user.click(screen.getByRole("combobox", { name: "类型" }))
    await user.click(await screen.findByRole("option", { name: "urltest" }))
    await user.click(screen.getByRole("tab", { name: "分组" }))
    fireEvent.change(screen.getByLabelText("测试地址"), { target: { value: "https://example.com" } })
    fireEvent.change(screen.getByLabelText("测试间隔"), { target: { value: "3m" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ type: "urltest", outbounds: ["a", "b", "c"], url: "https://example.com", interval: "3m" }))
  })

  it("shows SSH fields without TLS or transport tabs", async () => {
    const user = userEvent.setup()
    renderEditor(<OutboundEditorDialog title="编辑" item={{ tag: "ssh-out", type: "ssh", server: "host", server_port: 22 }} onClose={vi.fn()} onSave={vi.fn()} />)
    await user.click(screen.getByRole("tab", { name: "协议" }))
    expect(screen.getByLabelText("SSH 用户")).toBeInTheDocument()
    expect(screen.queryByRole("tab", { name: "TLS / uTLS / Reality" })).not.toBeInTheDocument()
    expect(screen.queryByRole("tab", { name: "传输与复用" })).not.toBeInTheDocument()
  })

  it("keeps invalid structured JSON blocking save across tabs", async () => {
    const user = userEvent.setup()
    renderEditor(<OutboundEditorDialog title="编辑" item={{ tag: "http-out", type: "http", server: "host", server_port: 8080 }} onClose={vi.fn()} onSave={vi.fn()} />)
    await user.click(screen.getByRole("tab", { name: "协议" }))
    fireEvent.change(screen.getByLabelText("请求 Headers"), { target: { value: "invalid" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "基础" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })

  it("hides TLS nested fields until enabled and exposes version presets", async () => {
    const user = userEvent.setup()
    renderEditor(<OutboundEditorDialog title="编辑" item={{ tag: "vless-out", type: "vless", server: "host", server_port: 443 }} onClose={vi.fn()} onSave={vi.fn()} />)
    await user.click(screen.getByRole("tab", { name: "TLS / uTLS / Reality" }))
    expect(screen.queryByLabelText("服务器名称")).not.toBeInTheDocument()
    expect(screen.queryByLabelText("最低 TLS 版本")).not.toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "启用 TLS" }))
    expect(screen.getByLabelText("服务器名称")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "最低 TLS 版本" }))
    for (const version of ["1.0", "1.1", "1.2", "1.3"]) {
      expect(await screen.findByRole("option", { name: version })).toBeInTheDocument()
    }
    expect(screen.queryByLabelText("uTLS 指纹")).not.toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "启用 uTLS" }))
    expect(screen.getByLabelText("uTLS 指纹")).toBeInTheDocument()
  })

  it("requires outbound base fields before saving", async () => {
    const user = userEvent.setup()
    renderEditor(<OutboundEditorDialog title="新增" item={{ type: "vless" }} onClose={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    expect(screen.getByText("缺少必填字段")).toBeInTheDocument()
    expect(screen.getByText("请补全类型、Tag、服务器地址与端口等基础必填项。")).toBeInTheDocument()
    expect(screen.getByText("请填写 Tag。")).toBeInTheDocument()
    expect(screen.getByText("请填写服务器地址。")).toBeInTheDocument()
    expect(screen.getByText("请填写 1-65535 的端口。")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("Tag"), { target: { value: "hk" } })
    fireEvent.change(screen.getByLabelText("服务器地址"), { target: { value: "hk.example.com" } })
    fireEvent.change(screen.getByLabelText("服务器端口"), { target: { value: "443" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
    expect(screen.queryByText("缺少必填字段")).not.toBeInTheDocument()
    await user.clear(screen.getByLabelText("服务器端口"))
    fireEvent.change(screen.getByLabelText("服务器端口"), { target: { value: "0" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    expect(screen.getByText("缺少必填字段")).toBeInTheDocument()
  })
})
