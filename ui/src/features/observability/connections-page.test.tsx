import { screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { CONNECTION_COLUMN_STORAGE_KEY } from "@/features/observability/connection-columns"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStore.clear()
  localStorage.removeItem(CONNECTION_COLUMN_STORAGE_KEY)
})

function mockConnectionsFetch() {
  const encoder = new TextEncoder()
  const body = new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(`data: ${JSON.stringify({
        active_connections: 2,
        list: [
          {
            id: 1,
            target: "example.com:443",
            outbound: "proxy",
            rule: "geosite-google",
            network: "tcp",
            source: "10.0.0.2:51234",
            inbound: "mixed-in",
            protocol: "tls",
            process: "/usr/bin/curl",
            upload: 10,
            download: 20,
            start: new Date(Date.now() - 1000).toISOString(),
          },
          {
            id: 2,
            target: "cdn.example.net:443",
            outbound: "direct",
            rule: "geoip-cn",
            network: "udp",
            protocol: "quic",
            upload: 1,
            download: 2,
            start: new Date(Date.now() - 2000).toISOString(),
          },
        ],
      })}\n\n`))
      controller.close()
    },
  })
  vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
    const raw = typeof input === "string"
      ? input
      : input instanceof URL
        ? input.pathname
        : new URL(input.url).pathname
    const path = raw.split("?")[0]
    if (path === "/api/stats/connections") {
      return Promise.resolve(new Response(body, { headers: { "Content-Type": "text/event-stream" } }))
    }
    if (path === "/api/settings/preferences") {
      return Promise.resolve(new Response(JSON.stringify({
        theme: "system", language: "zh", minimumLogLevel: "all",
      })))
    }
    if (path === "/api/settings/password") {
      return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
    }
    return Promise.resolve(new Response(JSON.stringify({})))
  }))
}

describe("ConnectionsPage", () => {
  it("shows live connections with rule and supports filtering", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    const user = userEvent.setup()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.getByText("显示 2 条")).toBeInTheDocument()
    expect(screen.getByText(/2 个出口/)).toBeInTheDocument()
    expect(screen.getByText("geosite-google")).toBeInTheDocument()
    expect(screen.getByText("1s")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "关闭全部连接" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "按出口" })).toBeInTheDocument()
    await user.click(screen.getByRole("tab", { name: "按出口" }))
    expect(await screen.findByText("proxy")).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: "关闭该组" }).length).toBeGreaterThan(0)

    await user.click(screen.getByRole("tab", { name: "连接列表" }))
    await user.type(screen.getByLabelText("搜索连接"), "direct")
    expect(screen.queryByText("example.com:443")).not.toBeInTheDocument()
    expect(screen.getByText("cdn.example.net:443")).toBeInTheDocument()
  })

  it("filters connections by network facet and clears filters", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    const user = userEvent.setup()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.getByText("cdn.example.net:443")).toBeInTheDocument()

    await user.click(screen.getByRole("combobox", { name: "网络" }))
    await user.click(await screen.findByRole("option", { name: "tcp (1)" }))
    expect(screen.getByText("example.com:443")).toBeInTheDocument()
    expect(screen.queryByText("cdn.example.net:443")).not.toBeInTheDocument()
    expect(screen.getByText("显示 1 条")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "关闭筛选结果" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "清除筛选" }))
    expect(screen.getByText("example.com:443")).toBeInTheDocument()
    expect(screen.getByText("cdn.example.net:443")).toBeInTheDocument()
    expect(screen.getByText("显示 2 条")).toBeInTheDocument()
  })

  it("pauses the stream and toggles optional columns", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    const user = userEvent.setup()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.queryByText("10.0.0.2:51234")).not.toBeInTheDocument()
    expect(screen.queryByRole("columnheader", { name: "来源" })).not.toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "暂停" }))
    expect(screen.getByText("已暂停")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "继续" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "显示列" }))
    const menu = await screen.findByRole("menu")
    await user.click(within(menu).getByRole("menuitemcheckbox", { name: "来源" }))
    expect(await screen.findByRole("columnheader", { name: "来源" })).toBeInTheDocument()
    expect(screen.getByText("10.0.0.2:51234")).toBeInTheDocument()
    expect(localStorage.getItem(CONNECTION_COLUMN_STORAGE_KEY)).toContain("source")
  })

  it("copies connection target to clipboard", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    const copySpy = vi.spyOn(await import("@/features/proxy/copy-tag-button"), "copyText").mockResolvedValue()
    const user = userEvent.setup()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "复制目标: example.com:443" }))
    expect(copySpy).toHaveBeenCalledWith("example.com:443")
    expect(await screen.findByText("目标已复制")).toBeInTheDocument()
    copySpy.mockRestore()
  })

  it("applies deep-link facets from the URL", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    renderApp(<App />, "/observability/connections?network=tcp&outbound=proxy")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.queryByText("cdn.example.net:443")).not.toBeInTheDocument()
    expect(screen.getByText("显示 1 条")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "清除筛选" })).toBeInTheDocument()
  })


  it("deep-links connection targets to log search", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    const link = screen.getByRole("link", { name: "查看日志: example.com:443" })
    expect(link).toHaveAttribute("href", "/observability/logs?q=example.com")
    expect(screen.getByRole("link", { name: "查看日志: cdn.example.net:443" })).toHaveAttribute(
      "href",
      "/observability/logs?q=cdn.example.net",
    )
  })


  it("filters connections by process facet from the URL", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    renderApp(<App />, "/observability/connections?process=/usr/bin/curl")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.queryByText("cdn.example.net:443")).not.toBeInTheDocument()
    expect(screen.getByText("显示 1 条")).toBeInTheDocument()
  })


  it("shows process groups for live connections", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    const user = userEvent.setup()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    await user.click(screen.getByRole("tab", { name: "按进程" }))
    expect(await screen.findByText("/usr/bin/curl")).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: "关闭该组" }).length).toBeGreaterThan(0)
  })


  it("renders clickable outbound facet links", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    mockConnectionsFetch()
    renderApp(<App />, "/observability/connections")

    expect(await screen.findByText("example.com:443")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "出站: proxy" })).toHaveAttribute(
      "href",
      "/observability/connections?outbound=proxy",
    )
    expect(screen.getByRole("link", { name: "网络: tcp" })).toHaveAttribute(
      "href",
      "/observability/connections?network=tcp",
    )
  })

})
