import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"

import { FailedSubscriptionsPreview } from "@/features/dashboard/failed-subscriptions-preview"
import { i18n } from "@/i18n"
import type { Subscription } from "@/lib/api/types"

const items: Subscription[] = [
  {
    id: "1",
    name: "主订阅",
    url: "https://example.com/a",
    interval_min: 60,
    last_updated: "2026-01-01T00:00:00Z",
    error: "subscription HTTP 403",
    error_code: "forbidden",
  },
  {
    id: "2",
    name: "备用",
    url: "https://example.com/b",
    interval_min: 60,
    last_updated: "2026-01-02T00:00:00Z",
    error: "timeout",
    error_code: "timeout",
  },
]

function renderPreview(preview = items.slice(0, 1), total = 2) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <FailedSubscriptionsPreview items={preview} total={total} />
      </MemoryRouter>
    </I18nextProvider>,
  )
}

describe("FailedSubscriptionsPreview", () => {
  it("renders failed rows with deep links and remaining count", () => {
    renderPreview()
    expect(screen.getByText("主订阅")).toBeInTheDocument()
    expect(screen.getByText("subscription HTTP 403")).toBeInTheDocument()
    expect(screen.getByText("forbidden")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看: 主订阅" })).toHaveAttribute(
      "href",
      "/subscriptions?q=%E4%B8%BB%E8%AE%A2%E9%98%85&status=error",
    )
    expect(screen.getByText(/还有 1 个/)).toBeInTheDocument()
  })

  it("renders nothing when empty", () => {
    const { container } = renderPreview([], 0)
    expect(container.querySelector('[data-slot="failed-subscriptions-preview"]')).toBeNull()
  })
})
