import type { ReactElement } from "react"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { i18n } from "@/i18n"
import * as copy from "@/features/proxy/copy-tag-button"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function wrap(node: ReactElement) {
  return (
    <MemoryRouter>
      <I18nextProvider i18n={i18n}>{node}</I18nextProvider>
    </MemoryRouter>
  )
}

describe("ConfigSaveErrorAlert", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it("renders path actions, code, and dismiss", async () => {
    const user = userEvent.setup()
    const onJumpToPath = vi.fn()
    const onDismiss = vi.fn()
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    render(wrap(
      <ConfigSaveErrorAlert
        error={{
          message: "inbounds[0].listen_port: invalid",
          path: "inbounds[0].listen_port",
          code: "config_invalid",
          section: "inbounds",
        }}
        onJumpToPath={onJumpToPath}
        onDismiss={onDismiss}
      />,
    ))
    expect(screen.getByTestId("config-save-error")).toBeInTheDocument()
    expect(screen.getByText("config_invalid")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "跳转到路径" }))
    expect(onJumpToPath).toHaveBeenCalledWith("inbounds[0].listen_port")
    await user.click(screen.getByRole("button", { name: "复制保存错误" }))
    expect(spy).toHaveBeenCalled()
    expect(spy.mock.calls[0][0]).toContain("code: config_invalid")
    await user.click(screen.getByRole("button", { name: "关闭提示" }))
    expect(onDismiss).toHaveBeenCalled()
    expect(screen.getByRole("link", { name: "打开对应分区" })).toHaveAttribute("href", "/proxy/inbounds")
    expect(screen.getByRole("link", { name: "查看应用时间线" })).toHaveAttribute("href", "/")
  })

  it("renders plain message without jump", () => {
    render(wrap(<ConfigSaveErrorAlert error={{ message: "network down", code: "network" }} />))
    expect(screen.getByText("network down")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "跳转到路径" })).not.toBeInTheDocument()
    expect(screen.getByText("network")).toBeInTheDocument()
  })
})
