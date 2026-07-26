import { describe, expect, it } from "vitest"

import {
  buildConnectionExportFilename,
  compareConnections,
  formatConnectionClipboardText,
  formatConnectionExport,
  formatConnectionLine,
  sortConnections,
  type ConnectionSortKey,
} from "@/features/observability/connection-export"
import type { Connection } from "@/lib/api/types"

const sample: Connection[] = [
  { id: 1, target: "z.com:443", outbound: "proxy", rule: "r1", upload: 10, download: 90, start: "2026-07-23T00:02:00Z" },
  { id: 2, target: "a.com:443", outbound: "direct", rule: "", upload: 50, download: 10, start: "2026-07-23T00:00:00Z" },
  { id: 3, target: "m.com:443", outbound: "proxy", rule: "r2", upload: 1, download: 1, start: "2026-07-23T00:01:00Z" },
]

describe("connection-export", () => {
  it("sorts by traffic, duration, and target", () => {
    expect(sortConnections(sample, "traffic").map((item) => item.id)).toEqual([1, 2, 3])
    expect(sortConnections(sample, "duration").map((item) => item.id)).toEqual([2, 3, 1])
    expect(sortConnections(sample, "target").map((item) => item.id)).toEqual([2, 3, 1])
    expect(sortConnections(sample, "upload").map((item) => item.id)).toEqual([2, 1, 3])
    expect(sortConnections(sample, "download").map((item) => item.id)).toEqual([1, 2, 3])
    expect(sortConnections([
      { ...sample[0], uploadRate: 1, downloadRate: 2 },
      { ...sample[1], uploadRate: 10, downloadRate: 20 },
      sample[2],
    ], "rate").map((item) => item.id)).toEqual([2, 1, 3])
    expect(sortConnections(sample, "outbound").map((item) => item.id)).toEqual([2, 3, 1])
    expect(compareConnections(
      { ...sample[0], upload: 1, download: 1 },
      { ...sample[1], upload: 1, download: 1 },
      "download",
    )).toBeGreaterThan(0)
    expect(compareConnections(
      { ...sample[0], upload: 1, download: 1 },
      { ...sample[1], upload: 1, download: 1 },
      "upload",
    )).toBeGreaterThan(0)
    expect(compareConnections(
      { ...sample[0], start: "invalid" },
      { ...sample[1], start: "invalid" },
      "duration",
    )).toBeGreaterThan(0)
    expect(compareConnections(
      { ...sample[0], target: "", outbound: "" },
      { ...sample[1], target: "", outbound: "" },
      "outbound",
    )).toBe(0)
  })

  it("compares unknown sort as traffic", () => {
    expect(compareConnections(sample[0], sample[1], "nope" as ConnectionSortKey)).toBeLessThan(0)
  })

  it("formats export payload", () => {
    expect(formatConnectionLine(sample[1])).toBe("2\ta.com:443\tdirect\t-\t-\t-\t-\t-\t-\t50\t10\t\t\t2026-07-23T00:00:00Z")
    expect(formatConnectionLine({
      ...sample[1],
      network: "tcp",
      source: "10.0.0.2:1",
      inbound: "mixed-in",
      protocol: "tls",
      process: "/usr/bin/curl",
      rule: "r1",
      uploadRate: 12.5,
      downloadRate: 25,
    })).toBe("2\ta.com:443\tdirect\tr1\ttcp\t10.0.0.2:1\tmixed-in\ttls\t/usr/bin/curl\t50\t10\t12.5\t25\t2026-07-23T00:00:00Z")
    expect(formatConnectionExport([sample[1]])).toContain("network\tsource\tinbound\tprotocol\tprocess\tupload\tdownload\tupload_rate\tdownload_rate")
    expect(formatConnectionExport([])).toBe("")
    expect(formatConnectionLine({
      id: 8,
      target: "",
      outbound: "",
      upload: 0,
      download: 0,
      start: "",
      rule: "  ",
      network: "  ",
      source: "  ",
      inbound: "  ",
      protocol: "  ",
      process: "  ",
    })).toBe("8\t-\t-\t-\t-\t-\t-\t-\t-\t0\t0\t\t\t-")
  })

  it("builds filename", () => {
    expect(buildConnectionExportFilename(new Date("2026-07-23T01:02:03.004Z")))
      .toBe("boxd-connections-2026-07-23T01-02-03-004Z.log")
  })
})

  it("formats clipboard diagnostics", () => {
    const text = formatConnectionClipboardText({
      id: 9,
      target: "api.example.com:443",
      outbound: "proxy",
      rule: "geosite-google",
      network: "tcp",
      upload: 1,
      download: 2,
      start: "2026-07-24T00:00:00Z",
    })
    expect(text).toContain("id: 9")
    expect(text).toContain("target: api.example.com:443")
    expect(text).toContain("outbound: proxy")
    expect(text).toContain("rule: geosite-google")
    expect(formatConnectionClipboardText({ ...sample[0], uploadRate: 3, downloadRate: 4 })).toContain("upload_rate: 3")
  })

  it("omits blank optional clipboard fields", () => {
    expect(formatConnectionClipboardText({
      id: 10,
      target: "  ",
      outbound: "  ",
      upload: 0,
      download: 0,
      start: "  ",
    })).toBe(["id: 10", "upload: 0", "download: 0"].join("\n"))
  })
