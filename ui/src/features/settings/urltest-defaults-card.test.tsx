import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { toast } from "sonner"

import { URLTestDefaultsCard } from "@/features/settings/urltest-defaults-card"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function renderCard(defaults = {
  enabled: true,
  url: "https://cp.cloudflare.com/",
  interval: "3m",
  tolerance: 50,
}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <URLTestDefaultsCard defaults={defaults} />
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("URLTestDefaultsCard", () => {
  it("disables save on invalid fields and reports save errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "internal_error", message: "urltest save failed" }), { status: 500 })))
    renderCard()
    const user = userEvent.setup()
    const interval = document.getElementById("urltest-interval") as HTMLInputElement
    const tolerance = document.getElementById("urltest-tolerance") as HTMLInputElement
    expect(interval).toHaveClass("h-8")
    expect(tolerance).toHaveClass("h-8")
    expect(screen.getByRole("button", { name: "保存 URLTest 默认值" })).toHaveClass("h-8")
    await user.clear(interval)
    await user.type(interval, "bad")
    expect(screen.getByRole("button", { name: "保存 URLTest 默认值" })).toBeDisabled()
    await user.clear(interval)
    await user.type(interval, "3m")
    await user.clear(tolerance)
    await user.type(tolerance, "999999")
    expect(screen.getByRole("button", { name: "保存 URLTest 默认值" })).toBeDisabled()
    await user.clear(tolerance)
    await user.type(tolerance, "50")
    expect(screen.getByRole("button", { name: "保存 URLTest 默认值" })).toBeDisabled()
    await user.clear(tolerance)
    await user.type(tolerance, "60")
    await user.click(screen.getByRole("button", { name: "保存 URLTest 默认值" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("urltest save failed"))
  })
})
