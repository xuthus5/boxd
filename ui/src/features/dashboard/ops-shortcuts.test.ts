import { describe, expect, it } from "vitest"

import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"

describe("dashboard ops shortcut targets", () => {
  it("deep-links to common observability and config surfaces", () => {
    expect(buildConnectionsHref()).toBe("/observability/connections")
    expect(buildLogsHref()).toBe("/observability/logs")
    expect(buildLogsHref({ preset: "errors" })).toBe("/observability/logs?preset=errors")
  })
})
