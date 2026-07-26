import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { renderApp } from "@/test/render"
import { ApiError } from "@/lib/api/client"
import { CardQueryError } from "@/features/common/card-query-error"

describe("CardQueryError", () => {
  it("renders code, hint, and copy control", async () => {
    const user = userEvent.setup()
    renderApp(
      <CardQueryError
        error={new ApiError("kernel not running", 503, "unavailable")}
        scope="proxy-selector"
        path="/api/service/status"
      />,
    )
    expect(screen.getByTestId("card-query-error")).toHaveAttribute("data-error-code", "unavailable")
    expect(screen.getByText("unavailable")).toBeInTheDocument()
    expect(screen.getByText("kernel not running")).toBeInTheDocument()
    expect(screen.getByText(/服务暂不可用|Service is temporarily unavailable/i)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /复制加载错误|Copy load error/i }))
  })

  it("invokes retry when provided", async () => {
    const user = userEvent.setup()
    const onRetry = vi.fn()
    renderApp(<CardQueryError error={new Error("boom")} scope="x" onRetry={onRetry} />)
    await user.click(screen.getByRole("button", { name: /重试|Retry/i }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })
})
