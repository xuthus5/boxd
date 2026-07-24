import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { renderApp } from "@/test/render"
import { ApiError } from "@/lib/api/client"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

describe("PageLoadErrorAlert", () => {
  it("renders code, hint, and copy control", async () => {
    const user = userEvent.setup()
    renderApp(
      <PageLoadErrorAlert
        error={new ApiError("service unavailable", 503, "unavailable")}
        scope="dashboard"
      />,
    )
    expect(screen.getByTestId("page-load-error")).toHaveAttribute("data-error-code", "unavailable")
    expect(screen.getByText("unavailable")).toBeInTheDocument()
    expect(screen.getByText("service unavailable")).toBeInTheDocument()
    expect(screen.getByText(/服务暂不可用|Service is temporarily unavailable/i)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /复制加载错误|Copy load error/i }))
  })

  it("invokes retry when provided", async () => {
    const user = userEvent.setup()
    const onRetry = vi.fn()
    renderApp(<PageLoadErrorAlert error={new Error("boom")} onRetry={onRetry} />)
    await user.click(screen.getByRole("button", { name: /重试|Retry/i }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })
})
