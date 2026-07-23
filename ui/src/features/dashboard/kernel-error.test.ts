import { describe, expect, it } from "vitest"

import {
  classifyKernelErrorMessage,
  kernelErrorHintKey,
  kernelLastErrorClipboardText,
  resolveKernelErrorCode,
} from "@/features/dashboard/kernel-error"

describe("kernel error helpers", () => {
  it("classifies common kernel failures", () => {
    expect(classifyKernelErrorMessage("invalid outbound")).toBe("config_invalid")
    expect(classifyKernelErrorMessage("restart failed after config save: boom")).toBe("restart_failed")
    expect(classifyKernelErrorMessage("listen tcp :1080: bind: address already in use")).toBe("start_failed")
    expect(classifyKernelErrorMessage("permission denied")).toBe("permission")
    expect(classifyKernelErrorMessage("no such file or directory")).toBe("config_missing")
    expect(classifyKernelErrorMessage("weird")).toBe("unknown")
  })

  it("prefers stored codes and builds clipboard diagnostics", () => {
    expect(resolveKernelErrorCode({ last_error: "x", last_error_code: "start_failed" })).toBe("start_failed")
    expect(resolveKernelErrorCode({ error: "restart failed" })).toBe("restart_failed")
    expect(kernelErrorHintKey("config_invalid")).toBe("dashboard.errorHintConfigInvalid")
    expect(kernelLastErrorClipboardText({
      running: false,
      config_path: "/data/config.json",
      version: "1.13.14",
      last_error: "invalid outbound",
      last_error_code: "config_invalid",
      last_error_at: "2026-07-23T01:02:03.000Z",
    })).toBe([
      "running: false",
      "config: /data/config.json",
      "version: 1.13.14",
      "code: config_invalid",
      "error: invalid outbound",
      "at: 2026-07-23T01:02:03.000Z",
    ].join("\n"))
    expect(kernelLastErrorClipboardText({ running: true })).toBe("")
  })
})
