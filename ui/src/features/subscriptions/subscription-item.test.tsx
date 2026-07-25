import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { SubscriptionItem } from "@/features/subscriptions/subscription-item"
import { i18n } from "@/i18n"
import * as copy from "@/features/proxy/copy-tag-button"
import type { Subscription } from "@/lib/api/types"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const failed: Subscription = {
  id: "1",
  name: "失败订阅",
  url: "https://example.com/bad",
  interval_min: 60,
  last_updated: "2026-02-01T00:00:00Z",
  error: "subscription HTTP 403",
  error_code: "forbidden",
  outbounds: [],
}

function renderItem(item: Subscription = failed) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <SubscriptionItem item={item} onEdit={() => undefined} onRefresh={() => undefined} onDelete={() => undefined} />
      </MemoryRouter>
    </I18nextProvider>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("SubscriptionItem failure actions", () => {
  it("copies error diagnostics and url", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    const user = userEvent.setup()
    renderItem()

    await user.click(screen.getByRole("button", { name: "复制错误: 失败订阅" }))
    await waitFor(() => expect(spy).toHaveBeenCalled())
    expect(spy.mock.calls[0][0]).toContain("subscription HTTP 403")
    expect(spy.mock.calls[0][0]).toContain("code: forbidden")
    expect(toast.success).toHaveBeenCalledWith("错误信息已复制")

    await user.click(screen.getByRole("button", { name: "复制 URL: 失败订阅" }))
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(2))
    expect(spy.mock.calls[1][0]).toBe("https://example.com/bad")
  })

  it("hides open url when scheme is not http(s)", () => {
    renderItem({ ...failed, url: "ftp://example.com/bad" })
    expect(screen.queryByRole("link", { name: "打开 URL: 失败订阅" })).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制 URL: 失败订阅" })).toBeInTheDocument()
  })

  it("shows the next automatic refresh schedule", () => {
    renderItem({
      ...failed,
      error: undefined,
      error_code: undefined,
      last_updated: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    })
    expect(screen.getByText(/每 60 分钟自动刷新/)).toBeInTheDocument()
  })
})
