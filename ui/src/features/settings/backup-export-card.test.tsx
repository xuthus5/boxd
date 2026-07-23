import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { toast } from "sonner"

import { BackupExportCard } from "@/features/settings/backup-export-card"
import { i18n } from "@/i18n"
import * as client from "@/lib/api/client"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function renderCard() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <BackupExportCard />
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("BackupExportCard", () => {
  it("downloads a backup archive and reports success", async () => {
    const trigger = vi.spyOn(client, "triggerBrowserDownload").mockImplementation(() => undefined)
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(new Uint8Array([1, 2, 3]), {
      status: 200,
      headers: {
        "Content-Type": "application/gzip",
        "Content-Disposition": 'attachment; filename="boxd-backup-test.tar.gz"',
      },
    })))

    renderCard()
    await userEvent.setup().click(screen.getByRole("button", { name: "导出备份" }))

    await waitFor(() => expect(trigger).toHaveBeenCalled())
    expect(trigger.mock.calls[0][1]).toBe("boxd-backup-test.tar.gz")
    expect(toast.success).toHaveBeenCalledWith(expect.stringContaining("boxd-backup-test.tar.gz"))
  })

  it("surfaces export failures", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "error",
      data: null,
      error: { code: "internal_error", message: "failed to create backup" },
      meta: null,
    }), { status: 500 })))

    renderCard()
    await userEvent.setup().click(screen.getByRole("button", { name: "导出备份" }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("failed to create backup"))
  })
})
