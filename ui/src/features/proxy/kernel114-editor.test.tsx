import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { InboundEditorDialog } from "@/features/proxy/inbound-editor-dialog"
import { OutboundEditorDialog } from "@/features/proxy/outbound-editor-dialog"
import { DNSRuleDialog } from "@/features/policy/dns-rule-dialog"
import { DNSServerDialog } from "@/features/policy/dns-server-dialog"
import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

function renderDialog(ui: React.ReactElement) {
  return renderApp(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>)
}

async function select(label: string, option: string) {
  await userEvent.click(screen.getByRole("combobox", { name: label }))
  await userEvent.click(await screen.findByRole("option", { name: option, exact: true }))
}

beforeEach(() => { installMockAPI() })

describe("sing-box 1.14 protocol editors", () => {
  it("saves a Snell v6 outbound with version-specific settings", async () => {
    const onSave = vi.fn()
    renderDialog(<OutboundEditorDialog title="Snell" item={{ type: "snell", tag: "snell", server: "example.org", server_port: 443 }} onClose={vi.fn()} onSave={onSave} />)
    await userEvent.click(screen.getByRole("tab", { name: "协议" }))
    fireEvent.change(screen.getByLabelText("版本"), { target: { value: "6" } })
    fireEvent.change(screen.getByLabelText("预共享密钥"), { target: { value: "test-key" } })
    await select("Snell v6 模式", "unshaped")
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ type: "snell", tag: "snell", server: "example.org", server_port: 443, version: 6, psk: "test-key", mode: "unshaped" })
  })

  it("configures Cloudflared without introducing unsupported listen fields", async () => {
    const onSave = vi.fn()
    renderDialog(<InboundEditorDialog title="Tunnel" item={{ type: "cloudflared", tag: "tunnel", token: "test-token" }} onClose={vi.fn()} onSave={onSave} />)
    expect(screen.queryByLabelText("监听端口")).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole("tab", { name: "协议" }))
    await select("隧道协议", "quic")
    fireEvent.change(screen.getByLabelText("控制连接拨号参数"), { target: { value: '{"detour":"proxy"}' } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ type: "cloudflared", tag: "tunnel", token: "test-token", protocol: "quic", control_dialer: { detour: "proxy" } })
  })

  it("saves TUN DNS and NAT options as current 1.14 fields", async () => {
    const onSave = vi.fn()
    renderDialog(<InboundEditorDialog title="TUN" item={{ type: "tun", tag: "tun", address: ["172.19.0.1/30"] }} onClose={vi.fn()} onSave={onSave} />)
    await userEvent.click(screen.getByRole("tab", { name: "TUN" }))
    await select("TUN DNS 模式", "hijack")
    fireEvent.change(screen.getByLabelText("TUN DNS 地址"), { target: { value: "172.19.0.2\nfdfe::2" } })
    await select("UDP NAT 映射方式", "address_dependent")
    fireEvent.change(screen.getByLabelText("UDP NAT 会话上限"), { target: { value: "8192" } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ dns_mode: "hijack", dns_address: ["172.19.0.2", "fdfe::2"], udp_mapping: "address_dependent", udp_nat_max: 8192 }))
  })

  it("blocks malformed certificate provider JSON after changing tabs", async () => {
    const onSave = vi.fn()
    renderDialog(<InboundEditorDialog title="TLS" item={{ type: "http", tag: "http", listen_port: 8080, tls: { enabled: true } }} onClose={vi.fn()} onSave={onSave} />)
    await userEvent.click(screen.getByRole("tab", { name: "TLS / Reality" }))
    fireEvent.change(screen.getByLabelText("证书提供者"), { target: { value: "{" } })
    await userEvent.click(screen.getByRole("tab", { name: "基础" }))
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    await userEvent.click(screen.getByRole("tab", { name: "TLS / Reality" }))
    fireEvent.change(screen.getByLabelText("证书提供者"), { target: { value: '"managed"' } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ tls: { enabled: true, certificate_provider: "managed" } }))
  })
})

describe("sing-box 1.14 DNS and route editors", () => {
  it("saves evaluate actions and response matching with correct JSON value types", async () => {
    const onSave = vi.fn()
    renderDialog(<DNSRuleDialog open title="DNS" item={{ action: "evaluate", server: "remote" }} serverTags={["remote"]} onOpenChange={vi.fn()} onSave={onSave} />)
    await userEvent.click(screen.getByRole("tab", { name: "执行动作" }))
    fireEvent.change(screen.getByLabelText("DNS 评估结果标签"), { target: { value: "answer" } })
    fireEvent.change(screen.getByLabelText("DNS 查询超时"), { target: { value: "3s" } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenLastCalledWith({ action: "evaluate", server: "remote", tag: "answer", timeout: "3s" })
    await select("执行动作", "respond")
    await userEvent.click(screen.getByRole("tab", { name: "域名与地址" }))
    fireEvent.change(screen.getByLabelText("匹配 DNS 响应"), { target: { value: '"answer"' } })
    fireEvent.change(screen.getByLabelText("DNS 响应代码"), { target: { value: "noerror" } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenLastCalledWith({ action: "respond", match_response: "answer", response_rcode: "NOERROR" })
  })

  it.each(["openvpn", "openconnect", "tailscale"])("saves %s DNS with its endpoint", async (type) => {
    const onSave = vi.fn()
    renderDialog(<DNSServerDialog open title="VPN DNS" item={{ type, tag: "vpn-dns" }} onOpenChange={vi.fn()} onSave={onSave} />)
    expect(screen.getByRole("button", { name: "保存" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("端点标签"), { target: { value: "vpn" } })
    await userEvent.click(screen.getByRole("tab", { name: "类型专属" }))
    await userEvent.click(screen.getByRole("switch", { name: "使用搜索域" }))
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ type, tag: "vpn-dns", endpoint: "vpn", accept_search_domain: true })
  })

  it("offers ICMP matching and new source identity matches", async () => {
    const onSave = vi.fn()
    renderDialog(<RouteRuleDialog open title="Route" item={{ action: "reject" }} onOpenChange={vi.fn()} onSave={onSave} />)
    await userEvent.click(screen.getByLabelText("网络 icmp"))
    await userEvent.click(screen.getByRole("tab", { name: "规则集与网络环境" }))
    fireEvent.change(screen.getByLabelText("来源主机名"), { target: { value: "client.example" } })
    await userEvent.click(screen.getByRole("button", { name: "保存" }))
    expect(onSave).toHaveBeenCalledWith({ action: "reject", network: ["icmp"], source_hostname: ["client.example"] }, expect.anything())
  })
})
