import { describe, expect, it } from "vitest"

import {
  experimentalFields,
  isExperimentalStructureValid,
  normalizeExperimentalObject,
  prepareExperimentalObject,
} from "@/features/advanced/experimental-form-model"
import { getPolicyPath } from "@/features/policy/policy-form-model"

describe("experimental form model", () => {
  it("normalizes non-object values to empty objects", () => {
    expect(normalizeExperimentalObject(undefined)).toEqual({})
    expect(normalizeExperimentalObject([])).toEqual({})
    expect(normalizeExperimentalObject("x")).toEqual({})
    expect(normalizeExperimentalObject({ cache_file: { enabled: true } })).toEqual({
      cache_file: { enabled: true },
    })
  })

  it("accepts only object structures", () => {
    expect(isExperimentalStructureValid({})).toBe(true)
    expect(isExperimentalStructureValid({ clash_api: {} })).toBe(true)
    expect(isExperimentalStructureValid([])).toBe(false)
    expect(isExperimentalStructureValid(null)).toBe(false)
    expect(isExperimentalStructureValid("x")).toBe(false)
  })

  it("prunes nested fields when parent switches are off", () => {
    const prepared = prepareExperimentalObject({
      cache_file: {
        enabled: false,
        path: "/tmp/cache.db",
        store_fakeip: true,
        store_rdrc: true,
        rdrc_timeout: "7d",
      },
      v2ray_api: {
        listen: "127.0.0.1:8080",
        stats: {
          enabled: false,
          inbounds: ["mixed-in"],
          outbounds: ["proxy"],
          users: ["alice"],
        },
      },
    })
    expect(getPolicyPath(prepared, "cache_file.enabled")).toBe(false)
    expect(getPolicyPath(prepared, "cache_file.path")).toBeUndefined()
    expect(getPolicyPath(prepared, "cache_file.store_fakeip")).toBeUndefined()
    expect(getPolicyPath(prepared, "cache_file.rdrc_timeout")).toBe("7d")
    expect(getPolicyPath(prepared, "v2ray_api.stats.enabled")).toBe(false)
    expect(getPolicyPath(prepared, "v2ray_api.stats.inbounds")).toBeUndefined()
    expect(getPolicyPath(prepared, "v2ray_api.listen")).toBe("127.0.0.1:8080")
  })

  it("keeps dependent fields when switches are enabled", () => {
    const prepared = prepareExperimentalObject({
      cache_file: {
        enabled: true,
        path: "/var/lib/boxd/cache.db",
        store_rdrc: true,
        rdrc_timeout: "1h",
      },
      v2ray_api: {
        stats: {
          enabled: true,
          outbounds: ["direct"],
        },
      },
    })
    expect(getPolicyPath(prepared, "cache_file.path")).toBe("/var/lib/boxd/cache.db")
    expect(getPolicyPath(prepared, "cache_file.rdrc_timeout")).toBe("1h")
    expect(getPolicyPath(prepared, "v2ray_api.stats.outbounds")).toEqual(["direct"])
    expect(experimentalFields.some((field) => field.path === "clash_api.external_controller")).toBe(true)
  })
})
