import { describe, expect, it } from "vitest"

import type { Outbound } from "@/lib/api/types"

function matchesQuery(node: Outbound, query: string) {
  if (!query) return true
  const haystack = [node.tag, node.type, node.server ?? "", node.source_name ?? "", String(node.port ?? "")]
    .join(" ")
    .toLowerCase()
  return haystack.includes(query)
}

describe("nodes search filter", () => {
  const nodes: Outbound[] = [
    { tag: "hk-01", type: "vless", server: "a.example.com", port: 443, source: "subscription", source_name: "主订阅" },
    { tag: "us-edge", type: "trojan", server: "b.example.com", port: 8443, source: "import" },
  ]

  it("matches tag type server and source name", () => {
    expect(nodes.filter((node) => matchesQuery(node, "hk"))).toHaveLength(1)
    expect(nodes.filter((node) => matchesQuery(node, "trojan"))).toHaveLength(1)
    expect(nodes.filter((node) => matchesQuery(node, "主订阅"))).toHaveLength(1)
    expect(nodes.filter((node) => matchesQuery(node, "8443"))).toHaveLength(1)
  })

  it("returns all nodes for empty query", () => {
    expect(nodes.filter((node) => matchesQuery(node, ""))).toHaveLength(2)
  })
})
