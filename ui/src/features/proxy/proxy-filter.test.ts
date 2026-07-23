import { describe, expect, it } from "vitest"

import { matchesProxyItem } from "@/features/proxy/proxy-filter"

describe("matchesProxyItem", () => {
  it("matches tag type listen server and transport", () => {
    const mixed = { tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 }
    const node = { tag: "hk", type: "vless", server: "a.example.com", server_port: 443, transport: { type: "ws" } }
    expect(matchesProxyItem(mixed, "mixed")).toBe(true)
    expect(matchesProxyItem(mixed, "1080")).toBe(true)
    expect(matchesProxyItem(node, "ws")).toBe(true)
    expect(matchesProxyItem(node, "a.example")).toBe(true)
    expect(matchesProxyItem(node, "trojan")).toBe(false)
  })

  it("returns true for empty query", () => {
    expect(matchesProxyItem({ tag: "x", type: "direct" }, "")).toBe(true)
  })
})
