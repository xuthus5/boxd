import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"

import { ConnectionDesktopRow, ConnectionMobileCard } from "@/features/observability/connection-list-rows"
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

  it("renders all desktop cells, deep links, and close actions", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    const onClose = vi.fn()
    const fullConnection: Connection = {
      ...connection,
      source: "10.0.0.2:1234",
      inbound: "mixed-in",
      protocol: "tls",
      process: "/usr/bin/curl",
    }
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <table>
            <tbody>
              <ConnectionDesktopRow
                connection={fullConnection}
                columns={["target", "source", "network", "inbound", "outbound", "rule", "protocol", "process", "upload", "download", "duration", "actions"]}
                busy={false}
                onClose={onClose}
              />
            </tbody>
          </table>
        </MemoryRouter>
      </I18nextProvider>,
    )
    const user = userEvent.setup()
    expect(screen.getByRole("link", { name: "查看日志: api.example.com:443" })).toHaveAttribute(
      "href",
      "/observability/logs?q=api.example.com",
    )
    expect(screen.getByRole("link", { name: "查看节点: proxy" })).toHaveAttribute("href", "/nodes?q=proxy")
    expect(screen.getByRole("link", { name: "查看规则: geosite-google" })).toHaveAttribute(
      "href",
      "/policy/route?q=geosite-google",
    )
    expect(screen.getByRole("link", { name: "网络: tcp" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "出站: proxy" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "规则: geosite-google" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "进程: /usr/bin/curl" })).toBeInTheDocument()
    expect(screen.getByTitle("10.0.0.2:1234")).toBeInTheDocument()
    expect(screen.getByText("mixed-in")).toBeInTheDocument()
    expect(screen.getByText("tls")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "复制目标: api.example.com:443" }))
    await user.click(screen.getByRole("button", { name: "复制连接: api.example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(2))
    expect(toast.success).toHaveBeenCalledWith("目标已复制")
    expect(toast.success).toHaveBeenCalledWith("连接信息已复制")
    await user.click(screen.getByRole("button", { name: "关闭" }))
    expect(onClose).toHaveBeenCalledWith("7")
  })

  it("handles missing desktop values and copy failures", async () => {
    const spy = vi.spyOn(copy, "copyText").mockRejectedValue(new Error("clipboard unavailable"))
    const sparse: Connection = {
      id: 8,
      target: "sparse.example:443",
      outbound: "",
      rule: "—",
      upload: 0,
      download: 0,
      start: "invalid",
    }
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <table>
            <tbody>
              <ConnectionDesktopRow
                connection={sparse}
                columns={["target", "network", "outbound", "rule", "process", "actions"]}
                busy
                onClose={vi.fn()}
              />
            </tbody>
          </table>
        </MemoryRouter>
      </I18nextProvider>,
    )
    const user = userEvent.setup()
    expect(screen.queryByRole("link", { name: /查看节点/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: /查看规则/ })).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "关闭" })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "复制目标: sparse.example:443" }))
    await user.click(screen.getByRole("button", { name: "复制连接: sparse.example:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(2))
    expect(toast.error).toHaveBeenCalled()
  })
})

describe("ConnectionMobileCard", () => {
  it("renders mobile metadata and actions", async () => {
    const spy = vi.spyOn(copy, "copyText").mockResolvedValue()
    const onClose = vi.fn()
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <ConnectionMobileCard
            connection={{ ...connection, source: "10.0.0.2:1234", inbound: "mixed-in", protocol: "tls", process: "/usr/bin/curl" }}
            columns={["target", "network", "inbound", "outbound", "rule", "protocol", "process", "source", "upload", "download", "duration", "actions"]}
            busy={false}
            onClose={onClose}
          />
        </MemoryRouter>
      </I18nextProvider>,
    )
    const user = userEvent.setup()
    expect(screen.getByRole("link", { name: "查看日志: api.example.com:443" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看节点: proxy" })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看规则: geosite-google" })).toBeInTheDocument()
    expect(screen.getByTitle("10.0.0.2:1234")).toBeInTheDocument()
    expect(screen.getByText(/上传:/)).toBeInTheDocument()
    expect(screen.getByText(/下载:/)).toBeInTheDocument()
    expect(screen.getByText(/时长:/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "复制目标: api.example.com:443" }))
    await user.click(screen.getByRole("button", { name: "复制连接: api.example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledTimes(2))
    await user.click(screen.getByRole("button", { name: "关闭" }))
    expect(onClose).toHaveBeenCalledWith("7")
  })

  it("omits optional mobile sections and disables close when busy", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <ConnectionMobileCard
            connection={{ ...connection, target: "", outbound: "", rule: "" }}
            columns={["actions"]}
            busy
            onClose={vi.fn()}
          />
        </MemoryRouter>
      </I18nextProvider>,
    )
    expect(screen.queryByRole("link", { name: /查看日志/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: /查看节点/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: /查看规则/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /复制目标/ })).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "关闭" })).toBeDisabled()
  })

  it("reports mobile target copy failures", async () => {
    const spy = vi.spyOn(copy, "copyText").mockRejectedValue(new Error("clipboard unavailable"))
    render(
      <I18nextProvider i18n={i18n}>
        <MemoryRouter>
          <ConnectionMobileCard
            connection={connection}
            columns={["target"]}
            busy={false}
            onClose={vi.fn()}
          />
        </MemoryRouter>
      </I18nextProvider>,
    )
    await userEvent.setup().click(screen.getByRole("button", { name: "复制目标: api.example.com:443" }))
    await waitFor(() => expect(spy).toHaveBeenCalledWith("api.example.com:443"))
    expect(toast.error).toHaveBeenCalled()
  })
})
