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
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 503 })))
    renderApp(<App />, "/observability/logs")
    expect(await screen.findAllByText("SSE request failed with status 503")).toHaveLength(2)
  })

  it("shows log source tabs inside the log page", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(sse({ level: "info", message: "kernel ready", timestamp: "2026-01-01T00:00:00Z" })))
    renderApp(<App />, "/observability/logs")

    expect(await screen.findByRole("tab", { name: "内核日志" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "应用日志" })).toBeInTheDocument()
    expect(await screen.findByText("kernel ready")).toBeInTheDocument()
    const timestamp = document.querySelector('time[datetime="2026-01-01T00:00:00Z"]')
    expect(timestamp).toBeInTheDocument()
    expect(timestamp).not.toHaveTextContent("—")
  })

  it("shows error logs without a timestamp", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(sse({ level: "error", message: "failed", timestamp: "" })))
    renderApp(<App />, "/observability/logs")
    expect(await screen.findByText("failed")).toBeInTheDocument()
    expect(screen.getByRole("columnheader", { name: "时间" })).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
  })

  it("preserves each tab filter while switching sources", async () => {
    sessionStore.set({ token: "token", expiresAt: "2099-01-01T00:00:00Z" })
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => Promise.resolve(sse({ level: "info", message: "ready" }))))
    const user = userEvent.setup()
    renderApp(<App />, "/observability/logs")
    const filter = (await screen.findAllByLabelText("搜索日志"))[0]
    await user.type(filter, "kernel")
    await user.click(screen.getByRole("tab", { name: "应用日志" }))
    await user.click(screen.getByRole("tab", { name: "内核日志" }))
    expect(filter).toHaveValue("kernel")
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

  it("maps Warn and Error and preserves each tab threshold", async () => {
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
    expect(within(await screen.findByRole("tabpanel")).getByRole("combobox", { name: "最低日志级别" })).toHaveTextContent("全部")
    await user.click(screen.getByRole("tab", { name: "内核日志" }))
    expect(within(await screen.findByRole("tabpanel")).getByRole("combobox", { name: "最低日志级别" })).toHaveTextContent("Error")
  })
})
