import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

import App from "@/App"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

function sse(data: unknown) {
  const encoder = new TextEncoder()
  const events = Array.isArray(data) ? data : [data]
  return new Response(new ReadableStream({ start(controller) { controller.enqueue(encoder.encode(events.map((event) => `data: ${JSON.stringify(event)}\n\n`).join(""))); controller.close() } }))
}

afterEach(() => { vi.unstubAllGlobals(); sessionStore.clear() })

function setupLevelThreshold() {
  sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
  vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse([
    { level: "trace", message: "trace entry" },
    { level: "debug", message: "debug entry" },
    { level: "info", message: "info entry" },
    { level: "warn", message: "warn entry" },
    { level: "error", message: "error entry" },
  ]))))
  renderApp(<App />, "/observability/logs")
  return userEvent.setup()
}

describe("LogsPage", () => {
  it("shows stream connection errors", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(new Response(null, { status: 503 }))))
    renderApp(<App />, "/observability/logs")
    expect(await screen.findAllByText("SSE request failed with status 503")).toHaveLength(2)
    expect(await screen.findAllByText("unavailable")).toHaveLength(2)
    expect(screen.getAllByText("流已断开（unavailable）").length).toBeGreaterThan(0)
  })

  it("shows log source tabs inside the log page", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse({ level: "info", message: "kernel ready", timestamp: "2026-01-01T00:00:00Z" }))))
    renderApp(<App />, "/observability/logs")

    expect(await screen.findByRole("tab", { name: "内核日志" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "应用日志" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "内核日志" }).closest('[data-slot="tabs-list"]')).toHaveClass("overflow-x-auto")
    expect((await screen.findAllByText("kernel ready")).length).toBeGreaterThan(0)
    const timestamps = document.querySelectorAll('time[datetime="2026-01-01T00:00:00Z"]')
    expect(timestamps.length).toBeGreaterThan(0)
    expect(timestamps[0]).not.toHaveTextContent("—")
  })

  it("shows error logs without a timestamp", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse({ level: "error", message: "failed", timestamp: "" }))))
    renderApp(<App />, "/observability/logs")
    expect((await screen.findAllByText("failed")).length).toBeGreaterThan(0)
    expect(screen.getAllByRole("columnheader", { name: "时间" }).length).toBeGreaterThan(0)
    expect(screen.getAllByText("—").length).toBeGreaterThan(0)
  })

  it("shares URL filters across log source tabs", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse({ level: "info", message: "ready" }))))
    const user = userEvent.setup()
    renderApp(<App />, "/observability/logs")
    const filter = (await screen.findAllByLabelText("搜索日志"))[0]
    await user.type(filter, "kernel")
    await user.click(screen.getByRole("tab", { name: "应用日志" }))
    const appFilter = within(await screen.findByRole("tabpanel")).getByLabelText("搜索日志")
    expect(appFilter).toHaveValue("kernel")
    await user.click(screen.getByRole("tab", { name: "内核日志" }))
    expect(within(await screen.findByRole("tabpanel")).getByLabelText("搜索日志")).toHaveValue("kernel")
  })

  it("seeds filters from deep-link query params", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse([
      { level: "info", message: "dns query ok" },
      { level: "error", message: "connection failed" },
    ]))))
    renderApp(<App />, "/observability/logs?tab=application&preset=errors")
    expect(await screen.findByRole("tab", { name: "应用日志" })).toHaveAttribute("data-active")
    const panel = await screen.findByRole("tabpanel")
    expect(within(panel).getByLabelText("搜索日志")).toHaveValue("error")
    expect(within(panel).getByRole("combobox", { name: "最低日志级别" })).toHaveTextContent("Error")
    expect(within(panel).getByRole("button", { name: "错误" })).toHaveAttribute("aria-pressed", "true")
    expect(within(panel).queryByText("dns query ok")).not.toBeInTheDocument()
    expect(await within(panel).findByText("connection failed")).toBeInTheDocument()
  })
  it("deep-links log hosts to connection search", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse({
      level: "info",
      message: "outbound/vless[hk]: outbound connection to api.telegram.org:443",
      timestamp: "2026-01-01T00:00:00Z",
    }))))
    renderApp(<App />, "/observability/logs")

    expect((await screen.findAllByText(/outbound connection to api.telegram.org:443/)).length).toBeGreaterThan(0)
    const link = screen.getAllByRole("link", { name: /查看连接/ })[0]
    expect(link).toHaveAttribute("href", "/observability/connections?q=api.telegram.org")
  })



  it("clears log filters from empty-state action", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
      const raw = typeof input === "string" ? input : input instanceof URL ? input.pathname : new URL(input.url).pathname
      const path = raw.split("?")[0]
      if (path.endsWith("/api/stats/logs") || path.endsWith("/api/stats/app-logs")) {
        return Promise.resolve(sse({
          level: "info",
          message: "kernel ready",
          timestamp: "2026-01-01T00:00:00Z",
        }))
      }
      if (path.endsWith("/api/settings/preferences")) {
        return Promise.resolve(new Response(JSON.stringify({
          theme: "system", language: "zh", minimumLogLevel: "all",
        })))
      }
      if (path.endsWith("/api/settings/password")) {
        return Promise.resolve(new Response(JSON.stringify({ defaultPassword: false })))
      }
      return Promise.resolve(new Response(JSON.stringify({})))
    }))
    const user = userEvent.setup()
    renderApp(<App />, "/observability/logs?q=missing-token")

    const panel = await screen.findByRole("tabpanel")
    expect(await within(panel).findByText("无匹配日志")).toBeInTheDocument()
    // Toolbar clear + empty-state clear can both exist; use the empty-state (last) action.
    const clearButtons = within(panel).getAllByRole("button", { name: "清除过滤" })
    await user.click(clearButtons[clearButtons.length - 1])
    expect(await within(panel).findByText("kernel ready")).toBeInTheDocument()
  })

})


describe("LogsPage level threshold", () => {
  it("describes the threshold and maps All and Debug", async () => {
    const user = setupLevelThreshold()
    const panel = await screen.findByRole("tabpanel")
    const select = within(panel).getByRole("combobox", { name: "最低日志级别" })
    expect(await within(panel).findByText("trace entry")).toBeInTheDocument()
    expect(select).toHaveAccessibleDescription("选择最低日志级别后，将显示该级别及以上日志。")
    await user.click(select)
    await user.click(await screen.findByRole("option", { name: "Debug" }))
    expect(within(panel).queryByText("trace entry")).not.toBeInTheDocument()
    for (const message of ["debug entry", "info entry", "warn entry", "error entry"]) {
      expect(within(panel).getAllByText(message).length).toBeGreaterThan(0)
    }
  })

  it("maps the Info threshold", async () => {
    const user = setupLevelThreshold()
    const panel = await screen.findByRole("tabpanel")
    await within(panel).findByText("debug entry")
    await user.click(within(panel).getByRole("combobox", { name: "最低日志级别" }))
    await user.click(await screen.findByRole("option", { name: "Info" }))
    expect(within(panel).queryByText("debug entry")).not.toBeInTheDocument()
    expect(within(panel).getAllByText("info entry").length).toBeGreaterThan(0)
    expect(within(panel).getAllByText("warn entry").length).toBeGreaterThan(0)
    expect(within(panel).getAllByText("error entry").length).toBeGreaterThan(0)
  })

  it("maps Warn and Error and shares threshold via URL", async () => {
    const user = setupLevelThreshold()
    const panel = await screen.findByRole("tabpanel")
    await within(panel).findByText("debug entry")
    await user.click(within(panel).getByRole("combobox", { name: "最低日志级别" }))
    await user.click(await screen.findByRole("option", { name: "Warn" }))
    expect(within(panel).queryByText("info entry")).not.toBeInTheDocument()
    expect(within(panel).getAllByText("warn entry").length).toBeGreaterThan(0)
    await waitFor(() => expect(screen.queryByRole("option", { name: "Warn" })).not.toBeInTheDocument())
    await user.click(within(panel).getByRole("combobox", { name: "最低日志级别" }))
    await user.click(await screen.findByRole("option", { name: "Error" }))
    expect(within(panel).queryByText("warn entry")).not.toBeInTheDocument()
    expect(within(panel).getAllByText("error entry").length).toBeGreaterThan(0)
    await user.click(screen.getByRole("tab", { name: "应用日志" }))
    expect(within(await screen.findByRole("tabpanel")).getByRole("combobox", { name: "最低日志级别" })).toHaveTextContent("Error")
    await user.click(screen.getByRole("tab", { name: "内核日志" }))
    expect(within(await screen.findByRole("tabpanel")).getByRole("combobox", { name: "最低日志级别" })).toHaveTextContent("Error")
  })
  it("filters logs from level summary chips", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse([
      { level: "info", message: "info entry" },
      { level: "error", message: "error entry" },
    ]))))
    const user = userEvent.setup()
    renderApp(<App />, "/observability/logs")
    const panel = await screen.findByRole("tabpanel")
    expect(await within(panel).findByText(/级别分布/)).toBeInTheDocument()
    expect(await within(panel).findByText("info entry")).toBeInTheDocument()
    expect(within(panel).getByText("error entry")).toBeInTheDocument()
    await user.click(within(panel).getAllByRole("button", { name: /error/i })[0])
    expect(within(panel).queryByText("info entry")).not.toBeInTheDocument()
    expect(within(panel).getByText("error entry")).toBeInTheDocument()
    expect(within(panel).getAllByRole("button", { name: /error/i })[0]).toHaveAttribute("aria-pressed", "true")
  })

  it("seeds exact level filter from deep-link query params", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse([
      { level: "info", message: "info entry" },
      { level: "error", message: "error entry" },
    ]))))
    renderApp(<App />, "/observability/logs?level=error")
    const panel = await screen.findByRole("tabpanel")
    expect(await within(panel).findByText("error entry")).toBeInTheDocument()
    expect(within(panel).queryByText("info entry")).not.toBeInTheDocument()
    expect(within(panel).getAllByRole("button", { name: /error/i })[0]).toHaveAttribute("aria-pressed", "true")
  })

})
