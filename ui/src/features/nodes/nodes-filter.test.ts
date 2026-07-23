import { describe, expect, it } from "vitest"

import {
  filterAndSortNodes,
  matchesNodeQuery,
  matchesNodeStability,
  nodeFiltersActive,
  nodeLatencyHealth,
  pickNodeHistorySeries,
  sortNodes,
} from "@/features/nodes/nodes-filter"
import type { LatencyPoint, Outbound } from "@/lib/api/types"

const nodes: Outbound[] = [
  { tag: "hk-01", type: "vless", server: "a.example.com", port: 443, source: "subscription", source_name: "主订阅" },
  { tag: "us-edge", type: "trojan", server: "b.example.com", port: 8443, source: "import" },
  { tag: "jp-core", type: "vmess", server: "c.example.com", port: 443, source: "import" },
]

const history = {
  "hk-01": {
    tcp: [
      { timestamp: "1", success: true, latency_ms: 20 },
      { timestamp: "2", success: true, latency_ms: 30 },
      { timestamp: "3", success: true, latency_ms: 25 },
      { timestamp: "4", success: true, latency_ms: 22 },
    ] satisfies LatencyPoint[],
  },
  "us-edge": {
    tcp: [
      { timestamp: "1", success: false },
      { timestamp: "2", success: false },
      { timestamp: "3", success: true, latency_ms: 120 },
      { timestamp: "4", success: false },
    ] satisfies LatencyPoint[],
  },
  "jp-core": {
    http: [
      { timestamp: "1", success: true, latency_ms: 40 },
      { timestamp: "2", success: false },
      { timestamp: "3", success: true, latency_ms: 50 },
      { timestamp: "4", success: true, latency_ms: 45 },
    ] satisfies LatencyPoint[],
  },
}

describe("nodes-filter", () => {
  it("matches tag type server and source name", () => {
    expect(nodes.filter((node) => matchesNodeQuery(node, "hk"))).toHaveLength(1)
    expect(nodes.filter((node) => matchesNodeQuery(node, "trojan"))).toHaveLength(1)
    expect(nodes.filter((node) => matchesNodeQuery(node, "主订阅"))).toHaveLength(1)
    expect(nodes.filter((node) => matchesNodeQuery(node, "8443"))).toHaveLength(1)
    expect(nodes.filter((node) => matchesNodeQuery(node, ""))).toHaveLength(3)
  })

  it("picks preferred test series and builds health", () => {
    expect(pickNodeHistorySeries(history["jp-core"]).map((point) => point.latency_ms)).toEqual([40, undefined, 50, 45])
    expect(nodeLatencyHealth(nodes[0], history).tone).toBe("excellent")
    expect(nodeLatencyHealth(nodes[1], history).tone).toBe("poor")
    expect(nodeLatencyHealth(nodes[2], history).tone).toBe("fair")
  })

  it("filters by stability buckets", () => {
    const stable = filterAndSortNodes(nodes, { stability: "stable" }, history).map((node) => node.tag)
    expect(stable).toEqual(["hk-01"])
    expect(filterAndSortNodes(nodes, { stability: "unstable" }, history).map((node) => node.tag)).toEqual(["us-edge"])
    expect(filterAndSortNodes(nodes, { stability: "fair" }, history).map((node) => node.tag)).toEqual(["jp-core"])
    expect(matchesNodeStability(nodeLatencyHealth(nodes[0], undefined), "unknown")).toBe(true)
    expect(nodeFiltersActive({ query: "hk" })).toBe(true)
    expect(nodeFiltersActive({ stability: "stable" })).toBe(true)
    expect(nodeFiltersActive({})).toBe(false)
  })

  it("sorts by stability and latency", () => {
    expect(sortNodes(nodes, "stability", history).map((node) => node.tag)).toEqual(["hk-01", "jp-core", "us-edge"])
    expect(sortNodes(nodes, "latency", history).map((node) => node.tag)).toEqual(["hk-01", "jp-core", "us-edge"])
    expect(sortNodes(nodes, "name", history).map((node) => node.tag)).toEqual(["hk-01", "jp-core", "us-edge"])
  })
})
