import { describe, expect, it } from "vitest"

import { dnsProbeInput, isDNSServerProbeable } from "@/features/policy/dns-probe"

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
