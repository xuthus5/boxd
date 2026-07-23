import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { ConnectionDesktopRow } from "@/features/observability/connection-list-rows"
import { i18n } from "@/i18n"
import * as copy from "@/features/proxy/copy-tag-button"
import type { Connection } from "@/lib/api/types"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const connection: Connection = {
  id: 7,
  target: "api.example.com:443",
  outbound: "proxy",
  rule: "geosite-google",
  network: "tcp",
  upload: 10,
  download: 20,
  start: "2026-07-24T00:00:00Z",
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("ConnectionDesktopRow", () => {
  it("copies full connection diagnostics", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <table>
            <tbody>
              <ConnectionDesktopRow
                connection={connection}
                columns={["target", "outbound", "actions"]}
                busy={false}
                onClose={vi.fn()}
              />
            </tbody>
          </table>
        </MemoryRouter>
      </I18nextProvider>,
    )
    await userEvent.setup().click(screen.getByRole("button", { name: "复制连接: api.example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalled())
    expect(String(spy.mock.calls[0][0])).toContain("id: 7")
    expect(String(spy.mock.calls[0][0])).toContain("target: api.example.com:443")
    expect(String(spy.mock.calls[0][0])).toContain("outbound: proxy")
    expect(toast.success).toHaveBeenCalledWith("连接信息已复制")
  })
})
