import type { ReactElement } from "react"
import { fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { describe, expect, it, vi } from "vitest"

import { InboundEditorDialog } from "@/features/proxy/inbound-editor-dialog"
import { renderApp } from "@/test/render"

function renderEditor(ui: ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}

describe("inbound editor", () => {
  it("offers supported inbound types and mixed fields", async () => {
    const user = userEvent.setup()
    renderEditor(<InboundEditorDialog title="新增入站" item={{}} onClose={vi.fn()} onSave={vi.fn()} />)

    await user.click(screen.getByRole("combobox", { name: "类型" }))
    for (const type of ["mixed", "socks", "http", "tun", "vless", "hysteria2", "tuic"]) {
      expect(await screen.findByRole("option", { name: type })).toBeInTheDocument()
    }
    await user.click(screen.getByRole("option", { name: "mixed" }))
    await user.click(screen.getByRole("tab", { name: "协议" }))
    expect(screen.getByRole("button", { name: "添加用户" })).toBeInTheDocument()
    expect(screen.getByRole("switch", { name: "设置系统代理" })).toBeInTheDocument()
  })

  it("edits VLESS TLS, Reality, transport, and multiplex fields", async () => {
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderEditor(<InboundEditorDialog title="编辑 vless" item={{ tag: "in", type: "vless", custom: "keep" }} onClose={vi.fn()} onSave={onSave} />)

    await user.click(screen.getByRole("tab", { name: "协议" }))
    await user.click(screen.getByRole("button", { name: "添加用户" }))
    fireEvent.change(screen.getByLabelText("认证用户 1 名称"), { target: { value: "alice" } })
    fireEvent.change(screen.getByLabelText("认证用户 1 UUID"), { target: { value: "uuid" } })
    fireEvent.change(screen.getByLabelText("认证用户 1 Flow"), { target: { value: "xtls-rprx-vision" } })
    await user.click(screen.getByRole("tab", { name: "TLS / Reality" }))
    await user.click(screen.getByRole("switch", { name: "启用 TLS" }))
    fireEvent.change(screen.getByLabelText("服务器名称"), { target: { value: "example.com" } })
    await user.click(screen.getByRole("switch", { name: "启用 Reality" }))
    fireEvent.change(screen.getByLabelText("Reality 握手服务器"), { target: { value: "origin.example.com" } })
    await user.click(screen.getByRole("tab", { name: "传输与复用" }))
    await user.click(screen.getByRole("combobox", { name: "传输类型" }))
    await user.click(await screen.findByRole("option", { name: "ws" }))
    fireEvent.change(screen.getByLabelText("传输路径"), { target: { value: "/ws" } })
    await user.click(screen.getByRole("switch", { name: "启用多路复用" }))
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({
      custom: "keep",
      tls: expect.objectContaining({ enabled: true, server_name: "example.com", reality: expect.objectContaining({ enabled: true }) }),
      transport: expect.objectContaining({ type: "ws", path: "/ws" }),
      multiplex: expect.objectContaining({ enabled: true }),
      users: [{ name: "alice", uuid: "uuid", flow: "xtls-rprx-vision" }],
    }))
  })

  it("shows TUN fields without listen address and clears fields when type changes", async () => {
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderEditor(<InboundEditorDialog title="编辑" item={{ type: "mixed", listen: "::", listen_port: 1080, users: [{ username: "old" }] }} onClose={vi.fn()} onSave={onSave} />)

    await user.click(screen.getByRole("combobox", { name: "类型" }))
    await user.click(await screen.findByRole("option", { name: "tun" }))
    expect(screen.queryByLabelText("监听地址")).not.toBeInTheDocument()
    expect(screen.getByLabelText("接口名称")).toBeInTheDocument()
    await user.type(screen.getByLabelText("接口名称"), "tun0")
    await user.click(screen.getByRole("button", { name: "保存" }))

    await waitFor(() => expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ type: "tun", interface_name: "tun0" })))
    const saved = onSave.mock.calls[0][0]
    expect(saved).not.toHaveProperty("listen")
    expect(saved).not.toHaveProperty("listen_port")
    expect(saved).not.toHaveProperty("users")
  })

  it("shows transport-specific fields and saves HTTP Upgrade host as a string", async () => {
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderEditor(<InboundEditorDialog title="编辑" item={{ type: "vless", transport: { type: "ws", path: "/ws" } }} onClose={vi.fn()} onSave={onSave} />)

    expect(screen.getByRole("dialog")).toHaveClass("sm:max-w-5xl")
    await user.click(screen.getByRole("tab", { name: "传输与复用" }))
    expect(screen.getByLabelText("传输路径")).toBeInTheDocument()
    expect(screen.getByLabelText("传输 Headers")).toBeInTheDocument()
    expect(screen.queryByLabelText("传输 Host")).not.toBeInTheDocument()
    expect(screen.queryByLabelText("服务名称")).not.toBeInTheDocument()

    await user.click(screen.getByRole("combobox", { name: "传输类型" }))
    await user.click(await screen.findByRole("option", { name: "httpupgrade" }))
    fireEvent.change(screen.getByLabelText("传输 Host"), { target: { value: "upgrade.example.com" } })
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ transport: expect.objectContaining({ type: "httpupgrade", host: "upgrade.example.com" }) }))
  })


  it("supports listen address presets and manual input", async () => {
    const onSave = vi.fn()
    const user = userEvent.setup()
    renderEditor(<InboundEditorDialog title="编辑" item={{ type: "mixed", listen: "::", listen_port: 1080 }} onClose={vi.fn()} onSave={onSave} />)

    await user.click(screen.getByRole("combobox", { name: "监听地址" }))
    await user.click(await screen.findByRole("option", { name: "0.0.0.0（IPv4 全接口）" }))
    await user.click(screen.getByRole("combobox", { name: "监听地址" }))
    await user.click(await screen.findByRole("option", { name: "手动输入" }))
    fireEvent.change(screen.getByLabelText("自定义监听地址"), { target: { value: "192.168.1.10" } })
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ type: "mixed", listen: "192.168.1.10", listen_port: 1080 }))
  })

  it("requires an inbound type before saving", () => {
    renderEditor(<InboundEditorDialog title="新增" item={{}} onClose={vi.fn()} onSave={vi.fn()} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
  })

  it("blocks saving while a structured field contains invalid JSON", async () => {
    const user = userEvent.setup()
    renderEditor(<InboundEditorDialog title="编辑" item={{ type: "vless", transport: { type: "ws" } }} onClose={vi.fn()} onSave={vi.fn()} />)
    await user.click(screen.getByRole("tab", { name: "传输与复用" }))
    fireEvent.change(screen.getByLabelText("传输 Headers"), { target: { value: "invalid" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "基础" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await user.click(screen.getByRole("tab", { name: "传输与复用" }))
    fireEvent.change(screen.getByLabelText("传输 Headers"), { target: { value: "{}" } })
    expect(screen.getByRole("button", { name: "保存" })).toBeEnabled()
  })
})
