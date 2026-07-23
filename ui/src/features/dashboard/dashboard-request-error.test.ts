import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api/client"
import {
  classifyDashboardRequestError,
  dashboardRequestErrorClipboardText,
  dashboardRequestErrorHintKey,
  formatDashboardRequestErrorToast,
} from "@/features/dashboard/dashboard-request-error"

describe("dashboard request error diagnostics", () => {
  it("maps codes to hints", () => {
    expect(dashboardRequestErrorHintKey("start_failed")).toBe("dashboard.errorHintStartFailed")
    expect(dashboardRequestErrorHintKey("network")).toBe("dashboard.errorHintRequestNetwork")
    expect(dashboardRequestErrorHintKey("nope")).toBe("dashboard.errorHintUnknown")
  })

  it("classifies service and maintenance failures", () => {
    expect(classifyDashboardRequestError(new ApiError("listen tcp :1080: bind: address already in use", 500, "internal_error"))).toBe("start_failed")
    expect(classifyDashboardRequestError(new ApiError("kernel not running", 503, "unavailable"))).toBe("unavailable")
    expect(classifyDashboardRequestError(new ApiError("boom", 500, "internal_error"))).toBe("internal")
    expect(classifyDashboardRequestError(new Error("failed to fetch"))).toBe("network")
  })

  it("formats toast and clipboard payloads", () => {
    expect(formatDashboardRequestErrorToast(new ApiError("boom", 500, "internal_error"), "fallback")).toBe("internal: boom")
    expect(formatDashboardRequestErrorToast(new Error("mystery"), "fallback")).toBe("mystery")
    const payload = dashboardRequestErrorClipboardText(new ApiError("bind failed", 500, "internal_error"), {
      scope: "service",
      action: "start",
    })
    expect(payload).toContain("scope: service")
    expect(payload).toContain("action: start")
    expect(payload).toContain("code: start_failed")
  })
})
