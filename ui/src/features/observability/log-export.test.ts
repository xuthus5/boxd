import { afterEach, describe, expect, it, vi } from "vitest"

import {
  buildLogExportFilename,
  downloadTextFile,
  formatLogExport,
  formatLogLine,
  formatLogMessage,
  formatLogTimestamp,
} from "@/features/observability/log-export"

describe("log-export", () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("formats lines and export payload", () => {
    expect(formatLogLine({ level: "info", message: "ready", timestamp: "2026-01-01T00:00:00Z" }))
      .toBe("2026-01-01T00:00:00Z\tinfo\tready")
    expect(formatLogExport([
      { level: "info", message: "a", timestamp: "t1" },
      { level: "error", message: "b", timestamp: "" },
    ])).toBe("t1\tinfo\ta\n-\terror\tb")
  })

  it("builds a stable filename", () => {
    expect(buildLogExportFilename("Kernel Logs", new Date("2026-07-23T01:02:03.004Z")))
      .toBe("boxd-kernel-logs-2026-07-23T01-02-03-004Z.log")
  })

  it("downloads text via temporary anchor", () => {
    const click = vi.fn()
    const remove = vi.fn()
    const appendChild = vi.fn()
    const createElement = vi.fn().mockReturnValue({
      href: "",
      download: "",
      rel: "",
      click,
      remove,
    })
    const createObjectURL = vi.fn().mockReturnValue("blob:mock")
    const revokeObjectURL = vi.fn()
    const originalCreate = URL.createObjectURL
    const originalRevoke = URL.revokeObjectURL
    // @ts-expect-error test stub
    URL.createObjectURL = createObjectURL
    // @ts-expect-error test stub
    URL.revokeObjectURL = revokeObjectURL
    try {
      downloadTextFile("a.log", "body", {
        createElement,
        body: { appendChild },
      } as unknown as Document)
      expect(createElement).toHaveBeenCalledWith("a")
      expect(appendChild).toHaveBeenCalled()
      expect(click).toHaveBeenCalled()
      expect(remove).toHaveBeenCalled()
      expect(revokeObjectURL).toHaveBeenCalledWith("blob:mock")
    } finally {
      URL.createObjectURL = originalCreate
      URL.revokeObjectURL = originalRevoke
    }
  })
  it("formats message-only clipboard text", () => {
    expect(formatLogMessage({ timestamp: "t", level: "info", message: " hello " })).toBe("hello")
    expect(formatLogMessage({ message: "" })).toBe("")
  })

  it("formats display timestamps safely", () => {
    expect(formatLogTimestamp()).toBe("—")
    expect(formatLogTimestamp("invalid")).toBe("—")
    expect(formatLogTimestamp("2026-07-24T00:00:00Z")).not.toBe("—")
  })

})
