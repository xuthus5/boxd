import { afterEach, describe, expect, it, vi } from "vitest"

import {
  cellValue,
  formatDuration,
  nodeHref,
  ruleRouteHref,
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
    expect(ruleRouteHref("geosite-google")).toBe("/policy/route?q=geosite-google")
    expect(ruleRouteHref("—")).toBe("")
    expect(targetLogsHref(undefined)).toBe("")
    expect(targetLogsHref("  ")).toBe("")
    expect(nodeHref(undefined)).toBe("")
    expect(ruleRouteHref(undefined)).toBe("")
    expect(ruleRouteHref("  ")).toBe("")
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

  it("covers every cell and title fallback", () => {
    const empty: Connection = {
      id: 2,
      target: "",
      outbound: "",
      upload: 0,
      download: 0,
      start: "invalid",
    }
    const ids = ["target", "source", "network", "inbound", "outbound", "rule", "protocol", "process"] as const
    for (const id of ids) expect(cellValue(empty, id, "2s")).toBe("—")
    expect(cellValue(empty, "download", "2s")).toBe("0 B")
    expect(cellValue(empty, "duration", "2s")).toBe("2s")
    expect(cellValue(empty, "actions", "2s")).toBe("—")
    expect(titleFor(empty, "target")).toBeUndefined()
    expect(titleFor(empty, "source")).toBeUndefined()
    expect(titleFor(empty, "inbound")).toBeUndefined()
    expect(titleFor(empty, "rule")).toBeUndefined()
    expect(titleFor(empty, "process")).toBeUndefined()
  })

  it("formats elapsed time without going negative", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-01-01T00:00:10Z"))
    expect(formatDuration("2026-01-01T00:00:08Z")).toBe("2s")
    expect(formatDuration("2026-01-01T00:00:20Z")).toBe("0s")
    vi.useRealTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })
})
