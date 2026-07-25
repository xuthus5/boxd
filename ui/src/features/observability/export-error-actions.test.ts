import { beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { toast } from "sonner"

import { reportExportError } from "@/features/observability/export-error-actions"

describe("export error actions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("reports densified export failures with clipboard action", () => {
    const t = (key: string) => key
    reportExportError(new Error("clipboard unavailable"), t, {
      scope: "logs",
      kind: "copy",
      count: 2,
      fallback: "observability.logsCopyFailed",
    })
    expect(toast.error).toHaveBeenCalled()
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toContain("clipboard_unavailable")
    expect(options).toEqual(expect.objectContaining({
      description: "observability.errorHintClipboardUnavailable",
      action: expect.objectContaining({ label: "observability.copyExportError" }),
    }))
  })

  it("uses the default fallback and omits an empty diagnostic action", () => {
    reportExportError(new Error(""), (key) => key)
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(message).toBe("observability.exportFailed")
    expect(options?.action).toBeUndefined()
  })
})
