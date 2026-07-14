import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function setup(route: string) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  installMockAPI()
  renderApp(<App />, route)
  return userEvent.setup()
}

describe("node and subscription interactions", () => {
  it("tests, syncs, deletes, and imports nodes", async () => {
    const user = setup("/nodes")
    await screen.findByText("hk-01")
    await user.click(screen.getByRole("button", { name: "测速" }))
    await user.click(screen.getByRole("button", { name: "同步到配置" }))
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    await user.click(screen.getByRole("button", { name: "导入节点" }))
    await user.type(screen.getByLabelText("节点链接"), "vless://node")
    await user.click(screen.getByRole("button", { name: "解析" }))
    expect(await screen.findByText("new-node")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "保存节点" }))
  })

  it("creates, edits, refreshes, and deletes subscriptions", async () => {
    const user = setup("/subscriptions")
    await screen.findByText("主订阅")
    await user.click(screen.getByRole("button", { name: "刷新全部" }))
    await user.click(screen.getByRole("button", { name: "编辑" }))
    await user.click(screen.getByRole("button", { name: "保存" }))
    await user.click(screen.getByRole("button", { name: "刷新" }))
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    await user.click(screen.getByRole("button", { name: "新增订阅" }))
    await user.type(screen.getByLabelText("名称"), "备用")
    await user.type(screen.getByLabelText("URL"), "https://example.com/backup")
    await user.click(screen.getByRole("button", { name: "保存" }))
  })
})
