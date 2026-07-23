import { describe, expect, it } from "vitest"

import {
  collectInboundBaseInvalid,
  collectOutboundBaseInvalid,
  inboundBaseFieldValid,
  outboundBaseFieldValid,
  portValid,
  requiredTextValid,
} from "@/features/proxy/proxy-base-validation"

describe("proxy-base-validation", () => {
  it("validates required text and ports", () => {
    expect(requiredTextValid("  ")).toBe(false)
    expect(requiredTextValid("tag")).toBe(true)
    expect(portValid(443)).toBe(true)
    expect(portValid(0)).toBe(false)
    expect(portValid(70000)).toBe(false)
    expect(portValid("abc")).toBe(false)
  })

  it("validates outbound base fields by type", () => {
    expect(collectOutboundBaseInvalid({ type: "vless" })).toEqual(["tag", "server", "server_port"])
    expect(outboundBaseFieldValid({ type: "direct", tag: "d" }, "server")).toBe(true)
    expect(collectOutboundBaseInvalid({ type: "vless", tag: "hk", server: "a.com", server_port: 443 })).toEqual([])
  })

  it("validates inbound base fields and skips listen_port for tun", () => {
    expect(collectInboundBaseInvalid({ type: "mixed" })).toEqual(["tag", "listen_port"])
    expect(collectInboundBaseInvalid({ type: "tun" })).toEqual(["tag"])
    expect(inboundBaseFieldValid({ type: "mixed", tag: "in", listen_port: 1080 }, "listen_port")).toBe(true)
    expect(inboundBaseFieldValid({ type: "tun", tag: "tun0" }, "listen_port")).toBe(true)
  })
})
