import { describe, expect, it } from "vitest"

import type { ConfigApplyEvent } from "@/lib/api/types"
import {
  buildConfigHistoryExportFilename,
  formatConfigHistoryExport,
} from "@/features/advanced/config-history-export"

const event: ConfigApplyEvent = {
  id: "event-1",
  source: "raw",
  status: "rolled_back",
  hash: "abc123",
  size: 42,
  error: "invalid route",
  applied_at: "2026-07-26T00:00:00Z",
}

describe("config history export", () => {
  it("serializes filters and records as readable JSON", () => {
    const output = formatConfigHistoryExport([event], {
      query: "  route  ",
      filter: "failed",
      exportedAt: new Date("2026-07-26T01:02:03.004Z"),
    })
    expect(JSON.parse(output)).toEqual({
      exported_at: "2026-07-26T01:02:03.004Z",
      query: "route",
      filter: "failed",
      count: 1,
      records: [event],
    })
    expect(output.endsWith("\n")).toBe(true)
  })

  it("builds a stable JSON filename", () => {
    expect(buildConfigHistoryExportFilename(new Date("2026-07-26T01:02:03.004Z")))
      .toBe("boxd-config-history-2026-07-26T01-02-03-004Z.json")
  })
})
