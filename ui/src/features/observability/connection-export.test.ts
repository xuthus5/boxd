import { describe, expect, it } from "vitest"

import {
  buildConnectionExportFilename,
  compareConnections,
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
    expect(sortConnections(sample, "outbound").map((item) => item.id)).toEqual([2, 3, 1])
  })

  it("compares unknown sort as traffic", () => {
    expect(compareConnections(sample[0], sample[1], "nope" as ConnectionSortKey)).toBeLessThan(0)
  })

  it("formats export payload", () => {
    expect(formatConnectionLine(sample[1])).toBe("2\ta.com:443\tdirect\t-\t-\t-\t-\t-\t-\t50\t10\t2026-07-23T00:00:00Z")
    expect(formatConnectionLine({
      ...sample[1],
      network: "tcp",
      source: "10.0.0.2:1",
      inbound: "mixed-in",
      protocol: "tls",
      process: "/usr/bin/curl",
      rule: "r1",
    })).toBe("2\ta.com:443\tdirect\tr1\ttcp\t10.0.0.2:1\tmixed-in\ttls\t/usr/bin/curl\t50\t10\t2026-07-23T00:00:00Z")
    expect(formatConnectionExport([sample[1]])).toContain("network\tsource\tinbound")
    expect(formatConnectionExport([])).toBe("")
  })

  it("builds filename", () => {
    expect(buildConnectionExportFilename(new Date("2026-07-23T01:02:03.004Z")))
      .toBe("boxd-connections-2026-07-23T01-02-03-004Z.log")
  })
})
