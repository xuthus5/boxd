import { describe, expect, it } from "vitest"

import { buildNodesHref } from "@/features/nodes/nodes-filter"
import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import { buildRouteHref } from "@/features/policy/route-rule-filter"
import { buildSubscriptionsHref } from "@/features/subscriptions/subscription-list"

describe("dashboard ops shortcut targets", () => {
  it("deep-links to common observability and config surfaces", () => {
    expect(buildConnectionsHref()).toBe("/observability/connections")
    expect(buildLogsHref()).toBe("/observability/logs")
    expect(buildLogsHref({ preset: "errors" })).toBe("/observability/logs?preset=errors")
    expect(buildLogsHref({ preset: "reject" })).toBe("/observability/logs?preset=reject")
    expect(buildSubscriptionsHref({ status: "error" })).toBe("/subscriptions?status=error")
    expect(buildNodesHref({ stability: "unstable" })).toBe("/nodes?stability=unstable")
    expect(buildRouteHref({ action: "reject" })).toBe("/policy/route?action=reject")
  })
})
