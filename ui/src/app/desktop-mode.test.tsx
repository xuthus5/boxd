import { screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/features/dashboard/dashboard-page", () => ({
  DashboardPage: () => <div>desktop dashboard</div>,
}))

vi.mock("@/lib/api/desktop", () => ({
  isDesktop: vi.fn(() => true),
  autoLogin: vi.fn(() => Promise.resolve({ token: "desktop-token", expires_at: "2099-01-01T00:00:00Z" })),
  desktopRequest: vi.fn(),
  desktopGet: vi.fn(() => Promise.resolve(null)),
  subscribeDesktopStream: vi.fn(() => Promise.resolve(() => undefined)),
  streamEventName: vi.fn(() => null),
  callBridge: vi.fn(),
}))

import App from "@/App"
import { desktopRequest } from "@/lib/api/desktop"
import { sessionStore } from "@/lib/session"
import { renderApp } from "@/test/render"

function desktopPayload(path: string): unknown {
  if (path === "/api/settings/preferences") return { theme: "system", language: "zh", minimumLogLevel: "all" }
  if (path === "/api/settings/url-test") return { url: "https://example.com/generate_204" }
  if (path === "/api/settings/kernel-autostart") return { enabled: false }
  if (path === "/api/settings/urltest-defaults") {
    return { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 }
  }
  if (path === "/api/config/rule-sets/auto-update") return { enabled: false, interval: "24h" }
  if (path === "/api/desktop/autostart") return { enabled: false }
  if (path === "/api/service/status") return { running: true, uptime: "1m" }
  return {}
}

beforeEach(() => {
  vi.mocked(desktopRequest).mockImplementation((path) => Promise.resolve(desktopPayload(path)))
})

afterEach(() => {
  vi.clearAllMocks()
  sessionStore.clear()
})

describe("desktop mode feature gating", () => {
  it("opens the application without a login session", async () => {
    renderApp(<App />, "/login")

    expect(await screen.findByText("desktop dashboard")).toBeInTheDocument()
    expect(screen.queryByLabelText("用户名")).not.toBeInTheDocument()
  })

  it("hides logout and account security settings", async () => {
    renderApp(<App />, "/settings")

    expect(await screen.findByRole("heading", { name: "应用设置" })).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "退出登录" })).not.toBeInTheDocument()
    expect(screen.queryByText("登录与安全")).not.toBeInTheDocument()
    const passwordCalls = vi.mocked(desktopRequest).mock.calls.filter(([path]) => path === "/api/settings/password")
    const jwtCalls = vi.mocked(desktopRequest).mock.calls.filter(([path]) => path === "/api/settings/jwt-secret")
    expect(passwordCalls).toHaveLength(0)
    expect(jwtCalls).toHaveLength(0)
  })
})
