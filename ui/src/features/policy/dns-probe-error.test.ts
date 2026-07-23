import { describe, expect, it } from "vitest"

import {
  classifyDNSProbeErrorMessage,
  dnsProbeErrorClipboardText,
  dnsProbeErrorHintKey,
  formatDNSProbeFailureSample,
  resolveDNSProbeErrorCode,
} from "@/features/policy/dns-probe-error"

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
})
