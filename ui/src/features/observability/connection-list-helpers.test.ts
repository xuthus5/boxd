import { describe, expect, it } from "vitest"

import {
  cellValue,
  formatDuration,
  nodeHref,
  targetLogsHref,
  titleFor,
} from "@/features/observability/connection-list-helpers"
import type { Connection } from "@/lib/api/types"

const sample: Connection = {
  id: 1,
  target: "example.com:443",
  outbound: "proxy",
  rule: "geosite-google",
  network: "tcp",
  source: "127.0.0.1:1234",
  inbound: "mixed-in",
  protocol: "tls",
  process: "/usr/bin/curl",
  upload: 1024,
  download: 2048,
  start: "2026-01-01T00:00:00Z",
}

describe("connection list helpers", () => {
  it("builds deep links for logs and nodes", () => {
    expect(targetLogsHref("example.com:443")).toBe("/observability/logs?q=example.com")
    expect(targetLogsHref("")).toBe("")
    expect(nodeHref("proxy")).toBe("/nodes?q=proxy")
    expect(nodeHref("—")).toBe("")
    expect(nodeHref("  ")).toBe("")
  })

  it("formats duration and cell values", () => {
    expect(formatDuration("invalid")).toBe("—")
    expect(cellValue(sample, "target", "3s")).toBe("example.com:443")
    expect(cellValue(sample, "outbound", "3s")).toBe("proxy")
    expect(cellValue(sample, "upload", "3s")).toBe("1.00 KB")
    expect(cellValue(sample, "duration", "3s")).toBe("3s")
    expect(titleFor(sample, "target")).toBe("example.com:443")
    expect(titleFor(sample, "network")).toBeUndefined()
  })
})
