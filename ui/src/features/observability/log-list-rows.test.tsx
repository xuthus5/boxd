import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { LogDesktopRow, LogMobileCard } from "@/features/observability/log-list-rows"
import { i18n } from "@/i18n"
import type { LogEvent } from "@/lib/api/types"
import * as clipboard from "@/lib/clipboard"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const item: LogEvent = {
  timestamp: "2026-07-24T00:00:00Z",
  level: "error",
  message: "inbound connection to example.com:443",
}

function renderDesktop(log: LogEvent) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <table>
          <tbody>
            <LogDesktopRow item={log} />
          </tbody>
        </table>
      </MemoryRouter>
    </I18nextProvider>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("log list row actions", () => {
  it("copies message and full line from desktop row", async () => {
    const spy = vi.spyOn(clipboard, "copyText").mockResolvedValue()
    renderDesktop(item)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "更多操作: inbound connection to example.com:443" }))
    await user.click(await screen.findByRole("menuitem", { name: "复制消息: inbound connection to example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledWith("inbound connection to example.com:443"))
    expect(toast.success).toHaveBeenCalledWith("日志消息已复制")
    await user.click(screen.getByRole("button", { name: "更多操作: inbound connection to example.com:443" }))
    await user.click(await screen.findByRole("menuitem", { name: "复制整行: inbound connection to example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(2))
    expect(String(spy.mock.calls[1][0])).toContain("inbound connection to example.com:443")
    expect(String(spy.mock.calls[1][0])).toContain("error")
  })

  it("deep-links connections and DNS from the desktop menu", async () => {
    const user = userEvent.setup()
    const view = renderDesktop({
      timestamp: "2026-07-24T00:00:00Z",
      level: "info",
      message: "outbound connection to example.com:443",
    })
    await user.click(screen.getByRole("button", { name: "更多操作: outbound connection to example.com:443" }))
    expect(
      await screen.findByRole("menuitem", { name: "查看连接: outbound connection to example.com:443" }),
    ).toHaveAttribute("href", "/observability/connections?q=example.com")
    view.unmount()

    renderDesktop({ timestamp: "2026-07-24T00:00:00Z", level: "info", message: "dns query api.cloudflare.com" })
    await user.click(screen.getByRole("button", { name: "更多操作: dns query api.cloudflare.com" }))
    expect(await screen.findByRole("menuitem", { name: "查看 DNS: dns query api.cloudflare.com" })).toHaveAttribute(
      "href",
      "/policy/dns?rq=api.cloudflare.com",
    )
  })

  it("omits deep links for plain log lines", async () => {
    renderDesktop({ timestamp: "2026-07-24T00:00:00Z", level: "info", message: "kernel ready" })
    await userEvent.setup().click(screen.getByRole("button", { name: "更多操作: kernel ready" }))
    expect(await screen.findByRole("menuitem", { name: "复制消息: kernel ready" })).toBeInTheDocument()
    expect(screen.queryByRole("menuitem", { name: /查看连接/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("menuitem", { name: /查看 DNS/ })).not.toBeInTheDocument()
  })

  it("reports copy failures from the desktop menu", async () => {
    vi.spyOn(clipboard, "copyText").mockRejectedValue(new Error("clipboard unavailable"))
    renderDesktop(item)
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "更多操作: inbound connection to example.com:443" }))
    await user.click(await screen.findByRole("menuitem", { name: "复制整行: inbound connection to example.com:443" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalled())
  })

  it("renders mobile copy actions", async () => {
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <LogMobileCard item={item} />
        </MemoryRouter>
      </I18nextProvider>,
    )
    await userEvent.setup().click(screen.getByRole("button", { name: "更多操作: inbound connection to example.com:443" }))
    expect(await screen.findByRole("menuitem", { name: "复制消息: inbound connection to example.com:443" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "复制整行: inbound connection to example.com:443" })).toBeInTheDocument()
  })
})
