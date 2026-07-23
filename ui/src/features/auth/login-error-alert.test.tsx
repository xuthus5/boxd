import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

import { renderApp } from "@/test/render"
import { LoginErrorAlert } from "@/features/auth/login-error-alert"

describe("LoginErrorAlert", () => {
  it("renders code, hint, and copy control", async () => {
    const user = userEvent.setup()
    renderApp(<LoginErrorAlert error={{ message: "invalid credentials", code: "unauthorized" }} />)
    expect(screen.getByTestId("login-error")).toHaveAttribute("data-error-code", "unauthorized")
    expect(screen.getByText("unauthorized")).toBeInTheDocument()
    expect(screen.getByText("invalid credentials")).toBeInTheDocument()
    expect(screen.getByText(/用户名或密码不正确/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "复制登录错误" }))
  })

  it("renders nothing without error", () => {
    const { container } = renderApp(<LoginErrorAlert error={null} />)
    expect(container.querySelector("[data-testid=login-error]")).toBeNull()
  })
})
