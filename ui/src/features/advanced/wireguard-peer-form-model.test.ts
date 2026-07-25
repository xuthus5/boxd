import { describe, expect, it } from "vitest"

import {
  createWireGuardPeerDraft,
  isWireGuardPeerList,
  isWireGuardPeerListReady,
  isWireGuardPeerReady,
  prepareWireGuardPeer,
  summarizeWireGuardPeer,
  transformWireGuardPeerField,
  wireGuardPeerFields,
  wireGuardPeers,
} from "@/features/advanced/wireguard-peer-form-model"

describe("wireguard peer form model", () => {
  it("creates and normalizes peer drafts", () => {
    expect(createWireGuardPeerDraft()).toEqual({ public_key: "", allowed_ips: [] })
    expect(wireGuardPeers([{ public_key: "key" }, "invalid", null])).toEqual([{ public_key: "key" }])
    expect(wireGuardPeers(undefined)).toEqual([])
    expect(isWireGuardPeerList([])).toBe(true)
    expect(isWireGuardPeerList([{ public_key: "key" }])).toBe(true)
    expect(isWireGuardPeerList([{ public_key: "key" }, "invalid"])).toBe(false)
    expect(isWireGuardPeerList(undefined)).toBe(false)
  })

  it("requires a public key and at least one allowed IP", () => {
    expect(isWireGuardPeerReady({ public_key: "", allowed_ips: ["0.0.0.0/0"] })).toBe(false)
    expect(isWireGuardPeerReady({ public_key: "key", allowed_ips: [] })).toBe(false)
    expect(isWireGuardPeerReady({ public_key: "key", allowed_ips: "0.0.0.0/0" })).toBe(true)
    expect(isWireGuardPeerReady({ public_key: "key", allowed_ips: ["0.0.0.0/0"], port: 65536 })).toBe(false)
    expect(isWireGuardPeerReady({ public_key: "key", allowed_ips: ["0.0.0.0/0"], reserved: [1, 2] })).toBe(false)
    expect(isWireGuardPeerListReady([{ public_key: "key", allowed_ips: ["0.0.0.0/0"] }])).toBe(true)
    expect(isWireGuardPeerListReady([])).toBe(false)
    expect(isWireGuardPeerListReady([{ public_key: "key" }])).toBe(false)
  })

  it("rejects malformed optional peer values", () => {
    const peer = { public_key: "key", allowed_ips: ["0.0.0.0/0"] }
    expect(isWireGuardPeerReady({ ...peer, address: 1 })).toBe(false)
    expect(isWireGuardPeerReady({ ...peer, port: 1.5 })).toBe(false)
    expect(isWireGuardPeerReady({ ...peer, persistent_keepalive_interval: 65536 })).toBe(false)
    expect(isWireGuardPeerReady({ ...peer, reserved: "1,2,3" })).toBe(false)
    expect(isWireGuardPeerReady({ ...peer, reserved: [] })).toBe(true)
    expect(isWireGuardPeerReady({ ...peer, allowed_ips: ["", " "] })).toBe(false)
    expect(isWireGuardPeerReady({ ...peer, allowed_ips: [1] })).toBe(false)
    expect(isWireGuardPeerListReady(undefined)).toBe(false)
  })

  it("transforms bounded numeric peer fields", () => {
    const field = (path: string) => wireGuardPeerFields.find((item) => item.path === path)!
    expect(transformWireGuardPeerField({}, field("port"), "51820")).toEqual({ port: 51820 })
    expect(transformWireGuardPeerField({}, field("port"), "65536")).toBeNull()
    expect(transformWireGuardPeerField({}, field("port"), "invalid")).toBeNull()
    expect(transformWireGuardPeerField({}, field("persistent_keepalive_interval"), "25"))
      .toEqual({ persistent_keepalive_interval: 25 })
    expect(transformWireGuardPeerField({ persistent_keepalive_interval: 25 }, field("persistent_keepalive_interval"), ""))
      .toEqual({})
    expect(transformWireGuardPeerField({}, field("persistent_keepalive_interval"), "65536")).toBeNull()
    expect(transformWireGuardPeerField({}, field("reserved"), "1, 2, 3"))
      .toEqual({ reserved: [1, 2, 3] })
    expect(transformWireGuardPeerField({}, field("reserved"), "1, 2")).toBeNull()
    expect(transformWireGuardPeerField({ reserved: [1, 2, 3] }, field("reserved"), "")).toEqual({})
    expect(transformWireGuardPeerField({}, field("reserved"), "1, 2, invalid")).toBeNull()
    expect(transformWireGuardPeerField({ port: 10 }, field("port"), "")).toEqual({})
    expect(transformWireGuardPeerField({}, field("public_key"), "key")).toBeUndefined()
  })

  it("preserves unknown fields while normalizing supported list values", () => {
    expect(prepareWireGuardPeer({
      public_key: "key",
      allowed_ips: "0.0.0.0/0",
      reserved: [1, 2, 3],
      custom: { keep: true },
    })).toEqual({
      public_key: "key",
      allowed_ips: ["0.0.0.0/0"],
      reserved: [1, 2, 3],
      custom: { keep: true },
    })
    expect(prepareWireGuardPeer({ allowed_ips: "", reserved: "1, 2, 3" })).toEqual({ reserved: [1, 2, 3] })
    expect(prepareWireGuardPeer({ reserved: "" })).toEqual({})
    expect(prepareWireGuardPeer({ reserved: [] })).toEqual({})
    expect(prepareWireGuardPeer({ reserved: "1, 2" })).toEqual({ reserved: "1, 2" })
  })

  it("summarizes peer endpoint and allowed IPs", () => {
    expect(summarizeWireGuardPeer({
      address: "peer.example",
      port: 51820,
      public_key: "abcdefghijklmnop",
      allowed_ips: ["10.0.0.0/24", "fd00::/8"],
      persistent_keepalive_interval: 25,
    })).toEqual({
      endpoint: "peer.example:51820",
      publicKey: "abcdefgh…",
      allowedIPs: 2,
      keepalive: 25,
    })
    expect(summarizeWireGuardPeer({ public_key: "key", allowed_ips: "0.0.0.0/0" })).toEqual({
      endpoint: undefined,
      publicKey: "key",
      allowedIPs: 1,
      keepalive: undefined,
    })
    expect(summarizeWireGuardPeer({
      port: 51820,
      public_key: 1,
      allowed_ips: [1],
      persistent_keepalive_interval: 65536,
    })).toEqual({
      endpoint: ":51820",
      publicKey: "",
      allowedIPs: 0,
      keepalive: undefined,
    })
  })
})
