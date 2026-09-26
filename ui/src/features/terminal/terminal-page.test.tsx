import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"
import { toast } from "sonner"
import { afterEach, describe, expect, it, vi } from "vitest"

import { TerminalPage } from "@/features/terminal/terminal-page"
import { i18n } from "@/i18n"
import { copyText } from "@/lib/clipboard"
import { setupAPI } from "@/lib/api/setup"
import { setupStatus } from "@/test/setup-fixtures"

vi.mock("@/lib/clipboard", () => ({ copyText: vi.fn() }))
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

afterEach(() => {
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={["/terminal"]}>
          <TerminalPage />
        </MemoryRouter>
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

describe("TerminalPage", () => {
  it("renders per-platform commands from the detected inbound and copies them", async () => {
    vi.spyOn(setupAPI, "status").mockResolvedValue(setupStatus({
      listeners: [{ tag: "mixed-in", type: "mixed", listen: "127.0.0.1", listen_port: 1080 }],
    }))
    vi.mocked(copyText).mockResolvedValue()
    const user = userEvent.setup()
    renderPage()

    expect(await screen.findByText("当前代理地址：http://127.0.0.1:1080")).toBeInTheDocument()
    expect(screen.getByText("Linux / macOS · Bash / Zsh")).toBeInTheDocument()
    expect(screen.getByText("Windows · CMD")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "复制命令: Linux / macOS · Bash / Zsh" }))
    expect(copyText).toHaveBeenLastCalledWith(
      'export http_proxy="http://127.0.0.1:1080" https_proxy="http://127.0.0.1:1080" HTTP_PROXY="http://127.0.0.1:1080" HTTPS_PROXY="http://127.0.0.1:1080"',
    )
    expect(toast.success).toHaveBeenCalledWith("命令已复制")
  })

  it("reports clipboard failures", async () => {
    vi.spyOn(setupAPI, "status").mockResolvedValue(setupStatus({
      listeners: [{ tag: "mixed-in", type: "mixed", listen: "127.0.0.1", listen_port: 1080 }],
    }))
    vi.mocked(copyText).mockRejectedValue(new Error("clipboard unavailable"))
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole("button", { name: "复制命令: Windows · PowerShell" }))
    expect(toast.error).toHaveBeenCalledWith("复制失败，请手动复制")
  })

  it("shows a load error with retry when the setup status fails", async () => {
    vi.spyOn(setupAPI, "status").mockRejectedValue(new Error("setup unavailable"))
    const user = userEvent.setup()
    renderPage()

    expect(await screen.findByTestId("page-load-error")).toBeInTheDocument()
    expect(screen.getByText("setup unavailable")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重试" }))
    expect(setupAPI.status).toHaveBeenCalledTimes(2)
  })

  it("guides users to the inbound page when no proxy inbound exists", async () => {
    vi.spyOn(setupAPI, "status").mockResolvedValue(setupStatus({ listeners: [] }))
    renderPage()

    expect(await screen.findByText("未找到可用的代理入站")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "前往入站配置" })).toHaveAttribute("href", "/proxy/inbounds")
  })
})
