import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import { ConfigDiagnosticsCard } from "@/features/dashboard/config-diagnostics-card"
import { renderApp } from "@/test/render"

afterEach(() => vi.unstubAllGlobals())

const healthy = {
  status: "healthy" as const,
  checked_at: "2026-01-01T00:00:00Z",
  summary: { errors: 0, warnings: 0 },
  counts: { inbounds: 1, outbounds: 2, endpoints: 0, route_rules: 3, rule_sets: 1, dns_servers: 2, dns_rules: 1 },
  features: { tun: true, clash_api: true, cache_file: true, fakeip: true, selector: true, urltest: false, wireguard: false, remote_rule_set: true },
  issues: [],
}

function renderCard() {
  return renderApp(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ConfigDiagnosticsCard />
    </QueryClientProvider>,
    "/dashboard",
  )
}

describe("ConfigDiagnosticsCard", () => {
  it("renders topology, enabled features, and refreshes", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(healthy)))
    vi.stubGlobal("fetch", fetchMock)
    const user = userEvent.setup()
    renderCard()

    expect(await screen.findByText("配置正常")).toBeInTheDocument()
    expect(screen.getByText("TUN")).toBeInTheDocument()
    expect(screen.getByText("远程规则集")).toBeInTheDocument()
    expect(screen.getByText("3", { selector: "p" })).toBeInTheDocument()
    expect(screen.getByText("未发现结构或 sing-box 语义问题。")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "刷新配置诊断" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  })

  it("links actionable issues to the matching editor", async () => {
    const report = {
      ...healthy,
      status: "error" as const,
      summary: { errors: 8, warnings: 3 },
      issues: [
        { code: "unknown_outbound_reference", severity: "error" as const, path: "route.rules[0].outbound", value: "missing" },
        { code: "tls_insecure", severity: "warning" as const, path: "outbounds[0].tls.insecure", value: "node" },
        { code: "legacy_dns_server", severity: "warning" as const, path: "dns.servers[0]", value: "local" },
        { code: "missing_tag", severity: "error" as const, path: "route.rule_set[0].tag" },
        { code: "empty_group", severity: "error" as const, path: "outbounds[1].outbounds", value: "urltest" },
        { code: "invalid_group_default", severity: "error" as const, path: "outbounds[2].default", value: "node" },
        { code: "outbound_dependency_cycle", severity: "error" as const, path: "outbounds[3].detour", value: "cycle" },
        { code: "dns_dependency_cycle", severity: "error" as const, path: "dns.servers[1].domain_resolver", value: "dns-a" },
        { code: "invalid_dns_default", severity: "error" as const, path: "dns.final", value: "fake" },
        { code: "multiple_fakeip_dns_servers", severity: "error" as const, path: "dns.servers[3].type", value: "fake-extra" },
        { code: "future_code", severity: "warning" as const, path: "config", detail: "unknown detail" },
      ],
    }
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(report))))
    renderCard()

    expect(await screen.findByText("配置有错误")).toBeInTheDocument()
    expect(screen.getByText("missing")).toBeInTheDocument()
    expect(screen.getByText("规则集缺少 Tag")).toBeInTheDocument()
    expect(screen.getByText("空代理组")).toBeInTheDocument()
    expect(screen.getByText("Selector 默认项无效")).toBeInTheDocument()
    expect(screen.getByText("出站依赖循环")).toBeInTheDocument()
    expect(screen.getByText("DNS 解析依赖循环")).toBeInTheDocument()
    expect(screen.getByText("默认 DNS 不能使用 FakeIP")).toBeInTheDocument()
    expect(screen.getByText("存在多个 FakeIP DNS")).toBeInTheDocument()
    expect(screen.getByText("unknown detail")).toBeInTheDocument()
    const editorLinks = screen.getAllByRole("link", { name: "打开编辑器" })
    expect(editorLinks[0]).toHaveAttribute("href", "/policy/route?path=route.rules%5B0%5D.outbound")
    expect(editorLinks[1]).toHaveAttribute("href", "/proxy/outbounds?path=outbounds%5B0%5D.tls.insecure")
    expect(screen.getByRole("link", { name: "查看迁移指南" })).toHaveAttribute(
      "href",
      expect.stringContaining("migrate-to-new-dns-server-formats"),
    )
  })

  it("shows loading and query error states", async () => {
    let resolveRequest: ((response: Response) => void) | undefined
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>((resolve) => { resolveRequest = resolve })))
    renderCard()
    expect(screen.getByTestId("config-diagnostics-card").querySelector('[data-slot="skeleton"]')).toBeInTheDocument()
    resolveRequest?.(new Response(JSON.stringify(healthy)))
    await screen.findByText("配置正常")

    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "error", data: null, error: { code: "internal_error", message: "diagnostics failed" }, meta: null,
    }), { status: 500 })))
    renderCard()
    expect(await screen.findByText("diagnostics failed")).toBeInTheDocument()
  })
})
