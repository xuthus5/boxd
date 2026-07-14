import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function authenticate() {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  return userEvent.setup()
}

describe("proxy and policy interactions", () => {
  it("adds, edits, and deletes inbound configuration", async () => {
    const user = authenticate(); installMockAPI(); renderApp(<App />, "/proxy/inbounds")
    await screen.findByText("mixed-in")
    await user.click(screen.getByRole("button", { name: "新增入站" }))
    await user.type(screen.getByLabelText("Tag"), "new-in")
    await user.type(screen.getByLabelText("类型"), "mixed")
    await user.type(screen.getByLabelText("地址"), "::")
    await user.type(screen.getByLabelText("端口"), "1081")
    await user.click(screen.getByRole("button", { name: "保存" }))
    await user.click(screen.getByRole("button", { name: "编辑" }))
    await user.click(screen.getByRole("button", { name: "取消" }))
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
  })

  it("installs defaults and saves route, DNS, and outbound configuration", async () => {
    const user = authenticate(); installMockAPI()
    const outbound = renderApp(<App />, "/proxy/outbounds")
    await user.click(await screen.findByRole("button", { name: "安装默认出站" }))
    outbound.unmount()
    const route = renderApp(<App />, "/policy/route")
    await user.click(await screen.findByRole("button", { name: "保存配置" }))
    await user.click(screen.getByRole("button", { name: "安装默认路由" }))
    route.unmount()
    renderApp(<App />, "/policy/dns")
    await user.click(await screen.findByRole("button", { name: "安装默认 DNS" }))
  })
})
