import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { I18nextProvider } from "react-i18next"
import { describe, expect, it, vi } from "vitest"

import { ConfigPreflightPanel } from "@/features/config/config-preflight-panel"
import type { ConfigPreflightIssue } from "@/features/config/config-preflight"
import { configPreflightMessageKey } from "@/features/config/config-preflight-message"
import { i18n } from "@/i18n"

function renderPanel(issues: ConfigPreflightIssue[], onSelectPath = vi.fn()) {
  return {
    onSelectPath,
    ...render(
      <I18nextProvider i18n={i18n}>
        <ConfigPreflightPanel issues={issues} onSelectPath={onSelectPath} />
      </I18nextProvider>,
    ),
  }
}

describe("ConfigPreflightPanel", () => {
  it("renders the clean state", () => {
    renderPanel([])
    expect(screen.getByTestId("config-preflight")).toHaveTextContent("配置预检")
    expect(screen.getByText(/未发现明显/)).toBeInTheDocument()
  })

  it("renders errors, warnings, related definitions, and jump actions", async () => {
    const onSelectPath = vi.fn()
    const user = userEvent.setup()
    renderPanel([
      { severity: "error", code: "missing_outbound", path: "route.final", reference: "proxy" },
      { severity: "warning", code: "empty_group", path: "outbounds[0].outbounds", reference: "selector", relatedPath: "outbounds[0].tag" },
    ], onSelectPath)
    expect(screen.getByText("错误 1")).toBeInTheDocument()
    expect(screen.getByText("警告 1")).toBeInTheDocument()
    expect(screen.getByText(/引用的出站/)).toBeInTheDocument()
    expect(screen.getByText(/已在 outbounds\[0\]\.tag 定义/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "route.final" }))
    expect(onSelectPath).toHaveBeenCalledWith("route.final")
  })

  it("maps every supported diagnostic code", () => {
    const codes: ConfigPreflightIssue["code"][] = [
      "duplicate_tag",
      "missing_tag",
      "missing_outbound",
      "missing_dns_server",
      "missing_rule_set",
      "empty_group",
      "invalid_group_default",
      "outbound_dependency_cycle",
      "dns_dependency_cycle",
      "invalid_dns_default",
      "multiple_fakeip_dns_servers",
      "missing_domain_resolver",
    ]
    for (const code of codes) expect(configPreflightMessageKey(code)).toMatch(/^advanced\.preflight/)
  })

  it("explains DNS startup topology errors", () => {
    renderPanel([
      { severity: "error", code: "dns_dependency_cycle", path: "dns.servers[1].domain_resolver", reference: "dns-a" },
      { severity: "error", code: "invalid_dns_default", path: "dns.final", reference: "fake" },
      { severity: "error", code: "multiple_fakeip_dns_servers", path: "dns.servers[2].type", reference: "fake-extra" },
      { severity: "error", code: "missing_domain_resolver", path: "dns.servers[3].server", reference: "remote" },
    ])
    expect(screen.getByText(/DNS\/出站启动闭环/)).toBeInTheDocument()
    expect(screen.getByText(/不能作为显式或隐式默认 DNS/)).toBeInTheDocument()
    expect(screen.getByText(/仅支持一个 FakeIP DNS/)).toBeInTheDocument()
    expect(screen.getByText(/未设置 domain_resolver 或 detour/)).toBeInTheDocument()
  })

  it("limits long issue lists and keeps warning-only state non-destructive", () => {
    const issues = Array.from({ length: 14 }, (_, index) => ({
      severity: "warning" as const,
      code: "empty_group" as const,
      path: `outbounds[${index}].outbounds`,
      reference: "selector",
    }))
    renderPanel(issues)
    expect(screen.getByText("错误 0")).toBeInTheDocument()
    expect(screen.getByText("警告 14")).toBeInTheDocument()
    expect(screen.getByText("还有 2 项未展开。")).toBeInTheDocument()
    expect(screen.getAllByText("警告")).toHaveLength(12)
  })
})
