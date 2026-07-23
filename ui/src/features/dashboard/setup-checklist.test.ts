import { describe, expect, it } from "vitest"

import {
  buildSetupSteps,
  hasClashAPI,
  hasLocalInbound,
  hasProxyOutbound,
  hasRouteRules,
  setupProgress,
} from "@/features/dashboard/setup-checklist"
import type { SingBoxConfig } from "@/lib/api/types"

const emptyConfig = {} as SingBoxConfig

describe("setup-checklist", () => {
  it("detects inbound/outbound/route/clash readiness", () => {
    expect(hasLocalInbound(emptyConfig)).toBe(false)
    expect(hasLocalInbound({ inbounds: [{ type: "mixed", tag: "mixed-in" }] } as SingBoxConfig)).toBe(true)
    expect(hasProxyOutbound({ outbounds: [{ type: "direct", tag: "direct" }] } as SingBoxConfig)).toBe(false)
    expect(hasProxyOutbound({
      outbounds: [{ type: "selector", tag: "proxy", outbounds: ["hk"] }],
    } as SingBoxConfig)).toBe(true)
    expect(hasRouteRules({ route: { rules: [{ outbound: "proxy" }] } } as SingBoxConfig)).toBe(true)
    expect(hasClashAPI({ experimental: { clash_api: { external_controller: "127.0.0.1:9090" } } } as SingBoxConfig)).toBe(true)
  })

  it("builds progress across steps", () => {
    const steps = buildSetupSteps({
      status: { running: true, uptime: "1m" },
      config: {
        inbounds: [{ type: "tun", tag: "tun-in" }],
        outbounds: [{ type: "vless", tag: "node" }],
        route: { rules: [] },
        experimental: {},
      } as SingBoxConfig,
      subscriptions: [],
    })
    const progress = setupProgress(steps)
    expect(progress.done).toBe(3)
    expect(progress.total).toBe(6)
    expect(progress.complete).toBe(false)
    expect(steps.find((step) => step.id === "subscriptions")?.done).toBe(false)
  })
})
