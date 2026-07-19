import { describe, expect, it } from "vitest"

import {
  changeEndpointType,
  createEndpointDraft,
  isEndpointReady,
  isEndpointsStructureValid,
  normalizeEndpoints,
  prepareEndpointObject,
  prepareEndpoints,
  summarizeEndpoint,
} from "@/features/advanced/endpoints-form-model"
import { getPolicyPath } from "@/features/policy/policy-form-model"

describe("endpoints form model", () => {
  it("normalizes non-array values to empty lists", () => {
    expect(normalizeEndpoints(undefined)).toEqual([])
    expect(normalizeEndpoints({})).toEqual([])
    expect(normalizeEndpoints("x")).toEqual([])
    expect(normalizeEndpoints([{ type: "wireguard", tag: "wg" }])).toEqual([
      { type: "wireguard", tag: "wg" },
    ])
  })

  it("accepts only arrays of objects", () => {
    expect(isEndpointsStructureValid([])).toBe(true)
    expect(isEndpointsStructureValid([{ type: "tailscale", tag: "ts" }])).toBe(true)
    expect(isEndpointsStructureValid([1])).toBe(false)
    expect(isEndpointsStructureValid({})).toBe(false)
    expect(isEndpointsStructureValid(null)).toBe(false)
  })

  it("creates and switches endpoint drafts by type", () => {
    expect(createEndpointDraft("wireguard")).toMatchObject({
      type: "wireguard",
      tag: "",
      private_key: "",
    })
    expect(createEndpointDraft("tailscale")).toMatchObject({ type: "tailscale", tag: "" })
    const switched = changeEndpointType({ type: "wireguard", tag: "home", private_key: "k" }, "tailscale")
    expect(switched).toMatchObject({ type: "tailscale", tag: "home" })
    expect(switched.private_key).toBeUndefined()
  })

  it("validates required wireguard fields and always accepts ready tailscale tags", () => {
    expect(isEndpointReady({ type: "wireguard", tag: "wg" })).toBe(false)
    expect(isEndpointReady({
      type: "wireguard",
      tag: "wg",
      address: ["10.0.0.2/32"],
      private_key: "private",
    })).toBe(true)
    expect(isEndpointReady({ type: "tailscale", tag: "ts" })).toBe(true)
    expect(isEndpointReady({ type: "tailscale", tag: "" })).toBe(false)
  })

  it("prunes type-specific invisible fields", () => {
    const prepared = prepareEndpointObject({
      type: "tailscale",
      tag: "ts",
      private_key: "should-drop",
      peers: [{ public_key: "x" }],
      hostname: "node-a",
      system_interface: false,
      system_interface_name: "tailscale0",
    })
    expect(getPolicyPath(prepared, "private_key")).toBeUndefined()
    expect(getPolicyPath(prepared, "peers")).toBeUndefined()
    expect(getPolicyPath(prepared, "hostname")).toBe("node-a")
    expect(getPolicyPath(prepared, "system_interface_name")).toBeUndefined()
  })

  it("summarizes endpoints and prepares lists", () => {
    expect(summarizeEndpoint({
      type: "wireguard",
      tag: "wg",
      address: ["10.0.0.2/32"],
      peers: [{}, {}],
    })).toEqual({ type: "wireguard", detail: "10.0.0.2/32", meta: 2 })
    expect(summarizeEndpoint({ type: "tailscale", tag: "ts", hostname: "box" }))
      .toEqual({ type: "tailscale", detail: "box", meta: 0 })
    expect(prepareEndpoints([{ type: "tailscale", tag: "ts", private_key: "x" }])).toEqual([
      { type: "tailscale", tag: "ts" },
    ])
  })
})
