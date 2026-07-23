import { describe, expect, it } from "vitest"

import { matchesDNSRule, matchesDNSServer } from "@/features/policy/dns-filter"

describe("dns-filter", () => {
  it("matches servers by tag type and address", () => {
    const server = { tag: "dns-remote", type: "https", server: "1.1.1.1", server_port: 443 }
    expect(matchesDNSServer(server, "remote")).toBe(true)
    expect(matchesDNSServer(server, "https")).toBe(true)
    expect(matchesDNSServer(server, "1.1.1.1")).toBe(true)
    expect(matchesDNSServer(server, "local")).toBe(false)
    expect(matchesDNSServer(server, "")).toBe(true)
  })

  it("matches rules by action server and domain", () => {
    const rule = { action: "route", server: "dns-remote", domain: ["google.com"], type: "default" }
    expect(matchesDNSRule(rule, "google")).toBe(true)
    expect(matchesDNSRule(rule, "dns-remote")).toBe(true)
    expect(matchesDNSRule(rule, "route")).toBe(true)
    expect(matchesDNSRule(rule, "reject")).toBe(false)
  })
})
