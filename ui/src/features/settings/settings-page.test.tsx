import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { toast } from "sonner"

import { RuntimeSettingsCard } from "@/features/settings/settings-page"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))
vi.mock("@/lib/api/desktop", () => ({ isDesktop: vi.fn(), desktopRequest: vi.fn() }))

import { desktopRequest, isDesktop } from "@/lib/api/desktop"

function renderCard(appAutostart = false, desktop = true) {
  vi.mocked(isDesktop).mockReturnValue(desktop)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <RuntimeSettingsCard url="https://www.gstatic.com/generate_204" enabled={false} appAutostart={appAutostart} />
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("RuntimeSettingsCard app autostart", () => {
  it("hides the app autostart switch in web mode", () => {
    renderCard(false, false)
    expect(screen.queryByText("开机自启动")).not.toBeInTheDocument()
  })

  it("shows the app autostart switch with current state in desktop mode", () => {
    renderCard(true, true)
    expect(screen.getByText("开机自启动")).toBeInTheDocument()
    expect(screen.getByText("登录系统时自动启动 boxd 桌面应用")).toBeInTheDocument()
    expect(screen.getByRole("switch", { name: "开机自启动" })).toBeChecked()
  })

  it("enables app autostart via the bridge and surfaces success", async () => {
    vi.mocked(desktopRequest).mockResolvedValue({ enabled: true })
    renderCard(false, true)
    const user = userEvent.setup()
    await user.click(screen.getByRole("switch", { name: "开机自启动" }))
    await waitFor(() => expect(desktopRequest).toHaveBeenCalled())
    expect(desktopRequest).toHaveBeenCalledWith("/api/desktop/autostart", "PUT", JSON.stringify({ enabled: true }))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith("开机自启动已更新"))
  })

  it("reverts the switch when the bridge call fails", async () => {
    vi.mocked(desktopRequest).mockRejectedValue(new Error("boom"))
    renderCard(false, true)
    const user = userEvent.setup()
    const toggle = screen.getByRole("switch", { name: "开机自启动" })
    await user.click(toggle)
    await waitFor(() => expect(toggle).not.toBeChecked())
    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("boom"), expect.objectContaining({
      description: expect.any(String),
    }))
  })
})
