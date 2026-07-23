import { beforeEach, describe, expect, it, vi } from "vitest"

import { ApiError } from "@/lib/api/client"

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { toast } from "sonner"

import { reportDashboardRequestError } from "@/features/dashboard/dashboard-request-error-actions"

describe("dashboard request error actions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("reports densified service failures with clipboard action", () => {
    const t = (key: string) => key
    reportDashboardRequestError(
      new ApiError("listen tcp :1080: bind: address already in use", 500, "internal_error"),
      t,
      { scope: "service", action: "start", fallback: "dashboard.serviceActionFailed" },
    )
    expect(toast.error).toHaveBeenCalled()
    const [message, options] = vi.mocked(toast.error).mock.calls.at(-1)!
    expect(String(message)).toContain("start_failed")
    expect(options).toEqual(expect.objectContaining({
      description: "dashboard.errorHintStartFailed",
      action: expect.objectContaining({ label: "dashboard.copyRequestError" }),
    }))
  })
})
