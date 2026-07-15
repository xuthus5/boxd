import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { installMockAPI } from "@/test/mock-api"
import { renderApp } from "@/test/render"

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function setup(route: string) {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  const fetchMock = installMockAPI()
  renderApp(<App />, route)
  return { fetchMock, user: userEvent.setup() }
}

describe("node and subscription interactions", () => {
  it("keeps node management out of the node page", async () => {
    const { user } = setup("/nodes")
    const all = await screen.findByRole("region", { name: "所有节点" })
    const card = within(all).getByRole("article", { name: "hk-01" })
    await user.click(within(card).getByRole("button", { name: "测速" }))
    expect(screen.queryByRole("button", { name: "同步到配置" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "导入节点" })).not.toBeInTheDocument()
    expect(within(card).queryByRole("button", { name: "删除" })).not.toBeInTheDocument()
  })

  it("imports nodes from subscriptions and synchronizes automatically", async () => {
    const { fetchMock, user } = setup("/subscriptions")
    await screen.findByText("主订阅")
    await user.click(screen.getByRole("button", { name: "导入节点" }))
    await user.type(screen.getByLabelText("节点链接"), "vless://node")
    await user.click(screen.getByRole("button", { name: "解析" }))
    expect(await screen.findByText("new-node")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "保存节点" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/nodes/sync-config", expect.objectContaining({ method: "POST" })))
  })

  it("deletes imported nodes and synchronizes automatically", async () => {
    const { fetchMock, user } = setup("/subscriptions")
    const card = await screen.findByRole("article", { name: "hk-01" })
    await user.click(within(card).getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/hk-01",
      expect.objectContaining({ method: "DELETE" }),
    ))
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/sync-config",
      expect.objectContaining({ method: "POST" }),
    )
  })
})

describe("subscription synchronization workflows", () => {
  it("creates, edits, refreshes, and deletes subscriptions", async () => {
    const { fetchMock, user } = setup("/subscriptions")
    await screen.findByText("主订阅")
    const subscription = screen.getByRole("article", { name: "主订阅" })
    await user.click(screen.getByRole("button", { name: "刷新全部" }))
    await user.click(within(subscription).getByRole("button", { name: "编辑" }))
    await user.click(screen.getByRole("button", { name: "保存" }))
    await user.click(within(subscription).getByRole("button", { name: "刷新" }))
    await user.click(within(subscription).getByRole("button", { name: "删除" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    await user.click(screen.getByRole("button", { name: "新增订阅" }))
    await user.type(screen.getByLabelText("名称"), "备用")
    await user.type(screen.getByLabelText("订阅 URL"), "https://example.com/backup")
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/nodes/sync-config", expect.objectContaining({ method: "POST" })))
  })

  it("reports partial refresh-all responses without synchronizing", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const path = typeof input === "string" ? input : input.toString()
      if (path === "/api/subscriptions/" || path === "/api/nodes/") {
        return Promise.resolve(new Response("[]"))
      }
      if (path === "/api/subscriptions/refresh-all") {
        return Promise.resolve(new Response(JSON.stringify({ status: "partial", data: null, error: null, meta: null })))
      }
      return Promise.resolve(new Response("{}"))
    })
    vi.stubGlobal("fetch", fetchMock)
    renderApp(<App />, "/subscriptions")

    await userEvent.setup().click(await screen.findByRole("button", { name: "刷新全部" }))
    expect(await screen.findByText("部分订阅刷新失败")).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith(
      "/api/nodes/sync-config",
      expect.objectContaining({ method: "POST" }),
    )
  })
})
