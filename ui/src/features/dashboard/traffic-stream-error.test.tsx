import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { TrafficChart } from "@/features/dashboard/traffic-chart"
import { RecentLogs } from "@/features/dashboard/recent-logs"
import { AuthProvider } from "@/features/auth/auth-context"
import { PreferencesProvider } from "@/features/preferences/preferences-provider"
import { renderApp } from "@/test/render"
import * as copy from "@/features/proxy/copy-tag-button"

describe("dashboard stream densify", () => {
  it("shows densified traffic stream error with copy", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    spy.mockClear()
    const user = userEvent.setup()
    renderApp(
      <TrafficChart
        points={[]}
        streamError="failed to fetch traffic"
        streamStatus="error"
        streamPath="/api/stats/traffic"
      />,
    )
    expect(screen.getByText("failed to fetch traffic")).toBeInTheDocument()
    const block = document.querySelector('[data-slot="health-stream-error"]')
    expect(block).toHaveAttribute("data-error-code", "network")
    expect(block?.querySelector("a")).toBeNull()
    await user.click(screen.getByRole("button", { name: "复制流错误" }))
    expect(spy).toHaveBeenCalled()
    expect(spy.mock.calls.at(-1)?.[0]).toContain("/api/stats/traffic")
  })

  it("shows densified recent-logs stream error with logs jump", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    spy.mockClear()
    const user = userEvent.setup()
    renderApp(
      <AuthProvider>
        <PreferencesProvider>
          <RecentLogs
            items={[]}
            streamError="failed to fetch logs"
            streamStatus="error"
            streamPath="/api/stats/logs"
          />
        </PreferencesProvider>
      </AuthProvider>,
    )
    expect(screen.getByText("failed to fetch logs")).toBeInTheDocument()
    expect(document.querySelector('[data-slot="health-stream-error"]')).toHaveAttribute(
      "data-error-code",
      "network",
    )
    const openLinks = screen.getAllByRole("link", { name: "查看日志" })
    expect(openLinks.some((link) => link.getAttribute("href") === "/observability/logs")).toBe(true)
    await user.click(screen.getByRole("button", { name: "复制流错误" }))
    expect(spy).toHaveBeenCalled()
    expect(spy.mock.calls.at(-1)?.[0]).toContain("failed to fetch logs")
  })
})
