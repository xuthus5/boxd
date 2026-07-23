import { describe, expect, it } from "vitest"

import {
  dnsProbeBatchFailureClipboardText,
  dnsProbeBatchToastTone,
  dnsProbeInput,
  formatDNSProbeBatchMessage,
  isDNSServerProbeable,
  mapDNSProbeBatchResults,
  summarizeDNSProbeResults,
} from "@/features/policy/dns-probe"
import type { DNSProbeResult } from "@/lib/api/types"

const sample: DNSProbeResult[] = [
  { tag: "cf", type: "udp", success: true, latency_ms: 12 },
  { tag: "google", type: "https", success: true, latency_ms: 30 },
  { tag: "bad", type: "udp", success: false, error: "timeout" },
]

const t = (key: string, values?: Record<string, string | number>) => {
  if (key === "policy.dns.probeBatchComplete") return "done"
  if (key === "policy.dns.probeBatchSummary") {
    return `${values?.success}/${values?.total} ok, ${values?.failed} failed, avg ${values?.avg}`
  }
  if (key === "policy.dns.probeBatchBest") return `best ${values?.tag} ${values?.latency}`
  if (key === "policy.dns.probeBatchWorst") return `worst ${values?.tag} ${values?.latency}`
  if (key === "policy.dns.probeBatchFailedSamples") return `failed ${values?.samples}`
  return key
}

describe("dnsProbeInput", () => {
  it("builds probeable udp input", () => {
    expect(isDNSServerProbeable({ type: "udp", server: "1.1.1.1" })).toBe(true)
    expect(dnsProbeInput({ tag: "cf", type: "udp", server: "1.1.1.1", server_port: 53 }))
      .toEqual({ tag: "cf", type: "udp", server: "1.1.1.1", server_port: 53 })
  })

  it("supports legacy address and rejects local", () => {
    expect(dnsProbeInput({ address: "https://dns.google/dns-query" }))
      .toEqual({ address: "https://dns.google/dns-query" })
    expect(isDNSServerProbeable({ type: "local" })).toBe(false)
    expect(dnsProbeInput({ type: "local" })).toBeNull()
  })
})

describe("DNS probe batch summary", () => {
  it("summarizes latency and failed samples", () => {
    expect(summarizeDNSProbeResults(sample)).toEqual({
      total: 3,
      success: 2,
      failed: 1,
      avgLatencyMs: 21,
      bestTag: "cf",
      bestLatencyMs: 12,
      worstTag: "google",
      worstLatencyMs: 30,
      failedSamples: [{ tag: "bad", error: "timeout" }],
    })
    const message = formatDNSProbeBatchMessage(summarizeDNSProbeResults(sample), t)
    expect(message).toContain("2/3 ok, 1 failed")
    expect(message).toContain("best cf 12ms")
    expect(message).toContain("failed bad: timeout")
    expect(dnsProbeBatchToastTone(summarizeDNSProbeResults(sample))).toBe("warning")
    expect(dnsProbeBatchToastTone(summarizeDNSProbeResults([
      { tag: "x", success: false, error: "boom" },
    ]))).toBe("error")
    expect(formatDNSProbeBatchMessage(summarizeDNSProbeResults([]), t)).toBe("done")
  })

  it("maps batch results by input order and tag", () => {
    const mapped = mapDNSProbeBatchResults(
      [
        { tag: "cf", type: "udp", server: "1.1.1.1" },
        { type: "udp", server: "8.8.8.8" },
      ],
      [
        { tag: "cf", success: true, latency_ms: 10 },
        { tag: "", success: false, error: "timeout" },
      ],
      (input, index) => input.tag || `idx:${index}`,
    )
    expect(mapped.cf.latency_ms).toBe(10)
    expect(mapped["idx:1"].error).toBe("timeout")
  })

  it("builds batch failure clipboard text", () => {
    const summary = summarizeDNSProbeResults([
      { tag: "a", success: false, error: "timeout", error_code: "timeout" },
      { tag: "b", success: true, latency_ms: 12 },
    ])
    expect(dnsProbeBatchFailureClipboardText(summary)).toContain("tag: a")
    expect(dnsProbeBatchFailureClipboardText(summary)).toContain("error: timeout")
    expect(dnsProbeBatchFailureClipboardText(summarizeDNSProbeResults([]))).toBe("")
  })

})
