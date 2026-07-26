import { describe, expect, it } from "vitest"

import {
  changeServiceType,
  createServiceDraft,
  applyServicesConfig,
  inferServiceType,
  isServiceReady,
  isServicesStructureValid,
  normalizeServices,
  prepareServiceObject,
  prepareServices,
  serviceTypes,
  summarizeService,
} from "@/features/advanced/services-form-model"

describe("services form model", () => {
  it("supports every sing-box 1.13 service type", () => {
    expect(serviceTypes).toEqual(["ccm", "derp", "ocm", "resolved", "ssm-api"])
    for (const type of serviceTypes) {
      expect(inferServiceType({ type })).toBe(type)
    }
    expect(inferServiceType({ type: "future" })).toBeUndefined()
  })

  it("creates useful drafts for resolved and listener services", () => {
    expect(createServiceDraft("resolved")).toEqual({
      type: "resolved",
      tag: "",
      listen: "127.0.0.53",
      listen_port: 53,
    })
    expect(createServiceDraft("ssm-api")).toEqual({
      type: "ssm-api",
      tag: "",
      listen: "",
      listen_port: 0,
    })
  })

  it("accepts service object arrays and rejects malformed roots", () => {
    expect(isServicesStructureValid([{ type: "resolved" }, { type: "ccm", tag: "api" }])).toBe(true)
    expect(isServicesStructureValid([])).toBe(true)
    expect(isServicesStructureValid({ type: "resolved" })).toBe(false)
    expect(isServicesStructureValid([{ tag: "missing type" }])).toBe(false)
    expect(isServicesStructureValid([{ type: 42 }])).toBe(false)
    expect(normalizeServices(undefined)).toEqual([])
    expect(normalizeServices([{ type: "resolved" }])).toEqual([
      { type: "resolved", listen: "127.0.0.53", listen_port: 53 },
    ])
  })

  it("requires listener settings for network services", () => {
    expect(isServiceReady({ type: "resolved" })).toBe(true)
    expect(isServiceReady({ type: "ccm", listen: "127.0.0.1", listen_port: 8080 })).toBe(true)
    expect(isServiceReady({ type: "ocm", listen: "127.0.0.1", listen_port: 0 })).toBe(false)
    expect(isServiceReady({ type: "derp", listen: "127.0.0.1", listen_port: 443 })).toBe(false)
    expect(isServiceReady({ type: "ssm-api", listen: "127.0.0.1", listen_port: 8080 })).toBe(false)
  })

  it("checks type-specific required fields", () => {
    expect(isServiceReady({ type: "derp", listen: "::", listen_port: 443, config_path: "derper.key" })).toBe(true)
    expect(isServiceReady({ type: "ssm-api", listen: "127.0.0.1", listen_port: 8080, servers: { "/": "ss-in" } })).toBe(true)
    expect(isServiceReady({ type: "derp", listen: "::", listen_port: 443, config_path: " " })).toBe(false)
    expect(isServiceReady({ type: "ssm-api", listen: "127.0.0.1", listen_port: 8080, servers: {} })).toBe(false)
  })

  it("keeps resolved defaults when normalizing a partial object", () => {
    expect(prepareServiceObject({ type: "resolved" })).toEqual({
      type: "resolved",
      listen: "127.0.0.53",
      listen_port: 53,
    })
  })

  it("removes hidden type fields but preserves unknown extensions", () => {
    const prepared = prepareServiceObject({
      type: "resolved",
      listen: "127.0.0.53",
      listen_port: 53,
      config_path: "stale.derp",
      future_option: { enabled: true },
      tls: { enabled: true },
    })
    expect(prepared).toEqual({
      type: "resolved",
      listen: "127.0.0.53",
      listen_port: 53,
      future_option: { enabled: true },
    })
  })

  it("preserves detour values across service families", () => {
    expect(prepareServiceObject({
      type: "ccm",
      listen: "127.0.0.1",
      listen_port: 8080,
      detour: "proxy-out",
    })).toHaveProperty("detour", "proxy-out")
    expect(prepareServiceObject({
      type: "resolved",
      detour: "dns-in",
    })).toHaveProperty("detour", "dns-in")
  })

  it("changes type without leaking incompatible fields", () => {
    expect(changeServiceType({ type: "derp", tag: "edge", config_path: "derper.key" }, "resolved")).toEqual({
      type: "resolved",
      tag: "edge",
      listen: "127.0.0.53",
      listen_port: 53,
    })
    expect(changeServiceType({ type: "resolved", tag: "dns" }, "ssm-api")).toEqual({
      type: "ssm-api",
      tag: "dns",
      listen: "",
      listen_port: 0,
    })
  })

  it("prepares and maps service arrays immutably", () => {
    const services = [{ type: "resolved", tag: "dns" }, { type: "ccm", listen: "127.0.0.1", listen_port: 8080 }]
    expect(prepareServices(services)).toEqual([
      { type: "resolved", tag: "dns", listen: "127.0.0.53", listen_port: 53 },
      { type: "ccm", listen: "127.0.0.1", listen_port: 8080 },
    ])
    expect(services[0]).toEqual({ type: "resolved", tag: "dns" })
  })

  it("summarizes listener and SSM API services", () => {
    expect(summarizeService({ type: "resolved" })).toEqual({
      type: "resolved",
      detail: "127.0.0.53:53",
      meta: 0,
    })
    expect(summarizeService({
      type: "ssm-api",
      listen: "127.0.0.1",
      listen_port: 8080,
      servers: { "/": "ss-in", "/backup": "ss-backup" },
    })).toEqual({ type: "ssm-api", detail: "127.0.0.1:8080", meta: 2 })
  })

  it("removes an empty services section without mutating the source config", () => {
    const config = { log: { level: "info" }, services: [{ type: "resolved" }] }
    expect(applyServicesConfig(config, [])).toEqual({ log: { level: "info" } })
    expect(config).toHaveProperty("services")
    expect(applyServicesConfig(config, [{ type: "resolved" }])).toEqual({
      log: { level: "info" },
      services: [{ type: "resolved", listen: "127.0.0.53", listen_port: 53 }],
    })
  })
})
