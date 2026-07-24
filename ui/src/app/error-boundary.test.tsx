import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { ErrorBoundary } from "@/app/error-boundary"
import { renderApp } from "@/test/render"

function BrokenPage(): never {
  throw new Error("failed to fetch upstream")
}

describe("ErrorBoundary", () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("shows densified recoverable error surface when a page throws", async () => {
    const user = userEvent.setup()
    vi.spyOn(console, "error").mockImplementation(() => undefined)

    renderApp(
      <ErrorBoundary>
        <BrokenPage />
      </ErrorBoundary>,
    )

    const fallback = screen.getByTestId("page-load-error")
    expect(fallback).toHaveAttribute("data-error-code", "network")
    expect(screen.getByText("页面出现异常")).toBeInTheDocument()
    expect(screen.getByText("failed to fetch upstream")).toBeInTheDocument()
    expect(screen.getByText("network")).toBeInTheDocument()
    expect(screen.getByText(/网络连接失败|Network error/i)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重试" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /复制加载错误|Copy load error/i }))
  })
})
