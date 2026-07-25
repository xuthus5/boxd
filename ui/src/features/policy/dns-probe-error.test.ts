import { describe, expect, it } from "vitest"

import {
  classifyDNSProbeErrorMessage,
  classifyDNSProbeRequestError,
  dnsProbeErrorClipboardText,
  dnsProbeErrorHintKey,
  dnsProbeRequestErrorClipboardText,
  formatDNSProbeFailureSample,
  formatDNSProbeRequestErrorToast,
  resolveDNSProbeErrorCode,
} from "@/features/policy/dns-probe-error"
import { ApiError } from "@/lib/api/client"

describe("dns probe error helpers", () => {
  it("formats clipboard diagnostics", () => {
    expect(dnsProbeErrorClipboardText({
      tag: "cf",
      type: "udp",
      domain: "example.com",
      success: false,
      error: "network down",
      error_code: "network",
    })).toBe([
      "tag: cf",
      "type: udp",
      "domain: example.com",
      "code: network",
      "error: network down",
    ].join("\n"))
    expect(dnsProbeErrorClipboardText({ tag: "cf", success: true })).toBe("")
  })

  it("classifies messages and prefers stored code", () => {
    expect(classifyDNSProbeErrorMessage("dns rcode SERVFAIL")).toBe("dns_rcode")
    expect(classifyDNSProbeErrorMessage("empty dns response")).toBe("empty_response")
    expect(classifyDNSProbeErrorMessage("dns type local is not probeable")).toBe("unsupported")
    expect(resolveDNSProbeErrorCode({ error: "x", error_code: "timeout" })).toBe("timeout")
    expect(resolveDNSProbeErrorCode({ error: "i/o timeout" })).toBe("timeout")
    expect(dnsProbeErrorHintKey("dns_rcode")).toBe("policy.dns.errorHintDNSRcode")
    expect(formatDNSProbeFailureSample({ error: "boom", error_code: "timeout" })).toBe("timeout: boom")
    expect(formatDNSProbeFailureSample({ error: "timeout", error_code: "timeout" })).toBe("timeout")
  })

  it("classifies request-level probe failures", () => {
    expect(classifyDNSProbeRequestError(new ApiError("service not available", 503, "unavailable"))).toBe("unavailable")
    expect(classifyDNSProbeRequestError(new Error("i/o timeout"))).toBe("timeout")
    expect(dnsProbeRequestErrorClipboardText(new Error("network down"))).toContain("code: network")
    expect(formatDNSProbeRequestErrorToast(new Error("boom"), (k) => k, "fallback")).toBe("boom")
    expect(formatDNSProbeRequestErrorToast(
      new ApiError("kernel offline", 503, "unavailable"),
      (k) => k,
      "fallback",
    )).toBe("unavailable: kernel offline")
  })

  it("covers every probe message classification", () => {
    const cases = [
      [undefined, "unknown"],
      ["", "unknown"],
      ["probe is not available", "unavailable"],
      ["unsupported DNS type", "unsupported"],
      ["server is not probeable", "unsupported"],
      ["invalid server address", "invalid_input"],
      ["server is required", "invalid_input"],
      ["empty address", "invalid_input"],
      ["no response received", "no_response"],
      ["empty dns response", "empty_response"],
      ["empty response body", "empty_response"],
      ["dns rcode NXDOMAIN", "dns_rcode"],
      ["request deadline exceeded", "timeout"],
      ["connection refused", "network"],
      ["connection reset by peer", "network"],
      ["no such host", "network"],
      ["network unavailable", "network"],
      ["broken pipe", "network"],
      ["unexpected failure", "unknown"],
    ] as const

    for (const [message, expectedCode] of cases) {
      const actualCode = classifyDNSProbeErrorMessage(message)
      expect(actualCode).toBe(expectedCode)
    }
    expect(dnsProbeErrorHintKey()).toBe("policy.dns.errorHintUnknown")
    expect(dnsProbeErrorHintKey("future")).toBe("policy.dns.errorHintUnknown")
  })

  it("handles missing result fields and all clipboard shapes", () => {
    const expectedUndefined = undefined
    const actualSuccess = resolveDNSProbeErrorCode({ success: true, error: "ignored", error_code: "network" })
    expect(actualSuccess).toBe(expectedUndefined)
    const actualEmpty = resolveDNSProbeErrorCode({ success: false })
    expect(actualEmpty).toBe(expectedUndefined)
    expect(resolveDNSProbeErrorCode({ success: false, error: "connection reset" })).toBe("network")

    const expectedMinimal = ""
    const actualMinimal = dnsProbeErrorClipboardText({ success: false })
    expect(actualMinimal).toBe(expectedMinimal)
    expect(dnsProbeErrorClipboardText({
      success: false,
      tag: "  ",
      type: "  ",
      domain: "  ",
      error: "  ",
      error_code: "",
    })).toBe("code: unknown")
    expect(formatDNSProbeFailureSample({ error: "", error_code: "unknown" })).toBe("failed")
    expect(formatDNSProbeFailureSample({ error: "network", error_code: "unknown" })).toBe("network")
    expect(formatDNSProbeFailureSample({ error: "", error_code: "network" })).toBe("network: failed")
  })

  it("maps request error codes and fallback messages", () => {
    const requestCases = [
      [{ code: "unavailable" }, "unavailable"],
      [{ code: "invalid_request" }, "invalid_input"],
      [{ code: "invalid_input" }, "invalid_input"],
      [{ code: "timeout" }, "timeout"],
      [{ code: "unsupported" }, "unsupported"],
      [{ code: "future" }, "unknown"],
      [new Error("broken pipe"), "network"],
      [null, "unknown"],
    ] as const
    for (const [error, expectedCode] of requestCases) {
      const actualCode = classifyDNSProbeRequestError(error)
      expect(actualCode).toBe(expectedCode)
    }
    expect(dnsProbeRequestErrorClipboardText("  ")).toBe("")
    expect(dnsProbeRequestErrorClipboardText("network down")).toBe("code: network\nerror: network down")
    expect(formatDNSProbeRequestErrorToast(new Error("  "), (key) => key, "fallback")).toBe("fallback")
    expect(formatDNSProbeRequestErrorToast("", (key) => key, "fallback")).toBe("fallback")
    expect(formatDNSProbeRequestErrorToast("broken pipe", (key) => key, "fallback")).toBe("network: fallback")
  })
})
