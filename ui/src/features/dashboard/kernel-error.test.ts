import { describe, expect, it } from "vitest"

import { en } from "@/i18n/locales/en"
import { zh } from "@/i18n/locales/zh"
import {
  classifyKernelErrorMessage,
  kernelErrorHintKey,
  kernelLastErrorClipboardText,
  resolveKernelErrorCode,
} from "@/features/dashboard/kernel-error"

describe("kernel error helpers", () => {
  it.each([
    "start inbound/tun[tun-in]: configure tun interface: set ipv6 address: element not found",
    "SET IPV6 ADDRESS: The system cannot find the file specified",
    "restart failed after config save: add address fdfe:dcba:9876::1/126: permission denied",
    "add address ::1/128: address family not supported",
    "add address ::/0: operation not permitted",
    "add address 192.0.2.1/24: denied; add address 2001:db8::1/64: permission denied",
  ])("identifies explicit IPv6 address setup errors: %s", (message) => {
    expect(classifyKernelErrorMessage(message)).toBe("ipv6_unavailable")
    expect(kernelErrorHintKey("ipv6_unavailable")).toBe("dashboard.errorHintIPv6Unavailable")
  })

  it.each([
    "permission denied",
    "add address 192.0.2.1/24: permission denied",
    "add address 2001:db8:::1/64: permission denied",
    "add address fdfe::1/129: permission denied",
    "add address fdfe::1/064: permission denied",
    "add address fdfe::1: permission denied",
    "add address : permission denied",
    "listen tcp [::]:1080: permission denied",
  ])("keeps unrelated permission errors distinct: %s", (message) => {
    expect(classifyKernelErrorMessage(message)).toBe("permission")
  })

  it.each(["config_restart_failed", "config_missing", "permission"])(
    "recovers IPv6 context from a generic saved error code: %s",
    (error_code) => {
      expect(resolveKernelErrorCode({
        error_code,
        error: "restart failed after config save: set ipv6 address: element not found",
      })).toBe("ipv6_unavailable")
    },
  )

  it("offers IPv4 and adapter binding checks for unavailable IPv6", () => {
    const hints = [
      en.translation.dashboard.errorHintIPv6Unavailable,
      zh.translation.dashboard.errorHintIPv6Unavailable,
    ]
    for (const hint of hints) {
      expect(hint).toContain("IPv4")
      expect(hint).toMatch(/unavailable|不可用/)
      expect(hint).toMatch(/IPv6.*(?:binding|绑定)/)
    }
  })

  it("preserves IPv6 context in saved status and clipboard diagnostics", () => {
    const last_error = "set ipv6 address: element not found"
    expect(resolveKernelErrorCode({ last_error, last_error_code: "config_missing" })).toBe("ipv6_unavailable")
    expect(kernelLastErrorClipboardText({ last_error, last_error_code: "config_missing" }))
      .toBe(`code: ipv6_unavailable\nerror: ${last_error}`)
    expect(resolveKernelErrorCode({ error_code: "ipv6_unavailable" })).toBe("ipv6_unavailable")
  })

  it("handles absent diagnostics and optional metadata", () => {
    expect(classifyKernelErrorMessage()).toBe("unknown")
    expect(resolveKernelErrorCode({})).toBeUndefined()
    expect(kernelErrorHintKey()).toBe("dashboard.errorHintUnknown")
    expect(kernelErrorHintKey("custom_failure")).toBe("dashboard.errorHintUnknown")
    expect(kernelLastErrorClipboardText({ running: true, last_error: "permission denied" }))
      .toBe("running: true\ncode: permission\nerror: permission denied")
  })

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
