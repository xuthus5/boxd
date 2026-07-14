import { screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function setup(route: string) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  installMockAPI()
  void route
  return userEvent.setup()
}

describe("observability interactions", () => {
  it("closes one or all connections", async () => {
    const user = setup("/observability/connections")
    renderApp(<App />, "/observability/connections")
    await screen.findByText("example.com:443")
    await user.click(screen.getByRole("button", { name: "关闭" }))
    await user.click(screen.getByRole("button", { name: "关闭全部连接" }))
    await user.click(screen.getByRole("button", { name: "确认关闭" }))
  })

  it("filters, pauses, clears, and switches log sources", async () => {
    const user = setup("/observability/logs")
    renderApp(<App />, "/observability/logs")
    const panel = await screen.findByRole("tabpanel")
    await within(panel).findByText("ready")
    await user.type(within(panel).getByLabelText("搜索日志"), "ready")
    await user.click(within(panel).getByRole("button", { name: "暂停" }))
    await user.click(within(panel).getByRole("button", { name: "清空显示" }))
    await user.click(screen.getByRole("tab", { name: "应用日志" }))
  })
})
