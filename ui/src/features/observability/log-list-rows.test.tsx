import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { LogDesktopRow, LogMobileCard } from "@/features/observability/log-list-rows"
import { i18n } from "@/i18n"
import * as exportLib from "@/features/observability/log-export"
import type { LogEvent } from "@/lib/api/types"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const item: LogEvent = {
  timestamp: "2026-07-24T00:00:00Z",
  level: "error",
  message: "inbound connection to example.com:443",
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("log list row actions", () => {
  it("copies message and full line from desktop row", async () => {
    const spy = vi.spyOn(exportLib, "copyText").mockResolvedValue()
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <table>
            <tbody>
              <LogDesktopRow item={item} />
            </tbody>
          </table>
        </MemoryRouter>
      </I18nextProvider>,
    )
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "复制消息: inbound connection to example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledWith("inbound connection to example.com:443"))
    expect(toast.success).toHaveBeenCalledWith("日志消息已复制")
    await user.click(screen.getByRole("button", { name: "复制整行: inbound connection to example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(2))
    expect(String(spy.mock.calls[1][0])).toContain("inbound connection to example.com:443")
    expect(String(spy.mock.calls[1][0])).toContain("error")
  })

  it("renders mobile copy actions", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <LogMobileCard item={item} />
        </MemoryRouter>
      </I18nextProvider>,
    )
    expect(screen.getByRole("button", { name: "复制消息: inbound connection to example.com:443" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制整行: inbound connection to example.com:443" })).toBeInTheDocument()
  })
})
