import { describe, expect, it } from "vitest"

import {
  isNTPStructureValid,
  normalizeNTPObject,
  ntpFields,
  prepareNTPObject,
  transformNTPField,
} from "@/features/advanced/ntp-form-model"
import { getPolicyPath } from "@/features/policy/policy-form-model"

describe("NTP form model", () => {
  it("normalizes missing sections and the sing-box resolver shorthand", () => {
    expect(normalizeNTPObject(undefined)).toEqual({})
    expect(normalizeNTPObject([])).toEqual({})
    expect(normalizeNTPObject({ server: "time.apple.com", domain_resolver: "dns-local" })).toEqual({
      server: "time.apple.com",
      domain_resolver: { server: "dns-local" },
    })
    expect(normalizeNTPObject({ domain_resolver: { server: "dns-local" } })).toEqual({
      domain_resolver: { server: "dns-local" },
    })
    expect(normalizeNTPObject({ domain_resolver: " " })).toEqual({})
  })

  it("accepts object sections and rejects malformed resolver objects", () => {
    expect(isNTPStructureValid({})).toBe(true)
    expect(isNTPStructureValid({ domain_resolver: "dns-local" })).toBe(true)
    expect(isNTPStructureValid({ domain_resolver: { server: "dns-local" } })).toBe(true)
    expect(isNTPStructureValid({ domain_resolver: 42 })).toBe(false)
    expect(isNTPStructureValid([])).toBe(false)
    expect(isNTPStructureValid(null)).toBe(false)
  })

  it("removes inactive NTP and resolver fields before saving", () => {
    const prepared = prepareNTPObject({
      enabled: false,
      server: "time.example.com",
      server_port: 123,
      interval: "30m",
      write_to_system: true,
      detour: "direct",
      domain_resolver: { server: "dns-local", rewrite_ttl: 60 },
    })
    expect(prepared).toEqual({ enabled: false })
  })

  it("keeps active fields and prunes resolver details when its server is empty", () => {
    const prepared = prepareNTPObject({
      enabled: true,
      server: "time.example.com",
      server_port: 123,
      interval: "30m",
      write_to_system: false,
      domain_resolver: { server: "", strategy: "prefer_ipv4" },
    })
    expect(getPolicyPath(prepared, "server")).toBe("time.example.com")
    expect(getPolicyPath(prepared, "server_port")).toBe(123)
    expect(getPolicyPath(prepared, "interval")).toBe("30m")
    expect(getPolicyPath(prepared, "domain_resolver")).toBeUndefined()
  })

  it("bounds ports, resolver TTLs, and routing marks", () => {
    const apply = (path: string, raw: string, kind: "number" | "text") => (
      transformNTPField({}, { path, label: path, kind }, raw)
    )
    expect(apply("server_port", "123", "number")).toEqual({ server_port: 123 })
    expect(apply("server_port", "65536", "number")).toBeNull()
    expect(apply("domain_resolver.rewrite_ttl", "4294967295", "number")).toEqual({
      domain_resolver: { rewrite_ttl: 4294967295 },
    })
    expect(apply("domain_resolver.rewrite_ttl", "4294967296", "number")).toBeNull()
    expect(apply("routing_mark", "0x10", "text")).toEqual({ routing_mark: "0x10" })
    expect(apply("routing_mark", "0x100000000", "text")).toBeNull()
    expect(apply("interval", "30m", "text")).toBeUndefined()
  })

  it("exposes the core NTP and dialer fields", () => {
    const paths = new Set(ntpFields.map((field) => field.path))
    for (const path of [
      "enabled",
      "server",
      "server_port",
      "interval",
      "write_to_system",
      "detour",
      "domain_resolver.server",
      "network_strategy",
    ]) expect(paths.has(path)).toBe(true)
  })
})
