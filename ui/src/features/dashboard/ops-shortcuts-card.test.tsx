import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { OpsShortcutsCard } from "@/features/dashboard/ops-shortcuts-card"
import { renderApp } from "@/test/render"

describe("OpsShortcutsCard", () => {
  it("renders deep links for common ops destinations", () => {
    renderApp(<OpsShortcutsCard />)
    expect(screen.getByText("运维入口")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "连接" })).toHaveAttribute("href", "/observability/connections")
    expect(screen.getByRole("link", { name: "日志" })).toHaveAttribute("href", "/observability/logs")
    expect(screen.getByRole("link", { name: "错误日志" })).toHaveAttribute("href", "/observability/logs?preset=errors")
    expect(screen.getByRole("link", { name: "拦截日志" })).toHaveAttribute("href", "/observability/logs?preset=reject")
    expect(screen.getByRole("link", { name: "失败订阅" })).toHaveAttribute("href", "/subscriptions?status=error")
    expect(screen.getByRole("link", { name: "不稳节点" })).toHaveAttribute("href", "/nodes?stability=unstable")
    expect(screen.getByRole("link", { name: "拦截规则" })).toHaveAttribute("href", "/policy/route?action=reject")
    expect(screen.getByRole("link", { name: "节点" })).toHaveAttribute("href", "/nodes")
    expect(screen.getByRole("link", { name: "出口" })).toHaveAttribute("href", "/proxy/outbounds")
    expect(screen.getByRole("link", { name: "路由" })).toHaveAttribute("href", "/policy/route")
    expect(screen.getByRole("link", { name: "DNS" })).toHaveAttribute("href", "/policy/dns")
  })
})
