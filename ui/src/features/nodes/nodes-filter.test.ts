import { describe, expect, it } from "vitest"

import {
  buildNodesHref,
  filterAndSortNodes,
  matchesNodeQuery,
  matchesNodeStability,
  nodeFiltersActive,
  nodeLatencyHealth,
  parseNodeSearchParams,
  pickNodeHistorySeries,
  sortNodes,
  listProblemNodes,
  summarizeNodeStability,
  stabilityBucketForHealth,
  toNodeSearchParams,
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
    expect(pickNodeHistorySeries(undefined)).toEqual([])
    expect(pickNodeHistorySeries({ custom: [{ timestamp: "x", success: true }] })).toEqual([
      { timestamp: "x", success: true },
    ])
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

  it("matches every stability bucket and empty query form", () => {
    const good = [{ timestamp: "1", success: true }, { timestamp: "2", success: true }, { timestamp: "3", success: true }, { timestamp: "4", success: true }, { timestamp: "5", success: false }]
    const failed = [{ timestamp: "1", success: false }, { timestamp: "2", success: false }]
    const goodHealth = nodeLatencyHealth(nodes[0], { "hk-01": { tcp: good } })
    const failedHealth = nodeLatencyHealth(nodes[1], { "us-edge": { tcp: failed } })
    expect(goodHealth.tone).toBe("good")
    expect(failedHealth.tone).toBe("failed")
    expect(matchesNodeStability(goodHealth, "stable")).toBe(true)
    expect(matchesNodeStability(failedHealth, "failed")).toBe(true)
    expect(matchesNodeStability(failedHealth, "unknown")).toBe(false)
    expect(matchesNodeStability(failedHealth, "")).toBe(true)
    expect(matchesNodeQuery(nodes[0], "  ")).toBe(true)
    expect(matchesNodeQuery({ tag: "bare", type: "vless", source: "import" }, "example")).toBe(false)
  })

  it("sorts by stability and latency", () => {
    expect(sortNodes(nodes, "stability", history).map((node) => node.tag)).toEqual(["hk-01", "jp-core", "us-edge"])
    expect(sortNodes(nodes, "latency", history).map((node) => node.tag)).toEqual(["hk-01", "jp-core", "us-edge"])
    expect(sortNodes(nodes, "name", history).map((node) => node.tag)).toEqual(["hk-01", "jp-core", "us-edge"])
  })

  it("sorts stability and latency ties with missing measurements", () => {
    const tieNodes: Outbound[] = [
      { tag: "z-node", type: "vless", server: "z", port: 443, source: "import" },
      { tag: "a-node", type: "vless", server: "a", port: 443, source: "import" },
      { tag: "missing", type: "vless", server: "m", port: 443, source: "import" },
    ]
    const tieHistory = {
      "z-node": { tcp: [{ timestamp: "1", success: true, latency_ms: 30 }] },
      "a-node": { tcp: [{ timestamp: "1", success: true, latency_ms: 30 }] },
    }
    expect(sortNodes(tieNodes, "stability", tieHistory).map((node) => node.tag)).toEqual([
      "a-node", "z-node", "missing",
    ])
    expect(sortNodes(tieNodes, "latency", tieHistory).map((node) => node.tag)).toEqual([
      "a-node", "z-node", "missing",
    ])
    expect(sortNodes([
      tieNodes[2], tieNodes[0],
    ], "latency", tieHistory).map((node) => node.tag)).toEqual(["z-node", "missing"])
  })

  it("parses and builds node list deep-link query strings", () => {
    expect(parseNodeSearchParams(new URLSearchParams("q=hk&stability=stable&sort=latency"))).toEqual({
      query: "hk",
      stability: "stable",
      sort: "latency",
    })
    expect(parseNodeSearchParams(new URLSearchParams("stability=nope&sort=bogus"))).toEqual({
      query: undefined,
      stability: undefined,
      sort: undefined,
    })
    expect(buildNodesHref({ query: "hk", sort: "stability" })).toBe("/nodes?q=hk&sort=stability")
    expect(buildNodesHref({ sort: "name" })).toBe("/nodes")
    expect(toNodeSearchParams({ stability: "failed" }).get("stability")).toBe("failed")
    expect(parseNodeSearchParams({ get: () => "  " })).toEqual({
      query: undefined, stability: undefined, sort: undefined,
    })
    expect(toNodeSearchParams({ query: "  ", stability: "unknown", sort: "name" }).toString())
      .toBe("stability=unknown")
    expect(buildNodesHref()).toBe("/nodes")
  })

  it("summarizes stability buckets for the current search query", () => {
    expect(summarizeNodeStability(nodes, history)).toEqual({
      total: 3,
      stable: 1,
      fair: 1,
      unstable: 1,
      failed: 0,
      unknown: 0,
    })
    expect(summarizeNodeStability(nodes, history, "hk")).toEqual({
      total: 1,
      stable: 1,
      fair: 0,
      unstable: 0,
      failed: 0,
      unknown: 0,
    })
    expect(stabilityBucketForHealth(nodeLatencyHealth(nodes[0], undefined))).toBe("unknown")
    expect(stabilityBucketForHealth(nodeLatencyHealth(nodes[1], {
      "us-edge": { tcp: [{ timestamp: "1", success: false }] },
    }))).toBe("failed")
  })

  it("lists problem nodes with failed first and limit", () => {
    const nodes = [
      { tag: "ok", type: "vless", server: "a", port: 1, source: "import" as const, raw: {} },
      { tag: "unstable", type: "vless", server: "b", port: 1, source: "import" as const, raw: {} },
      { tag: "failed", type: "vmess", server: "c", port: 1, source: "import" as const, raw: {} },
      { tag: "failed-b", type: "trojan", server: "d", port: 1, source: "import" as const, raw: {} },
    ]
    const history = {
      ok: { tcp: [
        { timestamp: "2026-07-23T00:00:00Z", success: true, latency_ms: 20 },
        { timestamp: "2026-07-23T00:01:00Z", success: true, latency_ms: 22 },
      ]},
      unstable: { tcp: [
        { timestamp: "2026-07-23T00:00:00Z", success: true, latency_ms: 30 },
        { timestamp: "2026-07-23T00:01:00Z", success: false },
        { timestamp: "2026-07-23T00:02:00Z", success: false },
        { timestamp: "2026-07-23T00:03:00Z", success: false },
      ]},
      failed: { tcp: [
        { timestamp: "2026-07-23T00:00:00Z", success: false },
        { timestamp: "2026-07-23T00:01:00Z", success: false },
      ]},
      "failed-b": { tcp: [
        { timestamp: "2026-07-23T00:00:00Z", success: false },
      ]},
    }
    expect(listProblemNodes(nodes, history, 2).map((item) => item.tag)).toEqual(["failed", "failed-b"])
    expect(listProblemNodes(nodes, history, 3).map((item) => item.stability)).toEqual([
      "failed", "failed", "unstable",
    ])
    expect(listProblemNodes([], history, 3)).toEqual([])
    expect(listProblemNodes(nodes, history, 0)).toEqual([])
    expect(listProblemNodes(undefined, history, 3)).toEqual([])
    expect(listProblemNodes(nodes, history, -1)).toEqual([])
    expect(listProblemNodes(nodes, history, 2.9).length).toBe(2)
  })
})
