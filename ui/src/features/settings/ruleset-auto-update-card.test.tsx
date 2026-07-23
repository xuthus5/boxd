import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { toast } from "sonner"

import { RuleSetAutoUpdateCard } from "@/features/settings/ruleset-auto-update-card"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function renderCard(defaults = { enabled: false, interval: "24h" }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <RuleSetAutoUpdateCard defaults={defaults} />
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("RuleSetAutoUpdateCard", () => {
  it("disables save for invalid interval and enables after fix", async () => {
    renderCard({ enabled: true, interval: "24h" })
    const user = userEvent.setup()
    const input = screen.getByLabelText("自动更新间隔")
    const save = screen.getByRole("button", { name: "保存" })
    expect(input).toHaveClass("h-8")
    expect(save).toHaveClass("h-8")
    expect(save).toBeDisabled()
    await user.clear(input)
    await user.type(input, "bad")
    expect(save).toBeDisabled()
    expect(screen.getByText("请输入大于 0 的时长，例如 3m 或 30s。")).toBeInTheDocument()
    await user.clear(input)
    await user.type(input, "12h")
    expect(save).toBeEnabled()
  })

  it("saves enabled interval and surfaces success toast", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ enabled: true, interval: "12h" })))
    vi.stubGlobal("fetch", fetchMock)
    renderCard({ enabled: false, interval: "24h" })
    const user = userEvent.setup()
    await user.click(screen.getByRole("switch", { name: "启用 local 规则集定时更新" }))
    await user.clear(screen.getByLabelText("自动更新间隔"))
    await user.type(screen.getByLabelText("自动更新间隔"), "12h")
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    expect(fetchMock).toHaveBeenCalledWith("/api/config/rule-sets/auto-update", expect.objectContaining({
      method: "PUT",
      body: JSON.stringify({ enabled: true, interval: "12h" }),
    }))
    await waitFor(() => expect(toast.success).toHaveBeenCalled())
  })

  it("reports save errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "internal_error", message: "ruleset save failed" }), { status: 500 })))
    renderCard()
    const user = userEvent.setup()
    await user.click(screen.getByRole("switch", { name: "启用 local 规则集定时更新" }))
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("ruleset save failed"))
  })
})
