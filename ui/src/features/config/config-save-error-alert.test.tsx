import type { ReactElement } from "react"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { I18nextProvider } from "react-i18next"
import { describe, expect, it, vi } from "vitest"

import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { i18n } from "@/i18n"

function wrap(node: ReactElement) {
  return <I18nextProvider i18n={i18n}>{node}</I18nextProvider>
}

describe("ConfigSaveErrorAlert", () => {
  it("renders path actions and dismiss", async () => {
    const user = userEvent.setup()
    const onJumpToPath = vi.fn()
    const onDismiss = vi.fn()
    render(wrap(
      <ConfigSaveErrorAlert
        error={{ message: "inbounds[0].listen_port: invalid", path: "inbounds[0].listen_port" }}
        onJumpToPath={onJumpToPath}
        onDismiss={onDismiss}
      />,
    ))
    expect(screen.getByTestId("config-save-error")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "跳转到路径" }))
    expect(onJumpToPath).toHaveBeenCalledWith("inbounds[0].listen_port")
    await user.click(screen.getByRole("button", { name: "关闭提示" }))
    expect(onDismiss).toHaveBeenCalled()
  })

  it("renders plain message without jump", () => {
    render(wrap(<ConfigSaveErrorAlert error={{ message: "network down" }} />))
    expect(screen.getByText("network down")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "跳转到路径" })).not.toBeInTheDocument()
  })
})
